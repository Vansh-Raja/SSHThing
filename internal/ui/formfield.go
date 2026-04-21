package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// FormField is a custom text input with manual cursor control.
type FormField struct {
	Label  string
	Value  string
	Cursor int
	Masked bool
}

// NewFormField creates a new plain form field.
func NewFormField(label string) FormField {
	return FormField{Label: label}
}

// NewMaskedField creates a new masked (password) form field.
func NewMaskedField(label string) FormField {
	return FormField{Label: label, Masked: true}
}

// SetValue sets the field value and places cursor at end.
func (f *FormField) SetValue(v string) {
	f.Value = v
	f.Cursor = utf8.RuneCountInString(v)
}

// InsertRune inserts a rune at the cursor position.
func (f *FormField) InsertRune(r rune) {
	runes := []rune(f.Value)
	if f.Cursor > len(runes) {
		f.Cursor = len(runes)
	}
	f.Value = string(runes[:f.Cursor]) + string(r) + string(runes[f.Cursor:])
	f.Cursor++
}

// DeleteBack removes the character before the cursor.
func (f *FormField) DeleteBack() {
	runes := []rune(f.Value)
	if f.Cursor > 0 && f.Cursor <= len(runes) {
		f.Value = string(runes[:f.Cursor-1]) + string(runes[f.Cursor:])
		f.Cursor--
	}
}

// MoveLeft moves the cursor left by one.
func (f *FormField) MoveLeft() {
	if f.Cursor > 0 {
		f.Cursor--
	}
}

// MoveRight moves the cursor right by one.
func (f *FormField) MoveRight() {
	if f.Cursor < utf8.RuneCountInString(f.Value) {
		f.Cursor++
	}
}

// CursorToEnd moves the cursor to the end of the value.
func (f *FormField) CursorToEnd() {
	f.Cursor = utf8.RuneCountInString(f.Value)
}

// RenderInput renders a form field input (add-host style: focused vs editing).
func (r *Renderer) RenderInput(f FormField, focused bool, width int, blink bool, editing bool) string {
	if width < 8 {
		width = 8
	}

	barColor := r.Theme.Surface0
	textColor := r.Theme.Subtext
	if focused {
		barColor = r.Theme.Accent
		textColor = r.Theme.Text
	}

	bar := lipgloss.NewStyle().Foreground(barColor).Render(r.Icons.Bar)

	val := f.Value
	if f.Masked && val != "" {
		val = strings.Repeat("\u2022", utf8.RuneCountInString(val))
	}

	isEditing := focused && editing
	contentWidth := max(1, width-3)

	if isEditing {
		runes := []rune(val)
		cur := f.Cursor
		if cur > len(runes) {
			cur = len(runes)
		}
		start := 0
		if cur > contentWidth-1 {
			start = cur - (contentWidth - 1)
		}
		if start > len(runes) {
			start = len(runes)
		}
		end := min(len(runes), start+contentWidth)
		view := runes[start:end]
		localCur := cur - start
		if localCur < 0 {
			localCur = 0
		}
		if localCur > len(view) {
			localCur = len(view)
		}
		before := lipgloss.NewStyle().Foreground(textColor).Render(string(view[:localCur]))
		cursorChar := " "
		afterStart := localCur
		if localCur < len(view) {
			cursorChar = string(view[localCur])
			afterStart = localCur + 1
		}
		after := lipgloss.NewStyle().Foreground(textColor).Render(string(view[afterStart:]))
		if start > 0 {
			before = lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(r.Icons.Truncation) + before
		}
		if end < len(runes) {
			after += lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(r.Icons.Truncation)
		}
		var cursorStyled string
		if blink {
			cursorStyled = lipgloss.NewStyle().Foreground(r.Theme.Base).Background(r.Theme.Accent).Render(cursorChar)
		} else {
			cursorStyled = lipgloss.NewStyle().Foreground(r.Theme.Base).Background(r.Theme.Overlay).Render(cursorChar)
		}
		return "  " + bar + before + cursorStyled + after
	}

	display := val
	if utf8.RuneCountInString(display) > contentWidth {
		display = r.TruncStr(display, contentWidth)
	}
	displayVal := lipgloss.NewStyle().Foreground(textColor).Render(display)
	if focused {
		underline := lipgloss.NewStyle().Foreground(r.Theme.Accent).Render("_")
		return "  " + bar + displayVal + underline
	}

	return "  " + bar + displayVal
}

func (r *Renderer) RenderSecretPreview(value string, width int, revealed bool, maxLines int) []string {
	if width < 12 {
		width = 12
	}
	if maxLines < 1 {
		maxLines = 1
	}
	if strings.TrimSpace(value) == "" {
		return []string{"  " + lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("[empty key]")}
	}
	if !revealed {
		runeCount := utf8.RuneCountInString(value)
		message := fmt.Sprintf("[private key hidden · %d chars]", runeCount)
		if utf8.RuneCountInString(message) > width {
			message = r.TruncStr(message, width)
		}
		return []string{"  " + lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(message)}
	}

	rawLines := strings.Split(strings.TrimRight(value, "\n"), "\n")
	lines := make([]string, 0, maxLines)
	truncated := false
	for lineIdx, raw := range rawLines {
		wrappedLines := wrapPlainTextLines(raw, width)
		for wrapIdx, wrapped := range wrappedLines {
			if len(lines) >= maxLines {
				truncated = true
				break
			}
			lines = append(lines, "  "+lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render(wrapped))
			if len(lines) == maxLines && (wrapIdx < len(wrappedLines)-1 || lineIdx < len(rawLines)-1) {
				truncated = true
				break
			}
		}
		if len(lines) >= maxLines {
			break
		}
	}
	if len(lines) == 0 {
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("[empty key]"))
	}
	if truncated && len(lines) > 0 {
		lines[maxLines-1] = lines[maxLines-1] + lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(" "+r.Icons.Truncation)
	}
	return lines
}

// RenderModalField renders a field for modal overlays (always-editing when focused).
func (r *Renderer) RenderModalField(value string, cursor int, masked bool, focused bool, blink bool, bg lipgloss.Color) string {
	barColor := r.Theme.Surface0
	textColor := r.Theme.Subtext
	if focused {
		barColor = r.Theme.Accent
		textColor = r.Theme.Text
	}
	bar := lipgloss.NewStyle().Foreground(barColor).Background(bg).Render(r.Icons.Bar)

	val := value
	if masked && val != "" {
		val = strings.Repeat("\u2022", utf8.RuneCountInString(val))
	}

	if focused {
		runes := []rune(val)
		cur := cursor
		if cur > len(runes) {
			cur = len(runes)
		}
		before := lipgloss.NewStyle().Foreground(textColor).Background(bg).Render(string(runes[:cur]))
		cursorChar := " "
		afterStart := cur
		if cur < len(runes) {
			cursorChar = string(runes[cur])
			afterStart = cur + 1
		}
		var cursorR string
		if blink {
			cursorR = lipgloss.NewStyle().Foreground(r.Theme.Base).Background(r.Theme.Accent).Render(cursorChar)
		} else {
			cursorR = lipgloss.NewStyle().Foreground(r.Theme.Base).Background(r.Theme.Overlay).Render(cursorChar)
		}
		after := lipgloss.NewStyle().Foreground(textColor).Background(bg).Render(string(runes[afterStart:]))
		return "  " + bar + before + cursorR + after
	}

	displayVal := lipgloss.NewStyle().Foreground(textColor).Background(bg).Render(val)
	return "  " + bar + displayVal
}

// RenderFormLabel renders a form field label.
func (r *Renderer) RenderFormLabel(text string, focused bool) string {
	if focused {
		return "  " + lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(text)
	}
	return "  " + lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(text)
}
