package apps

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
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
