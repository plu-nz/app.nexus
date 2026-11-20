package apps

import (
	"bufio"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
)

// IconFinder looks up an app's icon the way desktop shells do: the Icon key
// of its .desktop file, then the hicolor icon theme and pixmaps.
type IconFinder struct {
	// DataDirs are XDG data directories, most important first.
	DataDirs []string
}

// SystemIcons returns an IconFinder over the user's XDG data directories
// and the Flatpak exports.
func SystemIcons() IconFinder {
	home, _ := os.UserHomeDir()
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	dirs := []string{dataHome, filepath.Join(dataHome, "flatpak", "exports", "share")}
	system := os.Getenv("XDG_DATA_DIRS")
	if system == "" {
		system = "/usr/local/share:/usr/share"
	}
	dirs = append(dirs, filepath.SplitList(system)...)
	dirs = append(dirs, "/var/lib/flatpak/exports/share")
	seen := map[string]bool{}
	unique := []string{}
	for _, d := range dirs {
		if d != "" && !seen[d] {
			seen[d] = true
			unique = append(unique, d)
		}
	}
	return IconFinder{DataDirs: unique}
}

// Larger sizes first: the panel shows icons big, and SVG scales best.
var iconSizes = []string{"scalable", "512x512", "256x256", "192x192", "128x128", "96x96", "64x64", "48x48"}

// iconExts is in order of preference.
var iconExts = []string{".svg", ".png"}

var iconTypes = map[string]string{".svg": "image/svg+xml", ".png": "image/png"}

// Path returns the icon file for a desktop ID, or "" when there is none.
func (f IconFinder) Path(desktopID string) string {
	name := f.iconName(desktopID)
	if name == "" {
		return ""
	}
	if filepath.IsAbs(name) {
		if _, err := os.Stat(name); err == nil {
			return name
		}
		return ""
	}
	for _, size := range iconSizes {
		for _, dir := range f.DataDirs {
			for _, ext := range iconExts {
				p := filepath.Join(dir, "icons", "hicolor", size, "apps", name+ext)
				if _, err := os.Stat(p); err == nil {
					return p
				}
			}
		}
	}
	for _, dir := range f.DataDirs {
		for _, ext := range iconExts {
			p := filepath.Join(dir, "pixmaps", name+ext)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return ""
}

// DataURL returns the icon as a data: URL, or "" when there is none.
func (f IconFinder) DataURL(desktopID string) string {
	p := f.Path(desktopID)
	mime, ok := iconTypes[strings.ToLower(filepath.Ext(p))]
	if p == "" || !ok {
		return ""
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
}

// iconName reads the Icon key of the desktop entry, falling back to the
// desktop ID, which is also the usual icon name.
func (f IconFinder) iconName(desktopID string) string {
	for _, dir := range f.DataDirs {
		file, err := os.Open(filepath.Join(dir, "applications", desktopID+".desktop"))
		if err != nil {
			continue
		}
		inEntry := false
		scan := bufio.NewScanner(file)
		for scan.Scan() {
			line := strings.TrimSpace(scan.Text())
			if strings.HasPrefix(line, "[") {
				inEntry = line == "[Desktop Entry]"
				continue
			}
			if value, ok := strings.CutPrefix(line, "Icon="); ok && inEntry {
				file.Close()
				return strings.TrimSpace(value)
			}
		}
		file.Close()
		break
	}
	return desktopID
}
