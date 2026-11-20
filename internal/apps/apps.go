// Package apps knows the TV apps Nexus can open: which are installed, how
// to start them, and their icons. The catalog is fixed; nothing outside it
// can be launched.
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

// App is a catalog entry. Command is an executable name, or "flatpak:<id>"
// for a Flatpak.
type App struct {
	Name      string `json:"name"`
	Command   string `json:"command"`
	DesktopID string `json:"-"`
	// Class is the window class (Wayland app ID) of the app's windows.
	Class string `json:"-"`
	// Icon is a data: URL, or empty when no icon was found.
	Icon string `json:"icon"`
}

// Catalog is the list of apps Nexus offers, in display order.
var Catalog = []App{
	{Name: "Lounge", Command: "lounge", DesktopID: "nz.plu.lounge", Class: "nz.plu.lounge"},
	{Name: "VacuumTube", Command: "flatpak:rocks.shy.VacuumTube", DesktopID: "rocks.shy.VacuumTube", Class: "rocks.shy.VacuumTube"},
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
	l := &Launcher{Catalog: Catalog, Installed: installed, Start: start, Icons: SystemIcons(), Now: time.Now}
	if h := NewHyprland(); h != nil {
		l.Focus = h.Focus
	}
	return l
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
		if id, ok := strings.CutPrefix(command, "flatpak:"); ok {
			err = l.Start("flatpak", "run", id)
		} else {
			err = l.Start(command)
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

func installed(command string) bool {
	if id, ok := strings.CutPrefix(command, "flatpak:"); ok {
		return exec.Command("flatpak", "info", id).Run() == nil
	}
	_, err := exec.LookPath(command)
	return err == nil
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
