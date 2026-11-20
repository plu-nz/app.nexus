package apps

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Hyprland finds and focuses app windows through hyprctl.
type Hyprland struct {
	// Run executes a command and returns its standard output.
	Run func(name string, args ...string) (string, error)
}

// NewHyprland returns a Hyprland window finder when Nexus runs under
// Hyprland, and nil otherwise.
func NewHyprland() *Hyprland {
	if os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") == "" {
		return nil
	}
	return &Hyprland{Run: func(name string, args ...string) (string, error) {
		out, err := exec.Command(name, args...).Output()
		return string(out), err
	}}
}

type client struct {
	Address        string `json:"address"`
	Class          string `json:"class"`
	FocusHistoryID int    `json:"focusHistoryID"`
}

var addressPattern = regexp.MustCompile(`^0x[0-9a-f]+$`)

// Focus brings the most recently used window of class to the front,
// fullscreen. It reports false when the app has no window.
func (h *Hyprland) Focus(class string) (bool, error) {
	out, err := h.Run("hyprctl", "-j", "clients")
	if err != nil {
		return false, err
	}
	address, err := newestWindow(out, class)
	if err != nil || address == "" {
		return false, err
	}
	out, err = h.Run("hyprctl", "eval", focusScript(address))
	if err == nil && strings.HasPrefix(strings.TrimSpace(out), "Error") {
		err = fmt.Errorf("hyprctl: %s", strings.TrimSpace(out))
	}
	return true, err
}

// newestWindow returns the address of the most recently focused window of
// class in `hyprctl -j clients`, or "" when there is none.
func newestWindow(clients, class string) (string, error) {
	var list []client
	if err := json.Unmarshal([]byte(clients), &list); err != nil {
		return "", fmt.Errorf("reading hyprctl clients: %w", err)
	}
	best := -1
	for i, c := range list {
		if c.Class != class || !addressPattern.MatchString(c.Address) {
			continue
		}
		if best < 0 || c.FocusHistoryID < list[best].FocusHistoryID {
			best = i
		}
	}
	if best < 0 {
		return "", nil
	}
	return list[best].Address, nil
}

// focusScript is the Hyprland Lua that shows a window: out of a special
// (scratch) workspace onto the current one, focused, and fullscreen, as a
// TV app should be. address must already be validated.
func focusScript(address string) string {
	return `local w = hl.get_window('address:` + address + `')
if w then
  local active = hl.get_active_workspace()
  if w.workspace and w.workspace.special and active and not active.special then
    hl.dispatch(hl.dsp.window.move({ window = w, workspace = active.name, follow = false }))
  end
  hl.dispatch(hl.dsp.focus({ window = w }))
  hl.dispatch(hl.dsp.window.fullscreen({ window = w, mode = 'fullscreen', action = 'set' }))
end`
}
