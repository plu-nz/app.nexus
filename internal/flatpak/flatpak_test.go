package flatpak

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// fake answers Output calls from a table keyed by the joined arguments and
// streams canned lines.
type fake struct {
	outputs map[string]string
	fail    map[string]error
	lines   []string
	err     error
	calls   []string
}

func (f *fake) flatpak() *Flatpak {
	return &Flatpak{
		Output: func(_ context.Context, args ...string) ([]byte, error) {
			key := strings.Join(args, " ")
			f.calls = append(f.calls, key)
			if err := f.fail[key]; err != nil {
				return nil, err
			}
			return []byte(f.outputs[key]), nil
		},
		Stream: func(_ context.Context, line func(string), args ...string) error {
			f.calls = append(f.calls, strings.Join(args, " "))
			for _, l := range f.lines {
				line(l)
			}
			return f.err
		},
	}
}

func TestInstalledPrefersUserInstalls(t *testing.T) {
	f := &fake{outputs: map[string]string{
		"list --app --columns=application,installation": "com.stremio.Stremio\tsystem\nrocks.shy.VacuumTube\tuser\ntv.kodi.Kodi\tsystem\ntv.kodi.Kodi\tuser\n\n",
	}}
	got, err := f.flatpak().Installed(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]Installation{"com.stremio.Stremio": System, "rocks.shy.VacuumTube": User, "tv.kodi.Kodi": User}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Installed = %v, want %v", got, want)
	}
}

func TestUpdates(t *testing.T) {
	f := &fake{outputs: map[string]string{"remote-ls --user --updates --app --columns=application": "tv.kodi.Kodi\n"}}
	got, err := f.flatpak().Updates(context.Background())
	if err != nil || !reflect.DeepEqual(got, map[string]bool{"tv.kodi.Kodi": true}) {
		t.Fatalf("Updates = %v, %v", got, err)
	}
}

// remoteInfo is real `flatpak remote-info` output.
const remoteInfo = `            ID: rocks.shy.VacuumTube
           Ref: app/rocks.shy.VacuumTube/x86_64/stable
          Arch: x86_64
        Branch: stable
 Download Size: 133.4 MB
Installed Size: 301.3 MB
       Runtime: org.freedesktop.Platform/x86_64/26.08
           Sdk: org.freedesktop.Sdk/x86_64/26.08

        Commit: d6c0eddc203511da48266c93bae99e7247adb8b2564a5f55a320a13edf746a78
`

func TestInfoReadsSizesAndRuntime(t *testing.T) {
	f := &fake{outputs: map[string]string{
		"remote-info --user flathub rocks.shy.VacuumTube": remoteInfo,
		"list --runtime --columns=ref":                    "org.kde.Platform/x86_64/6.11\n",
	}}
	got, err := f.flatpak().Info(context.Background(), "rocks.shy.VacuumTube")
	if err != nil {
		t.Fatal(err)
	}
	want := Info{DownloadSize: 133_400_000, InstalledSize: 301_300_000, Runtime: "org.freedesktop.Platform/x86_64/26.08", NeedsRuntime: true}
	if got != want {
		t.Fatalf("Info = %+v, want %+v", got, want)
	}
	f.outputs["list --runtime --columns=ref"] = "org.freedesktop.Platform/x86_64/26.08\n"
	if got, _ := f.flatpak().Info(context.Background(), "rocks.shy.VacuumTube"); got.NeedsRuntime {
		t.Fatal("NeedsRuntime with the runtime installed")
	}
}

// installLines is real non-interactive `flatpak install` output.
var installLines = []string{
	"Installing runtime/org.freedesktop.Platform.GL.default/x86_64/26.08",
	"Installing runtime/org.freedesktop.Platform.Locale/x86_64/26.08",
	"Installing runtime/org.freedesktop.Platform/x86_64/26.08",
	"Installing app/rocks.shy.VacuumTube/x86_64/stable",
}

func TestInstallAddsFlathubAndReportsSteps(t *testing.T) {
	f := &fake{lines: installLines}
	var steps []Step
	if err := f.flatpak().Install(context.Background(), "rocks.shy.VacuumTube", func(s Step) { steps = append(steps, s) }); err != nil {
		t.Fatal(err)
	}
	wantCalls := []string{
		"remote-add --user --if-not-exists flathub " + FlathubURL,
		"install --user --noninteractive -y flathub rocks.shy.VacuumTube",
	}
	if !reflect.DeepEqual(f.calls, wantCalls) {
		t.Fatalf("calls %v, want %v", f.calls, wantCalls)
	}
	if len(steps) != 4 || steps[3] != (Step{Action: "install", Kind: "app", ID: "rocks.shy.VacuumTube", N: 4}) || steps[0].Kind != "runtime" {
		t.Fatalf("steps %+v", steps)
	}
}

func TestInstallReturnsFlatpakError(t *testing.T) {
	f := &fake{lines: []string{"error: Nothing matches org.example.Missing in remote flathub"}, err: errors.New("exit status 1")}
	err := f.flatpak().Install(context.Background(), "org.example.Missing", nil)
	if err == nil || err.Error() != "Nothing matches org.example.Missing in remote flathub" {
		t.Fatalf("Install error = %v", err)
	}
}

func TestUpdateAndUninstallTargetTheUserInstallation(t *testing.T) {
	f := &fake{}
	fp := f.flatpak()
	if err := fp.Update(context.Background(), "tv.kodi.Kodi", nil); err != nil {
		t.Fatal(err)
	}
	if err := fp.Uninstall(context.Background(), "tv.kodi.Kodi", nil); err != nil {
		t.Fatal(err)
	}
	want := []string{"update --user --noninteractive -y tv.kodi.Kodi", "uninstall --user --noninteractive -y tv.kodi.Kodi"}
	if !reflect.DeepEqual(f.calls, want) {
		t.Fatalf("calls %v, want %v", f.calls, want)
	}
}

func TestParseStep(t *testing.T) {
	cases := map[string]Step{
		"Updating app/tv.kodi.Kodi/x86_64/stable":               {Action: "update", Kind: "app", ID: "tv.kodi.Kodi"},
		"Uninstalling app/tv.kodi.Kodi/x86_64/stable":           {Action: "uninstall", Kind: "app", ID: "tv.kodi.Kodi"},
		"Installing runtime/org.kde.Platform/x86_64/6.11":       {Action: "install", Kind: "runtime", ID: "org.kde.Platform"},
		"  Installing runtime/org.kde.Platform.Locale/x86_64/6": {Action: "install", Kind: "runtime", ID: "org.kde.Platform.Locale"},
	}
	for line, want := range cases {
		if got, ok := ParseStep(line); !ok || got != want {
			t.Errorf("ParseStep(%q) = %+v, %v", line, got, ok)
		}
	}
	for _, line := range []string{"", "Nothing to update.", "Installing", "Installing something", "Looking for matches…"} {
		if _, ok := ParseStep(line); ok {
			t.Errorf("ParseStep(%q) accepted a non-step line", line)
		}
	}
}

func TestParseSize(t *testing.T) {
	cases := map[string]int64{"133.4 MB": 133_400_000, "1.2 GB": 1_200_000_000, "912 bytes": 912, "5,5 kB": 5500, "junk": 0}
	for in, want := range cases {
		if got := parseSize(in); got != want {
			t.Errorf("parseSize(%q) = %d, want %d", in, got, want)
		}
	}
}

// fakeCommand puts a flatpak script on PATH for the real runners.
func fakeCommand(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "flatpak"), []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestStreamSplitsRedrawnLinesAndForcesEnglish(t *testing.T) {
	fakeCommand(t, `printf 'locale=%s\nInstalling app/a/x86_64/stable\rInstalling app/b/x86_64/stable\n' "$LC_ALL"`)
	var got []string
	if err := stream(context.Background(), func(l string) { got = append(got, l) }); err != nil {
		t.Fatal(err)
	}
	want := []string{"locale=C.UTF-8", "Installing app/a/x86_64/stable", "Installing app/b/x86_64/stable"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("lines %q, want %q", got, want)
	}
}

func TestOutputReturnsTheErrorLine(t *testing.T) {
	fakeCommand(t, `echo "error: No remote refs found for 'flathub'" >&2; exit 1`)
	_, err := output(context.Background(), "remote-ls")
	if err == nil || err.Error() != "No remote refs found for 'flathub'" {
		t.Fatalf("output error = %v", err)
	}
}

func TestCancelStopsTheCommand(t *testing.T) {
	fakeCommand(t, `[ "$1" = remote-add ] && exit 0; trap 'echo stopped; exit 143' TERM; echo Installing app/a/x86_64/stable; while :; do sleep 0.05; done`)
	ctx, cancel := context.WithCancel(context.Background())
	fp := &Flatpak{Output: output, Stream: stream}
	var steps int
	start := time.Now()
	err := fp.Install(ctx, "a", func(Step) {
		steps++
		cancel()
	})
	if !errors.Is(err, context.Canceled) || steps != 1 {
		t.Fatalf("Install = %v after %d steps", err, steps)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("cancel waited for the kill delay instead of SIGTERM")
	}
}
