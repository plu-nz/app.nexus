package main

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/plu-nz/app.nexus/internal/apps"
	"github.com/plu-nz/app.nexus/internal/audio"
	"github.com/plu-nz/app.nexus/internal/brightness"
	"github.com/plu-nz/app.nexus/internal/flatpak"
	"github.com/plu-nz/app.nexus/internal/gtktheme"
	"github.com/plu-nz/app.nexus/internal/sleeptimer"
	"github.com/plu-nz/app.nexus/internal/store"
)

// The services are the methods the UI calls by name
// (main.<Service>.<Method>). They are thin; the logic lives in internal/.

// PanelService shows and hides the panel.
type PanelService struct{ panel *panel }

// Hide hides the panel. Nexus keeps running, so the sleep timer survives.
func (s *PanelService) Hide() { s.panel.hide() }

// SetExpanded makes room for the full-screen store. The overlay already
// covers the screen; only a normal window, used without the layer shell,
// changes size.
func (s *PanelService) SetExpanded(expanded bool) { s.panel.setExpanded(expanded) }

// AppsService lists and opens the catalog apps.
type AppsService struct {
	launcher *apps.Launcher
	panel    *panel
}

// List returns the installed catalog apps.
func (s *AppsService) List() []apps.App { return s.launcher.Available() }

// CloseAll asks the compositor to close every open catalog app window.
func (s *AppsService) CloseAll() (int, error) {
	return s.launcher.CloseAll()
}

// Launch hides the panel and opens a catalog app. The panel goes first:
// while it is on screen it holds the keyboard, and the compositor would
// not move focus to the app.
func (s *AppsService) Launch(command string) error {
	s.panel.hide()
	time.Sleep(panelUnmapDelay)
	err := s.launcher.Launch(command)
	if err != nil {
		log.Printf("opening %s: %v", command, err)
	}
	return err
}

// panelUnmapDelay lets the compositor take the hidden panel off screen.
const panelUnmapDelay = 200 * time.Millisecond

// StoreService lists the catalog with each app's state and installs,
// updates, and removes Flathub apps. Operations run in the background and
// report through "store" events; a change to what is installed also sends
// "apps".
type StoreService struct {
	store    *store.Store
	launcher *apps.Launcher
}

// storeCallTimeout bounds calls that ask Flathub over the network.
const storeCallTimeout = 30 * time.Second

// Available reports whether Flatpak apps can be installed.
func (s *StoreService) Available() bool { return s.store.Available() }

// Items returns the catalog with each app's state.
func (s *StoreService) Items() ([]store.Item, error) {
	ctx, cancel := context.WithTimeout(context.Background(), storeCallTimeout)
	defer cancel()
	return s.store.Items(ctx, s.launcher.Installed)
}

// Info returns an app's download and installed sizes.
func (s *StoreService) Info(appID string) (flatpak.Info, error) {
	ctx, cancel := context.WithTimeout(context.Background(), storeCallTimeout)
	defer cancel()
	return s.store.Info(ctx, appID)
}

// Install queues an app's installation.
func (s *StoreService) Install(appID string) error { return s.start(appID, store.Install) }

// Update queues an app's update.
func (s *StoreService) Update(appID string) error { return s.start(appID, store.Update) }

// Remove queues an app's removal. Its data stays.
func (s *StoreService) Remove(appID string) error { return s.start(appID, store.Remove) }

// Cancel stops an app's operation or drops it from the queue.
func (s *StoreService) Cancel(appID string) { s.store.Cancel(appID) }

// Jobs returns the running and queued operations.
func (s *StoreService) Jobs() []store.Job { return s.store.Jobs() }

func (s *StoreService) start(appID string, action store.Action) error {
	ctx, cancel := context.WithTimeout(context.Background(), storeCallTimeout)
	defer cancel()
	err := s.store.Start(ctx, appID, action)
	if err != nil {
		log.Printf("%s %s: %v", action, appID, err)
	}
	return err
}

// TimerService runs the sleep timer.
type TimerService struct {
	timer *sleeptimer.Timer
	panel *panel
}

// Start suspends the machine after the given minutes.
func (s *TimerService) Start(minutes int) error {
	if minutes < 1 || minutes > 240 {
		return errors.New("timer must be between 1 and 240 minutes")
	}
	s.timer.Start(minutes)
	return nil
}

// Cancel stops the sleep timer.
func (s *TimerService) Cancel() { s.timer.Cancel() }

// State reports the sleep timer.
func (s *TimerService) State() sleeptimer.State { return s.timer.State() }

// SuspendNow cancels the countdown and suspends immediately.
func (s *TimerService) SuspendNow() error {
	s.panel.hide()
	return s.timer.SuspendNow()
}

// DisplayService controls the display brightness.
type DisplayService struct{ control *brightness.Control }

// Brightness returns the brightness in percent.
func (s *DisplayService) Brightness() (int, error) { return s.control.Get() }

// SetBrightness sets the brightness in percent.
func (s *DisplayService) SetBrightness(percent int) error { return s.control.Set(percent) }

// AudioService controls the default audio output.
type AudioService struct{ mixer *audio.Mixer }

// Volume returns the default output's volume.
func (s *AudioService) Volume() (audio.Volume, error) { return s.mixer.Volume() }

// SetVolume sets the default output's volume in percent.
func (s *AudioService) SetVolume(percent int) error { return s.mixer.SetVolume(percent) }

// ToggleMute mutes or unmutes the default output.
func (s *AudioService) ToggleMute() error { return s.mixer.ToggleMute() }

// Outputs lists the audio outputs.
func (s *AudioService) Outputs() ([]audio.Output, error) { return s.mixer.Outputs() }

// SetOutput makes an output the default.
func (s *AudioService) SetOutput(id int) error { return s.mixer.SetOutput(id) }

// ThemeService reports the desktop's GTK theme colors.
type ThemeService struct {
	mu    sync.Mutex
	theme gtktheme.Theme
}

// GTK returns the active GTK theme's palette.
func (s *ThemeService) GTK() gtktheme.Theme {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.theme
}

func (s *ThemeService) set(t gtktheme.Theme) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := s.theme != t
	s.theme = t
	return changed
}
