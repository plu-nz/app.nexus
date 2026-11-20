package apps

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func testLauncher(installed ...string) (*Launcher, *[][]string) {
	var started [][]string
	have := map[string]bool{}
	for _, c := range installed {
		have[c] = true
	}
	return &Launcher{
		Catalog:   Catalog,
		Installed: func(c string) bool { return have[c] },
		Start: func(name string, args ...string) error {
			started = append(started, append([]string{name}, args...))
			return nil
		},
	}, &started
}

func TestAvailableListsOnlyInstalledApps(t *testing.T) {
	l, _ := testLauncher("lounge")
	got := l.Available()
	if len(got) != 1 || got[0].Name != "Lounge" {
		t.Fatalf("Available = %+v, want only Lounge", got)
	}
}

func TestLaunchRunsCatalogCommands(t *testing.T) {
	l, started := testLauncher("lounge", "flatpak:rocks.shy.VacuumTube")
	if err := l.Launch("lounge"); err != nil {
		t.Fatal(err)
	}
	if err := l.Launch("flatpak:rocks.shy.VacuumTube"); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"lounge"}, {"flatpak", "run", "rocks.shy.VacuumTube"}}
	if !reflect.DeepEqual(*started, want) {
		t.Fatalf("started %v, want %v", *started, want)
	}
}

func TestLaunchRejectsOtherCommands(t *testing.T) {
	l, started := testLauncher("lounge", "sh")
	if err := l.Launch("sh"); !errors.Is(err, ErrUnknown) {
		t.Fatalf("Launch(sh) = %v, want ErrUnknown", err)
	}
	if err := l.Launch("flatpak:rocks.shy.VacuumTube"); !errors.Is(err, ErrMissing) {
		t.Fatalf("Launch(missing) = %v, want ErrMissing", err)
	}
	if len(*started) != 0 {
		t.Fatalf("started %v, want nothing", *started)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIconFromDesktopEntry(t *testing.T) {
	user, system := t.TempDir(), t.TempDir()
	write(t, filepath.Join(user, "applications", "nz.plu.lounge.desktop"),
		"[Desktop Entry]\nName=Lounge\nIcon=lounge-icon\n[Desktop Action x]\nIcon=other\n")
	write(t, filepath.Join(system, "icons", "hicolor", "128x128", "apps", "lounge-icon.png"), "png")
	write(t, filepath.Join(system, "icons", "hicolor", "scalable", "apps", "lounge-icon.svg"), "<svg/>")
	f := IconFinder{DataDirs: []string{user, system}}
	if got := f.Path("nz.plu.lounge"); !strings.HasSuffix(got, "scalable/apps/lounge-icon.svg") {
		t.Fatalf("Path = %q, want the scalable SVG", got)
	}
	if got := f.DataURL("nz.plu.lounge"); got != "data:image/svg+xml;base64,PHN2Zy8+" {
		t.Fatalf("DataURL = %q", got)
	}
}

func TestIconFallsBackToDesktopIDAndPixmaps(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "pixmaps", "rocks.shy.VacuumTube.png"), "png")
	f := IconFinder{DataDirs: []string{dir}}
	if got := f.Path("rocks.shy.VacuumTube"); !strings.HasSuffix(got, "pixmaps/rocks.shy.VacuumTube.png") {
		t.Fatalf("Path = %q, want the pixmap", got)
	}
	if got := f.DataURL("missing"); got != "" {
		t.Fatalf("DataURL(missing) = %q, want empty", got)
	}
}

func TestLaunchFocusesAnOpenWindow(t *testing.T) {
	l, started := testLauncher("lounge")
	var focused []string
	l.Focus = func(class string) (bool, error) { focused = append(focused, class); return true, nil }
	if err := l.Launch("lounge"); err != nil {
		t.Fatal(err)
	}
	if len(*started) != 0 || !reflect.DeepEqual(focused, []string{"nz.plu.lounge"}) {
		t.Fatalf("started %v, focused %v; want only a focus", *started, focused)
	}
}

func TestLaunchWaitsForAStartingApp(t *testing.T) {
	l, started := testLauncher("lounge")
	now := time.Unix(1000, 0)
	l.Now = func() time.Time { return now }
	l.Focus = func(string) (bool, error) { return false, nil }
	for range 3 {
		if err := l.Launch("lounge"); err != nil {
			t.Fatal(err)
		}
	}
	if len(*started) != 1 {
		t.Fatalf("started %v, want one launch while the window is on the way", *started)
	}
	now = now.Add(startupGrace + time.Second)
	if err := l.Launch("lounge"); err != nil {
		t.Fatal(err)
	}
	if len(*started) != 2 {
		t.Fatalf("started %v, want a new launch after the grace period", *started)
	}
}

func TestCloseAllOnlyCatalogAppsAndClearsStartupGrace(t *testing.T) {
	l, started := testLauncher("lounge")
	l.Focus = func(string) (bool, error) { return false, nil }
	var classes []string
	l.CloseWindows = func(got []string) (int, error) {
		classes = got
		return 2, nil
	}
	if err := l.Launch("lounge"); err != nil {
		t.Fatal(err)
	}
	if count, err := l.CloseAll(); count != 2 || err != nil {
		t.Fatalf("CloseAll = %d, %v", count, err)
	}
	var want []string
	for _, a := range Catalog {
		if a.Class != "" {
			want = append(want, a.Class)
		}
	}
	if !reflect.DeepEqual(classes, want) {
		t.Fatalf("classes %v, want %v", classes, want)
	}
	if err := l.Launch("lounge"); err != nil {
		t.Fatal(err)
	}
	if len(*started) != 2 {
		t.Fatalf("started %v, want relaunch after close", *started)
	}
}

func TestCloseAllNoWindow(t *testing.T) {
	l, _ := testLauncher("lounge")
	l.CloseWindows = func([]string) (int, error) { return 0, nil }
	if count, err := l.CloseAll(); count != 0 || err != nil {
		t.Fatalf("CloseAll = %d, %v; want no open window", count, err)
	}
}

const clientsJSON = `[
 {"address":"0x5a1","class":"nz.plu.lounge","focusHistoryID":4},
 {"address":"0x5a2","class":"nz.plu.lounge","focusHistoryID":1},
 {"address":"0x5a3","class":"rocks.shy.VacuumTube","focusHistoryID":0},
 {"address":"bad;rm","class":"evil","focusHistoryID":0}
]`

func TestNewestWindow(t *testing.T) {
	cases := map[string]string{"nz.plu.lounge": "0x5a2", "rocks.shy.VacuumTube": "0x5a3", "kodi": "", "evil": ""}
	for class, want := range cases {
		if got, err := newestWindow(clientsJSON, class); err != nil || got != want {
			t.Errorf("newestWindow(%s) = %q, %v; want %q", class, got, err, want)
		}
	}
}

func TestHyprlandFocus(t *testing.T) {
	var script string
	h := &Hyprland{Run: func(name string, args ...string) (string, error) {
		if args[0] == "-j" {
			return clientsJSON, nil
		}
		script = args[len(args)-1]
		return "ok", nil
	}}
	found, err := h.Focus("nz.plu.lounge")
	if !found || err != nil {
		t.Fatalf("Focus = %v, %v", found, err)
	}
	for _, part := range []string{"address:0x5a2", "hl.dsp.focus", "action = 'set'"} {
		if !strings.Contains(script, part) {
			t.Errorf("script lacks %q:\n%s", part, script)
		}
	}
	if found, _ := h.Focus("kodi"); found {
		t.Fatal("Focus found a window for a class with none")
	}
	h.Run = func(name string, args ...string) (string, error) {
		if args[0] == "-j" {
			return clientsJSON, nil
		}
		return "Error: bad script", nil
	}
	if _, err := h.Focus("nz.plu.lounge"); err == nil {
		t.Fatal("Focus hid a Lua error")
	}
}

func TestHyprlandCloseAll(t *testing.T) {
	var script string
	h := &Hyprland{Run: func(name string, args ...string) (string, error) {
		if args[0] == "-j" {
			return clientsJSON, nil
		}
		script = args[len(args)-1]
		return "ok", nil
	}}
	count, err := h.CloseAll([]string{"nz.plu.lounge", "rocks.shy.VacuumTube"})
	if count != 3 || err != nil {
		t.Fatalf("CloseAll = %d, %v", count, err)
	}
	for _, address := range []string{"0x5a1", "0x5a2", "0x5a3"} {
		if !strings.Contains(script, "address:"+address) {
			t.Errorf("CloseAll script lacks %s: %s", address, script)
		}
	}
	if strings.Contains(script, "bad;rm") || strings.Contains(script, "evil") {
		t.Fatalf("CloseAll included unknown window: %s", script)
	}
	if count, err := h.CloseAll([]string{"kodi"}); count != 0 || err != nil {
		t.Fatalf("CloseAll(kodi) = %d, %v", count, err)
	}
	h.Run = func(name string, args ...string) (string, error) {
		if args[0] == "-j" {
			return clientsJSON, nil
		}
		return "Error: bad script", nil
	}
	if _, err := h.CloseAll([]string{"nz.plu.lounge"}); err == nil {
		t.Fatal("CloseAll hid a Lua error")
	}
}

func TestLaunchPassesArguments(t *testing.T) {
	l, started := testLauncher("flatpak:com.valvesoftware.Steam")
	if err := l.Launch("flatpak:com.valvesoftware.Steam"); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"flatpak", "run", "com.valvesoftware.Steam", "-gamepadui"}}
	if !reflect.DeepEqual(*started, want) {
		t.Fatalf("started %v, want %v", *started, want)
	}
}

func TestCatalogIsConsistent(t *testing.T) {
	ids, commands := map[string]bool{}, map[string]bool{}
	for _, a := range Catalog {
		if a.ID == "" || ids[a.ID] || commands[a.Command] {
			t.Errorf("%s: missing or duplicate ID or command", a.Name)
		}
		ids[a.ID], commands[a.Command] = true, true
		if a.Category != Video && a.Category != Games {
			t.Errorf("%s: unknown category %q", a.Name, a.Category)
		}
		switch a.Source {
		case Flathub:
			if a.FlatpakID() == "" || a.DesktopID != a.FlatpakID() {
				t.Errorf("%s: a Flathub app needs a flatpak: command matching its desktop ID", a.Name)
			}
		case External:
			if a.FlatpakID() != "" || !strings.HasPrefix(a.Website, "https://") {
				t.Errorf("%s: an external app needs a command and an https website", a.Name)
			}
		default:
			t.Errorf("%s: unknown source %q", a.Name, a.Source)
		}
	}
}

func TestFind(t *testing.T) {
	l, _ := testLauncher()
	if a, ok := l.Find("kodi"); !ok || a.FlatpakID() != "tv.kodi.Kodi" {
		t.Fatalf("Find(kodi) = %+v, %v", a, ok)
	}
	if _, ok := l.Find("nope"); ok {
		t.Fatal("Find found an app outside the catalog")
	}
}

func TestFlatpakListIsReadOncePerRefresh(t *testing.T) {
	dir := t.TempDir()
	count := filepath.Join(dir, "count")
	script := "#!/bin/sh\necho x >> " + count + "\nprintf 'tv.kodi.Kodi\\nrocks.shy.VacuumTube\\n'\n"
	if err := os.WriteFile(filepath.Join(dir, "flatpak"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	calls := func() int {
		b, _ := os.ReadFile(count)
		return strings.Count(string(b), "x")
	}
	f := &flatpakList{}
	if !f.installed("flatpak:tv.kodi.Kodi") || !f.installed("flatpak:rocks.shy.VacuumTube") || f.installed("flatpak:org.libretro.RetroArch") {
		t.Fatal("wrong installed answers")
	}
	if calls() != 1 {
		t.Fatalf("flatpak ran %d times, want once", calls())
	}
	f.forget()
	f.installed("flatpak:tv.kodi.Kodi")
	if calls() != 2 {
		t.Fatalf("flatpak ran %d times after forget, want twice", calls())
	}
}

func TestAvailableMarksOpenApps(t *testing.T) {
	l, _ := testLauncher("lounge", "flatpak:rocks.shy.VacuumTube")
	var asked []string
	l.OpenWindows = func(classes []string) (map[string]bool, error) {
		asked = classes
		return map[string]bool{"rocks.shy.VacuumTube": true}, nil
	}
	got := l.Available()
	if !reflect.DeepEqual(asked, []string{"nz.plu.lounge", "rocks.shy.VacuumTube"}) {
		t.Fatalf("asked about %v", asked)
	}
	if len(got) != 2 || got[0].Open || !got[1].Open {
		t.Fatalf("Available = %+v, want only VacuumTube open", got)
	}
	l.OpenWindows = func([]string) (map[string]bool, error) { return nil, errors.New("no hyprctl") }
	for _, a := range l.Available() {
		if a.Open {
			t.Fatalf("%s reads as open after an error", a.Name)
		}
	}
}

func TestCloseOnlyThatAppAndClearsStartupGrace(t *testing.T) {
	l, started := testLauncher("lounge")
	l.Focus = func(string) (bool, error) { return false, nil }
	var classes []string
	l.CloseWindows = func(got []string) (int, error) {
		classes = got
		return 1, nil
	}
	if err := l.Launch("lounge"); err != nil {
		t.Fatal(err)
	}
	if count, err := l.Close("lounge"); count != 1 || err != nil {
		t.Fatalf("Close = %d, %v", count, err)
	}
	if !reflect.DeepEqual(classes, []string{"nz.plu.lounge"}) {
		t.Fatalf("closed classes %v, want only Lounge's", classes)
	}
	if err := l.Launch("lounge"); err != nil {
		t.Fatal(err)
	}
	if len(*started) != 2 {
		t.Fatalf("started %v, want relaunch after close", *started)
	}
}

func TestCloseRejectsOtherCommands(t *testing.T) {
	l, _ := testLauncher("lounge")
	l.CloseWindows = func([]string) (int, error) { t.Fatal("closed windows"); return 0, nil }
	if _, err := l.Close("rm -rf /"); !errors.Is(err, ErrUnknown) {
		t.Fatalf("Close = %v, want ErrUnknown", err)
	}
	l.CloseWindows = nil
	if _, err := l.Close("lounge"); err == nil {
		t.Fatal("Close succeeded without a compositor")
	}
}

func TestHyprlandOpen(t *testing.T) {
	h := &Hyprland{Run: func(string, ...string) (string, error) { return clientsJSON, nil }}
	open, err := h.Open([]string{"nz.plu.lounge", "tv.kodi.Kodi", "evil"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(open, map[string]bool{"nz.plu.lounge": true}) {
		t.Fatalf("Open = %v, want only Lounge (evil has a bad address)", open)
	}
}
