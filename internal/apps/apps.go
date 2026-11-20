// Package apps knows the TV apps Nexus can open: which are installed, how
// to start them, and their icons. The catalog is fixed: nothing outside it
// can be launched or installed.
package apps

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Source says how a catalog app gets installed.
type Source string

const (
	// Flathub apps install from Flathub through Nexus.
	Flathub Source = "flathub"
	// External apps have their own installer; Website says where it is.
	External Source = "external"
)

// Categories group the catalog in the store.
const (
	Video = "video"
	Games = "games"
)

// App is a catalog entry. Command is an executable name, or "flatpak:<id>"
// for a Flatpak.
type App struct {
	// ID is stable across versions; the UI keys descriptions on it.
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Command  string   `json:"command"`
	Args     []string `json:"-"`
	Category string   `json:"category"`
	Source   Source   `json:"source"`
	Website  string   `json:"website,omitempty"`
	// DesktopID names the app's desktop entry, which gives its icon.
	DesktopID string `json:"-"`
	// Class is the window class (Wayland app ID) of the app's windows, or
	// empty when it is unknown; such apps always start a new instance.
	Class string `json:"-"`
	// Icon is a data: URL, or empty when no icon was found.
	Icon string `json:"icon"`
	// Open reports whether the app has a window right now.
	Open bool `json:"open"`
}

// FlatpakID returns the Flatpak application ID, or "" for other apps.
func (a App) FlatpakID() string {
	id, _ := strings.CutPrefix(a.Command, "flatpak:")
	if id == a.Command {
		return ""
	}
	return id
}

// Catalog is the list of apps Nexus offers, in display order. Every app
// works with the arrow keys, Enter, and Back, and fills the screen.
var Catalog = []App{
	{ID: "lounge", Name: "Lounge", Command: "lounge", Category: Video, Source: External, Website: "https://lounge.plu.nz/", DesktopID: "nz.plu.lounge", Class: "nz.plu.lounge"},
	{ID: "vacuumtube", Name: "VacuumTube", Command: "flatpak:rocks.shy.VacuumTube", Category: Video, Source: Flathub, DesktopID: "rocks.shy.VacuumTube", Class: "rocks.shy.VacuumTube"},
	{ID: "kodi", Name: "Kodi", Command: "flatpak:tv.kodi.Kodi", Category: Video, Source: Flathub, DesktopID: "tv.kodi.Kodi", Class: "Kodi"},
	{ID: "jellyfin", Name: "Jellyfin", Command: "flatpak:org.jellyfin.JellyfinDesktop", Category: Video, Source: Flathub, DesktopID: "org.jellyfin.JellyfinDesktop", Class: "org.jellyfin.JellyfinDesktop"},
	{ID: "moonlight", Name: "Moonlight", Command: "flatpak:com.moonlight_stream.Moonlight", Category: Games, Source: Flathub, DesktopID: "com.moonlight_stream.Moonlight", Class: "com.moonlight_stream.Moonlight"},
	{ID: "steam", Name: "Steam", Command: "flatpak:com.valvesoftware.Steam", Args: []string{"-gamepadui"}, Category: Games, Source: Flathub, DesktopID: "com.valvesoftware.Steam", Class: "steam"},
	{ID: "retroarch", Name: "RetroArch", Command: "flatpak:org.libretro.RetroArch", Category: Games, Source: Flathub, DesktopID: "org.libretro.RetroArch", Class: "com.libretro.RetroArch"},
}

// Find returns the catalog app with this ID.
func (l *Launcher) Find(id string) (App, bool) {
	for _, a := range l.Catalog {
		if a.ID == id {
			return a, true
		}
	}
	return App{}, false
}

// ErrUnknown is returned for a command that is not in the catalog.
var ErrUnknown = errors.New("unknown app")

// ErrMissing is returned for a catalog app that is not installed.
var ErrMissing = errors.New("app is not installed")

// Launcher finds and starts catalog apps. The zero value uses the real
// system; tests replace the hooks.
type Launcher struct {
	Catalog []App
	// Installed reports whether a command can run.
	Installed func(command string) bool
	// Start starts a process and returns once it is running.
	Start func(name string, args ...string) error
	// Icons finds icon files.
	Icons IconFinder
	// Focus brings an open window of the class to the front and reports
	// whether there was one. Nil means windows cannot be found, so every
	// launch starts the app.
	Focus func(class string) (bool, error)
	// CloseWindows asks the compositor to close all windows of catalog classes.
	CloseWindows func(classes []string) (int, error)
	// OpenWindows reports which of the classes have a window. Nil means
	// windows cannot be found, so no app reads as open.
	OpenWindows func(classes []string) (map[string]bool, error)
	// Forget drops what Installed remembers, after an install or removal.
	// Nil when Installed remembers nothing.
	Forget func()
	// Now is the clock; tests replace it.
	Now func() time.Time

	mu       sync.Mutex
	launched map[string]time.Time
}

// startupGrace is how long a launched app may take to open its window
// before another launch starts it again.
const startupGrace = 20 * time.Second

// New returns a Launcher for the real system.
func New() *Launcher {
	flatpaks := &flatpakList{}
	l := &Launcher{Catalog: Catalog, Installed: flatpaks.installed, Forget: flatpaks.forget, Start: start, Icons: SystemIcons(), Now: time.Now}
	if h := NewHyprland(); h != nil {
		l.Focus = h.Focus
		l.CloseWindows = h.CloseAll
		l.OpenWindows = h.Open
	}
	return l
}

// CloseAll asks the compositor to close every open window of catalog apps.
// It returns the number of windows that received a close request.
func (l *Launcher) CloseAll() (int, error) {
	if l.CloseWindows == nil {
		return 0, errNoClose
	}
	classes := make([]string, 0, len(l.Catalog))
	for _, a := range l.Catalog {
		if a.Class != "" {
			classes = append(classes, a.Class)
		}
	}
	closed, err := l.CloseWindows(classes)
	if err == nil {
		l.mu.Lock()
		clear(l.launched)
		l.mu.Unlock()
	}
	return closed, err
}

// errNoClose is returned when the compositor cannot close app windows.
var errNoClose = errors.New("closing apps is unavailable on this compositor")

// Close asks the compositor to close every window of the catalog app with
// this command. It returns the number of windows that received a close
// request.
func (l *Launcher) Close(command string) (int, error) {
	for _, a := range l.Catalog {
		if a.Command != command {
			continue
		}
		if l.CloseWindows == nil || a.Class == "" {
			return 0, errNoClose
		}
		closed, err := l.CloseWindows([]string{a.Class})
		if err == nil {
			l.mu.Lock()
			delete(l.launched, command)
			l.mu.Unlock()
		}
		return closed, err
	}
	return 0, ErrUnknown
}

// Available returns the installed catalog apps, with their icons.
func (l *Launcher) Available() []App {
	found := []App{}
	for _, a := range l.Catalog {
		if !l.Installed(a.Command) {
			continue
		}
		a.Icon = l.Icons.DataURL(a.DesktopID)
		found = append(found, a)
	}
	if l.OpenWindows == nil {
		return found
	}
	classes := make([]string, 0, len(found))
	for _, a := range found {
		if a.Class != "" {
			classes = append(classes, a.Class)
		}
	}
	open, err := l.OpenWindows(classes)
	if err != nil {
		log.Printf("finding open apps: %v", err)
		return found
	}
	for i := range found {
		found[i].Open = found[i].Class != "" && open[found[i].Class]
	}
	return found
}

// Launch shows the catalog app with this command: its open window when it
// has one, else a new instance. A press while the app is still starting
// does not start a second one.
func (l *Launcher) Launch(command string) error {
	for _, a := range l.Catalog {
		if a.Command != command {
			continue
		}
		if !l.Installed(command) {
			return ErrMissing
		}
		if l.Focus != nil && a.Class != "" {
			found, err := l.Focus(a.Class)
			if found {
				return err
			}
			if err != nil {
				log.Printf("finding %s windows: %v", a.Name, err)
			}
			if l.starting(command) {
				return nil
			}
		}
		var err error
		if id := a.FlatpakID(); id != "" {
			err = l.Start("flatpak", append([]string{"run", id}, a.Args...)...)
		} else {
			err = l.Start(command, a.Args...)
		}
		if err == nil {
			l.mu.Lock()
			if l.launched == nil {
				l.launched = map[string]time.Time{}
			}
			l.launched[command] = l.now()
			l.mu.Unlock()
		}
		return err
	}
	return ErrUnknown
}

// starting reports whether the command was launched recently enough that
// its window may still be on the way.
func (l *Launcher) starting(command string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	at, ok := l.launched[command]
	return ok && l.now().Sub(at) < startupGrace
}

func (l *Launcher) now() time.Time {
	if l.Now != nil {
		return l.Now()
	}
	return time.Now()
}

// flatpakList answers "is this installed?" for every catalog app from one
// `flatpak list`, which is much faster than a `flatpak info` per app. The
// answer is kept briefly, so one panel refresh lists Flatpaks once.
type flatpakList struct {
	mu  sync.Mutex
	at  time.Time
	ids map[string]bool
}

const flatpakListTTL = 2 * time.Second

func (f *flatpakList) installed(command string) bool {
	id, ok := strings.CutPrefix(command, "flatpak:")
	if !ok {
		_, err := exec.LookPath(command)
		return err == nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ids == nil || time.Since(f.at) > flatpakListTTL {
		f.ids = map[string]bool{}
		f.at = time.Now()
		out, err := exec.Command("flatpak", "list", "--app", "--columns=application").Output()
		if err != nil {
			return false
		}
		for _, line := range strings.Split(string(out), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				f.ids[line] = true
			}
		}
	}
	return f.ids[id]
}

func (f *flatpakList) forget() {
	f.mu.Lock()
	f.ids = nil
	f.mu.Unlock()
}

// start runs the app in its own session, so it outlives Nexus and does not
// get Nexus's signals, and reaps it when it exits.
func start(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
