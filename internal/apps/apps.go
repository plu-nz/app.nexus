// Package apps knows the TV apps Nexus can open: which are installed, how
// to start them, and their icons. The catalog is fixed; nothing outside it
// can be launched.
package apps

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// App is a catalog entry. Command is an executable name, or "flatpak:<id>"
// for a Flatpak.
type App struct {
	Name      string `json:"name"`
	Command   string `json:"command"`
	DesktopID string `json:"-"`
	// Icon is a data: URL, or empty when no icon was found.
	Icon string `json:"icon"`
}

// Catalog is the list of apps Nexus offers, in display order.
var Catalog = []App{
	{Name: "Lounge", Command: "lounge", DesktopID: "nz.plu.lounge"},
	{Name: "VacuumTube", Command: "flatpak:rocks.shy.VacuumTube", DesktopID: "rocks.shy.VacuumTube"},
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
}

// New returns a Launcher for the real system.
func New() *Launcher {
	return &Launcher{Catalog: Catalog, Installed: installed, Start: start, Icons: SystemIcons()}
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

// Launch starts the catalog app with this command.
func (l *Launcher) Launch(command string) error {
	for _, a := range l.Catalog {
		if a.Command != command {
			continue
		}
		if !l.Installed(command) {
			return ErrMissing
		}
		if id, ok := strings.CutPrefix(command, "flatpak:"); ok {
			return l.Start("flatpak", "run", id)
		}
		return l.Start(command)
	}
	return ErrUnknown
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
