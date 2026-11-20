package gtktheme

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Theme is a GTK theme reduced to the colors the UI uses. Colors are CSS
// rgb()/rgba() values; a color the theme does not define is empty.
type Theme struct {
	Name string `json:"name"`
	// Dark reports a dark background, whatever the theme calls itself.
	Dark       bool   `json:"dark"`
	Background string `json:"background"`
	Surface    string `json:"surface"`
	Text       string `json:"text"`
	Accent     string `json:"accent"`
	OnAccent   string `json:"onAccent"`
	Border     string `json:"border"`
	Subtle     string `json:"subtle"`
	Error      string `json:"error"`
	Success    string `json:"success"`
	Warning    string `json:"warning"`
}

type rgba struct{ r, g, b, a float64 }

func (c rgba) css() string {
	if c.a >= 1 {
		return fmt.Sprintf("rgb(%d, %d, %d)", int(c.r), int(c.g), int(c.b))
	}
	return fmt.Sprintf("rgba(%d, %d, %d, %.3g)", int(c.r), int(c.g), int(c.b), c.a)
}

// luminance is the relative luminance from WCAG.
func (c rgba) luminance() float64 {
	lin := func(v float64) float64 {
		v /= 255
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.r) + 0.7152*lin(c.g) + 0.0722*lin(c.b)
}

// parse reads the lines nexus_theme_colors writes.
func parse(out string) Theme {
	colors := map[string]rgba{}
	var t Theme
	for _, line := range strings.Split(out, "\n") {
		name, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if name == "@name" {
			t.Name = value
			continue
		}
		if strings.HasPrefix(name, "@") {
			continue
		}
		parts := strings.Split(value, ",")
		if len(parts) != 4 {
			continue
		}
		var c [4]float64
		valid := true
		for i, p := range parts {
			v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
			if err != nil {
				valid = false
				break
			}
			c[i] = v
		}
		if valid {
			colors[name] = rgba{c[0], c[1], c[2], c[3]}
		}
	}
	// Every GTK theme defines the classic names. Libadwaita's names come
	// second: GTK's own theme resolves some of them, such as the accent,
	// only at draw time, and a lookup returns black.
	pick := func(names ...string) (rgba, bool) {
		for _, n := range names {
			if c, ok := colors[n]; ok {
				return c, true
			}
		}
		return rgba{}, false
	}
	css := func(names ...string) string {
		if c, ok := pick(names...); ok {
			return c.css()
		}
		return ""
	}
	bg, hasBg := pick("theme_bg_color", "window_bg_color")
	if hasBg {
		t.Background = bg.css()
		t.Dark = bg.luminance() < 0.2
	}
	t.Surface = css("theme_base_color", "view_bg_color", "card_bg_color")
	t.Text = css("theme_fg_color", "window_fg_color", "theme_text_color", "view_fg_color")
	t.Accent = css("theme_selected_bg_color", "accent_bg_color", "accent_color")
	t.OnAccent = css("theme_selected_fg_color", "accent_fg_color")
	t.Border = css("borders")
	t.Subtle = css("insensitive_fg_color")
	t.Error = css("error_color", "destructive_bg_color")
	t.Success = css("success_color")
	t.Warning = css("warning_color")
	return t
}
