package main

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/plu-nz/app.nexus/internal/apps"
	"github.com/plu-nz/app.nexus/internal/audio"
	"github.com/plu-nz/app.nexus/internal/brightness"
	"github.com/plu-nz/app.nexus/internal/gtktheme"
	"github.com/plu-nz/app.nexus/internal/sleeptimer"
)

// The services are the methods the UI calls by name
// (main.<Service>.<Method>). They are thin; the logic lives in internal/.

// PanelService shows and hides the panel.
type PanelService struct{ panel *panel }

// Hide hides the panel. Nexus keeps running, so the sleep timer survives.
func (s *PanelService) Hide() { s.panel.hide() }

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
