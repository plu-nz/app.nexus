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

// CloseAll asks Hyprland to close every window with an allowed class.
func (h *Hyprland) CloseAll(classes []string) (int, error) {
	out, err := h.Run("hyprctl", "-j", "clients")
	if err != nil {
		return 0, err
	}
	addresses, err := matchingWindows(out, classes)
	if err != nil || len(addresses) == 0 {
		return 0, err
	}
	var script strings.Builder
	for _, address := range addresses {
		fmt.Fprintf(&script, "local w = hl.get_window('address:%s'); if w then hl.dispatch(hl.dsp.window.close({ window = w })) end\n", address)
	}
	out, err = h.Run("hyprctl", "eval", script.String())
	if err == nil && strings.HasPrefix(strings.TrimSpace(out), "Error") {
		err = fmt.Errorf("hyprctl: %s", strings.TrimSpace(out))
	}
	return len(addresses), err
}

// Open reports which of classes have at least one window.
func (h *Hyprland) Open(classes []string) (map[string]bool, error) {
	out, err := h.Run("hyprctl", "-j", "clients")
	if err != nil {
		return nil, err
	}
	return openClasses(out, classes)
}

// openClasses returns the allowed classes that have a window with a valid
// address in `hyprctl -j clients`.
func openClasses(clients string, classes []string) (map[string]bool, error) {
	allowed := make(map[string]bool, len(classes))
	for _, class := range classes {
		allowed[class] = true
	}
	var list []client
	if err := json.Unmarshal([]byte(clients), &list); err != nil {
		return nil, fmt.Errorf("reading hyprctl clients: %w", err)
	}
	open := map[string]bool{}
	for _, c := range list {
		if allowed[c.Class] && addressPattern.MatchString(c.Address) {
			open[c.Class] = true
		}
	}
	return open, nil
}

// matchingWindows keeps only windows of explicitly allowed classes and valid
// addresses before they are used in a Hyprland command.
func matchingWindows(clients string, classes []string) ([]string, error) {
	allowed := make(map[string]bool, len(classes))
	for _, class := range classes {
		allowed[class] = true
	}
	var list []client
	if err := json.Unmarshal([]byte(clients), &list); err != nil {
		return nil, fmt.Errorf("reading hyprctl clients: %w", err)
	}
	var addresses []string
	for _, c := range list {
		if allowed[c.Class] && addressPattern.MatchString(c.Address) {
			addresses = append(addresses, c.Address)
		}
	}
	return addresses, nil
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

// focusScript is the Hyprland Lua that shows a window as a TV app: out of
// a special (scratch) workspace onto the current one, any other fullscreen
// window on that workspace back to normal, then focused and fullscreen.
// address must already be validated.
func focusScript(address string) string {
	return `local w = hl.get_window('address:` + address + `')
if w then
  local target = w.workspace
  local active = hl.get_active_workspace()
  if target and target.special and active and not active.special then
    hl.dispatch(hl.dsp.window.move({ window = w, workspace = active.name, follow = false }))
    target = active
  end
  for _, other in ipairs(hl.get_windows()) do
    if other.stable_id ~= w.stable_id and target and other.workspace and other.workspace.id == target.id then
      hl.dispatch(hl.dsp.window.fullscreen({ window = other, action = 'unset' }))
    end
  end
  hl.dispatch(hl.dsp.focus({ window = w }))
  hl.dispatch(hl.dsp.window.fullscreen({ window = w, mode = 'fullscreen', action = 'set' }))
end`
}
