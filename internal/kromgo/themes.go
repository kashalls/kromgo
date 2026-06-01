package kromgo

import (
	catppuccin "github.com/catppuccin/go"
	charts "github.com/go-analyze/charts"
)

// Graph themes come in two flavors: the go-analyze/charts built-ins (selected by
// name) and kromgo's custom palettes below (Catppuccin via the official module,
// plus a few popular editor schemes). chartTheme resolves either; validTheme
// gates the config value at startup.

// builtinThemes are the go-analyze/charts themes we expose by name.
var builtinThemes = map[string]bool{
	charts.ThemeLight: true, charts.ThemeDark: true,
	charts.ThemeVividLight: true, charts.ThemeVividDark: true,
	charts.ThemeGrafana: true, charts.ThemeAnt: true,
	charts.ThemeNatureLight: true, charts.ThemeNatureDark: true,
	charts.ThemeRetro: true, charts.ThemeOcean: true,
	charts.ThemeSlate: true, charts.ThemeGray: true,
	charts.ThemeWinter: true, charts.ThemeSpring: true,
	charts.ThemeSummer: true, charts.ThemeFall: true,
}

// customThemes are palettes kromgo builds itself, keyed by the ?theme= value.
var customThemes = buildCustomThemes()

func buildCustomThemes() map[string]charts.ColorPalette {
	m := map[string]charts.ColorPalette{
		// Dracula — https://draculatheme.com
		"dracula": palette(true, "#282a36", "#f8f8f2", "#6272a4", "#44475a",
			"#bd93f9", "#8be9fd", "#50fa7b", "#ffb86c", "#ff79c6", "#ff5555", "#f1fa8c"),
		// Monokai
		"monokai": palette(true, "#272822", "#f8f8f2", "#75715e", "#3e3d32",
			"#66d9ef", "#a6e22e", "#f92672", "#fd971f", "#ae81ff", "#e6db74"),
		// Night Owl — https://github.com/sdras/night-owl-vscode-theme
		"night-owl": palette(true, "#011627", "#d6deeb", "#5f7e97", "#1d3b53",
			"#82aaff", "#c792ea", "#addb67", "#f78c6c", "#ef5350", "#7fdbca", "#ecc48d"),
	}
	for _, flavor := range []string{"latte", "frappe", "macchiato", "mocha"} {
		m["catppuccin-"+flavor] = catppuccinTheme(flavor)
	}
	return m
}

// palette builds a custom theme from hex strings: background, text, axis stroke,
// split-line, then one or more series colors.
func palette(dark bool, bg, text, axis, split string, series ...string) charts.ColorPalette {
	colors := make([]charts.Color, len(series))
	for i, s := range series {
		colors[i] = charts.ColorFromHex(s)
	}
	return charts.MakeTheme(charts.ThemeOption{
		IsDarkMode:         dark,
		BackgroundColor:    charts.ColorFromHex(bg),
		TextColor:          charts.ColorFromHex(text),
		AxisStrokeColor:    charts.ColorFromHex(axis),
		AxisSplitLineColor: charts.ColorFromHex(split),
		SeriesColors:       colors,
	})
}

// catppuccinTheme builds a theme from a Catppuccin flavor (latte is the light one).
func catppuccinTheme(flavor string) charts.ColorPalette {
	f := catppuccin.Variant(flavor)
	hex := func(c catppuccin.Color) charts.Color { return charts.ColorFromHex(c.Hex) }
	return charts.MakeTheme(charts.ThemeOption{
		IsDarkMode:         flavor != "latte",
		BackgroundColor:    hex(f.Base()),
		TextColor:          hex(f.Text()),
		AxisStrokeColor:    hex(f.Overlay0()),
		AxisSplitLineColor: hex(f.Surface0()),
		SeriesColors: []charts.Color{
			hex(f.Blue()), hex(f.Mauve()), hex(f.Green()),
			hex(f.Peach()), hex(f.Red()), hex(f.Teal()), hex(f.Yellow()),
		},
	})
}

// validTheme reports whether name is a known built-in or custom theme.
func validTheme(name string) bool {
	return builtinThemes[name] || customThemes[name] != nil
}

// chartTheme resolves a theme name to a palette, falling back to the library
// default for empty or unknown names.
func chartTheme(name string) charts.ColorPalette {
	if t := customThemes[name]; t != nil {
		return t
	}
	if builtinThemes[name] {
		return charts.GetTheme(name)
	}
	return charts.GetDefaultTheme()
}
