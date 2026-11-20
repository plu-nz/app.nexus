package gtktheme

import "testing"

func TestParseOlderThemeNames(t *testing.T) {
	th := parse("theme_bg_color=21,21,21,1.000\ntheme_fg_color=255,255,255,1.000\n" +
		"theme_selected_bg_color=243,139,168,1.000\ntheme_selected_fg_color=0,0,0,0.870\n" +
		"borders=255,255,255,0.120\n@name=Darkppuccin\n@dark=0\n")
	want := Theme{
		Name: "Darkppuccin", Dark: true,
		Background: "rgb(21, 21, 21)", Text: "rgb(255, 255, 255)",
		Accent: "rgb(243, 139, 168)", OnAccent: "rgba(0, 0, 0, 0.87)", Border: "rgba(255, 255, 255, 0.12)",
	}
	if th != want {
		t.Fatalf("parse = %+v, want %+v", th, want)
	}
}

func TestParsePrefersClassicNames(t *testing.T) {
	// GTK's own theme answers black for the accent it resolves at draw time.
	th := parse("window_bg_color=0,0,0,1.000\ntheme_bg_color=250,250,250,1.000\n" +
		"accent_bg_color=0,0,0,1.000\ntheme_selected_bg_color=53,132,228,1.000\n@name=Adwaita\n")
	if th.Background != "rgb(250, 250, 250)" || th.Accent != "rgb(53, 132, 228)" {
		t.Fatalf("parse = %+v, want the classic colors", th)
	}
	if th.Dark {
		t.Fatal("a light background was reported as dark")
	}
}

func TestParseSkipsMalformedLines(t *testing.T) {
	th := parse("garbage\nwindow_bg_color=1,2\naccent_bg_color=x,1,2,1\n")
	if th != (Theme{}) {
		t.Fatalf("parse = %+v, want nothing", th)
	}
}

func TestParseFallsBackToLibadwaitaNames(t *testing.T) {
	th := parse("window_bg_color=36,36,36,1.000\naccent_bg_color=53,132,228,1.000\n")
	if th.Background != "rgb(36, 36, 36)" || th.Accent != "rgb(53, 132, 228)" || !th.Dark {
		t.Fatalf("parse = %+v, want the libadwaita colors", th)
	}
}
