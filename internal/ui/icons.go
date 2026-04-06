package ui

import "strings"

// IconSet holds all icons used in the UI.
type IconSet struct {
	Name string
	// Sidebar
	Home, Settings, Tokens string
	// Status
	Connected, Idle, Offline string
	// Markers
	ActiveMarker, InactiveMarker string
	// Groups
	Expanded, Collapsed string
	// Selection
	Selected, Focused string
	// Nav
	LeftArrow, RightArrow string
	// Input
	Bar, Cursor string
	// Misc
	Truncation, Rule string
	// Semantic
	Lock, Warning, ErrorIcon, Success string
	Folder, Edit, DeleteIcon, Add     string
	Save, Cancel, Shield              string
}

// UnicodeIcons is the default icon preset using standard Unicode characters.
var UnicodeIcons = IconSet{
	Name:           "Unicode",
	Home:           "\u2302",
	Settings:       "\u25CB",
	Tokens:         "\u25C7",
	Connected:      "\u25CF",
	Idle:           "\u25CB",
	Offline:        "\u00B7",
	ActiveMarker:   "\u2022",
	InactiveMarker: "\u00B7",
	Expanded:       "\u25BF",
	Collapsed:      "\u25B9",
	Selected:       "\u25B8",
	Focused:        "\u2192",
	LeftArrow:      "\u25C4",
	RightArrow:     "\u25BA",
	Bar:            "\u258F",
	Cursor:         "\u2588",
	Truncation:     "\u2026",
	Rule:           "\u2500",
	Lock:           "\u25C6",
	Warning:        "\u25B3",
	ErrorIcon:      "\u2717",
	Success:        "\u2713",
	Folder:         "\u25AA",
	Edit:           "~",
	DeleteIcon:     "\u00D7",
	Add:            "+",
	Save:           "\u25B8",
	Cancel:         "\u25CB",
	Shield:         "\u25C7",
}

// NerdFontIcons is the icon preset using Nerd Font glyphs.
var NerdFontIcons = IconSet{
	Name:           "Nerd Font",
	Home:           "\uf015",
	Settings:       "\uf013",
	Tokens:         "\uf084",
	Connected:      "\uf058",
	Idle:           "\uf192",
	Offline:        "\uf10c",
	ActiveMarker:   "\uf111",
	InactiveMarker: "\uf10c",
	Expanded:       "\uf078",
	Collapsed:      "\uf054",
	Selected:       "\uf0da",
	Focused:        "\uf061",
	LeftArrow:      "\uf053",
	RightArrow:     "\uf054",
	Bar:            "\u258F",
	Cursor:         "\u2588",
	Truncation:     "\uf141",
	Rule:           "\u2500",
	Lock:           "\uf023",
	Warning:        "\uf071",
	ErrorIcon:      "\uf00d",
	Success:        "\uf00c",
	Folder:         "\uf07b",
	Edit:           "\uf044",
	DeleteIcon:     "\uf1f8",
	Add:            "\uf067",
	Save:           "\uf0c7",
	Cancel:         "\uf05e",
	Shield:         "\uf132",
}

// IconPresets contains all available icon set presets.
var IconPresets = []IconSet{UnicodeIcons, NerdFontIcons}

// IconSetByName returns the icon set with the given name, or the default.
func IconSetByName(name string) (IconSet, int) {
	for i, s := range IconPresets {
		if strings.EqualFold(s.Name, name) {
			return s, i
		}
	}
	return UnicodeIcons, 0
}
