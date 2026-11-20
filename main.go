// Nexus is a TV panel for Linux: it opens TV apps and holds the controls a
// couch needs (volume, output, brightness, sleep timer, power).
package main

import (
	"embed"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/plu-nz/app.nexus/internal/apps"
	"github.com/plu-nz/app.nexus/internal/audio"
	"github.com/plu-nz/app.nexus/internal/brightness"
	"github.com/plu-nz/app.nexus/internal/gtktheme"
	"github.com/plu-nz/app.nexus/internal/instance"
	"github.com/plu-nz/app.nexus/internal/layershell"
	"github.com/plu-nz/app.nexus/internal/sleeptimer"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

// panel shows and hides the Nexus window.
type panel struct {
	app       *application.App
	window    *application.WebviewWindow
	mu        sync.Mutex
	visible   bool
	hideTimer *time.Timer
	epoch     uint64
}

func (p *panel) show() {
	p.mu.Lock()
	p.epoch++
	if p.hideTimer != nil {
		p.hideTimer.Stop()
		p.hideTimer = nil
	}
	p.visible = true
	p.mu.Unlock()
	p.window.Show()
	// The UI refreshes its state and moves focus to the apps.
	p.app.Event.Emit("shown")
}

func (p *panel) hide() {
	p.mu.Lock()
	if !p.visible {
		p.mu.Unlock()
		return
	}
	p.visible = false
	p.epoch++
	epoch := p.epoch
	p.hideTimer = time.AfterFunc(150*time.Millisecond, func() {
		application.InvokeSync(func() {
			p.mu.Lock()
			defer p.mu.Unlock()
			if p.epoch == epoch && !p.visible {
				p.window.Hide()
				p.hideTimer = nil
			}
		})
	})
	p.mu.Unlock()
	// Let WebKit paint the concealed panel before unmapping the layer. Otherwise
	// Hyprland briefly reuses the last visible buffer on the next show.
	p.app.Event.Emit("hidden")
}

func (p *panel) toggle() {
	application.InvokeSync(func() {
		p.mu.Lock()
		visible := p.visible
		p.mu.Unlock()
		if visible {
			p.hide()
		} else {
			p.show()
		}
	})
}

func main() {
	// A second launch toggles the running panel instead of starting another.
	socket := instance.SocketPath()
	if instance.Toggle(socket) == nil {
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "--toggle" {
		os.Exit(1)
	}

	p := &panel{}
	themes := &ThemeService{}
	timer := sleeptimer.New(func() error {
		err := exec.Command("systemctl", "suspend").Run()
		if err != nil {
			log.Printf("sleep timer: suspend: %v", err)
		}
		return err
	})
	p.app = application.New(application.Options{
		Name:   "Nexus",
		Linux:  application.LinuxOptions{ApplicationID: "nz.plu.nexus"},
		Assets: application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Services: []application.Service{
			application.NewService(&PanelService{panel: p}),
			application.NewService(&AppsService{launcher: apps.New(), panel: p}),
			application.NewService(&TimerService{timer: timer, panel: p}),
			application.NewService(&DisplayService{control: brightness.New()}),
			application.NewService(&AudioService{mixer: audio.New()}),
			application.NewService(themes),
		},
	})
	p.window = p.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "Nexus",
		Width:     1280,
		Height:    300,
		Frameless: true,
		// Shown once it is set up as an overlay; see below.
		Hidden:      true,
		AlwaysOnTop: true,
		// The page paints its own background, so nothing flashes before it.
		BackgroundType:             application.BackgroundTypeTransparent,
		BackgroundColour:           application.NewRGBA(0, 0, 0, 0),
		DefaultContextMenuDisabled: os.Getenv("NEXUS_DEV") == "",
	})

	// The runtime becomes ready after every page load; set up once.
	var setup sync.Once
	p.window.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
		application.InvokeSync(func() {
			if themes.set(gtktheme.Read(p.window.NativeWindow())) {
				p.app.Event.Emit("theme")
			}
			setup.Do(func() {
				// NEXUS_WINDOWED keeps a normal window, for development.
				if os.Getenv("NEXUS_WINDOWED") == "" && !layershell.Init(p.window.NativeWindow(), os.Getenv("NEXUS_OUTPUT")) {
					log.Print("the compositor has no layer shell; showing a normal window")
				}
				layershell.Transparent(p.window.NativeWindow())
				watchTheme(p, themes)
				p.show()
			})
		})
	})

	stop, err := instance.Listen(socket, p.toggle)
	if err != nil {
		log.Fatalf("listening on %s: %v", socket, err)
	}
	defer stop()
	if err := p.app.Run(); err != nil {
		log.Fatal(err)
	}
}

// watchTheme re-reads the GTK theme when it changes; the UI hears about it
// through the "theme" event. Call it on the GTK main thread.
func watchTheme(p *panel, themes *ThemeService) {
	// GTK restyles after it announces a theme change, so read the colors a
	// moment later.
	gtktheme.Watch(func() {
		time.AfterFunc(200*time.Millisecond, func() {
			changed := application.InvokeSyncWithResult(func() bool {
				return themes.set(gtktheme.Read(p.window.NativeWindow()))
			})
			if changed {
				p.app.Event.Emit("theme")
			}
		})
	})
}
