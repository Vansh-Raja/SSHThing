package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RenderLoginView renders the master password login screen
func (s *Styles) RenderLoginView(width, height int, input textinput.Model, err error) string {
	var b strings.Builder

	b.WriteString(s.Title.Render("SSH Manager - LOGIN"))
	b.WriteString("\n\n")
	b.WriteString("Enter Master Password to unlock:\n\n")

	// Render input (minimal, centered)
	input.Width = 28
	input.Prompt = ""
	inputView := s.FormInputFocused.Width(36).Render(input.View())
	b.WriteString(lipgloss.PlaceHorizontal(46, lipgloss.Center, inputView))
	b.WriteString("\n\n")

	if err != nil {
		b.WriteString(s.Error.Render(err.Error()))
		b.WriteString("\n\n")
	}

	b.WriteString(s.HelpValue.Foreground(ColorTextDim).Render("[Enter] Unlock • [Esc] Quit"))

	// Center the box
	content := s.LoginBox.Render(b.String())

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// RenderSetupView renders the first-run password setup screen
func (s *Styles) RenderSetupView(width, height int, passwordInput, confirmInput textinput.Model, focus int, err error) string {
	var b strings.Builder

	b.WriteString(s.Title.Render("SSH Manager - FIRST RUN SETUP"))
	b.WriteString("\n\n")
	b.WriteString("Create a master password to encrypt your database.\n")
	b.WriteString(s.HelpValue.Foreground(ColorTextDim).Render("(minimum 8 characters)"))
	b.WriteString("\n\n")

	// Password input (minimal, centered)
	passwordInput.Width = 28
	passwordInput.Prompt = ""
	passwordStyle := s.FormInput.Width(36)
	if focus == 0 {
		passwordStyle = s.FormInputFocused.Width(36)
	}
	b.WriteString(lipgloss.PlaceHorizontal(46, lipgloss.Center, passwordStyle.Render(passwordInput.View())))
	b.WriteString("\n\n")

	// Confirm input
	confirmInput.Width = 28
	confirmInput.Prompt = ""
	confirmStyle := s.FormInput.Width(36)
	if focus == 1 {
		confirmStyle = s.FormInputFocused.Width(36)
	}
	b.WriteString(lipgloss.PlaceHorizontal(46, lipgloss.Center, confirmStyle.Render(confirmInput.View())))
	b.WriteString("\n\n")

	// Submit button
	var btnStyle lipgloss.Style
	if focus == 2 {
		btnStyle = s.FormButtonFocused
	} else {
		btnStyle = s.FormButton
	}
	b.WriteString(lipgloss.PlaceHorizontal(46, lipgloss.Center, btnStyle.Render("Create Database")))
	b.WriteString("\n\n")

	if err != nil {
		b.WriteString(s.Error.Render(err.Error()))
		b.WriteString("\n\n")
	}

	b.WriteString(s.HelpValue.Foreground(ColorTextDim).Render("[Tab] Next • [Enter] Submit • [Esc] Quit"))

	// Center the box
	content := s.LoginBox.Render(b.String())

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}
