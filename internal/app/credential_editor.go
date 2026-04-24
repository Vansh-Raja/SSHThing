package app

import (
	"fmt"
	"strings"

	"github.com/Vansh-Raja/SSHThing/internal/ssh"
	"github.com/Vansh-Raja/SSHThing/internal/ui"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const privateKeyEditorHeight = 6

func (m *Model) initFormKeyEditor(value string) {
	area := textarea.New()
	area.Prompt = "  "
	area.Placeholder = "Paste your SSH private key"
	area.ShowLineNumbers = false
	area.EndOfBufferCharacter = ' '
	area.CharLimit = 0
	area.MaxHeight = 0
	area.SetHeight(privateKeyEditorHeight)
	area.SetValue(value)
	m.formKeyEditor = area
	m.formKeyEditorOriginal = value
	m.styleFormKeyEditor()
}

func (m *Model) styleFormKeyEditor() {
	focused, blurred := textarea.DefaultStyles()
	focused.Base = lipgloss.NewStyle().Foreground(m.theme.Text)
	focused.CursorLine = lipgloss.NewStyle()
	focused.Prompt = lipgloss.NewStyle().Foreground(m.theme.Accent)
	focused.Placeholder = lipgloss.NewStyle().Foreground(m.theme.Overlay)
	focused.Text = lipgloss.NewStyle().Foreground(m.theme.Text)
	focused.EndOfBuffer = lipgloss.NewStyle().Foreground(m.theme.Surface0)

	blurred.Base = lipgloss.NewStyle().Foreground(m.theme.Subtext)
	blurred.CursorLine = lipgloss.NewStyle()
	blurred.Prompt = lipgloss.NewStyle().Foreground(m.theme.Surface0)
	blurred.Placeholder = lipgloss.NewStyle().Foreground(m.theme.Overlay)
	blurred.Text = lipgloss.NewStyle().Foreground(m.theme.Subtext)
	blurred.EndOfBuffer = lipgloss.NewStyle().Foreground(m.theme.Surface0)

	m.formKeyEditor.FocusedStyle = focused
	m.formKeyEditor.BlurredStyle = blurred
}

func (m *Model) prepareFormKeyEditor(width int, focused bool) tea.Cmd {
	if width < 12 {
		width = 12
	}
	m.styleFormKeyEditor()
	m.formKeyEditor.SetWidth(width)
	m.formKeyEditor.SetHeight(privateKeyEditorHeight)
	if focused {
		return m.formKeyEditor.Focus()
	}
	m.formKeyEditor.Blur()
	return nil
}

func (m *Model) preparePrivateKeyPopupEditor(width, height int) tea.Cmd {
	if width < 12 {
		width = 12
	}
	if height < 3 {
		height = 3
	}
	m.styleFormKeyEditor()
	m.formKeyEditor.SetWidth(width)
	m.formKeyEditor.SetHeight(height)
	return m.formKeyEditor.Focus()
}

func (m Model) privateKeyPopupSize() (int, int) {
	width := m.width - 4
	if width > 132 {
		width = 132
	}
	if width < 40 {
		width = 40
	}

	height := m.height - 4
	if height < 10 {
		height = 10
	}
	return width, height
}

func (m *Model) openPrivateKeyEditor(clear bool) tea.Cmd {
	if clear {
		m.formKeyEditor.SetValue("")
	} else {
		m.formKeyEditor.SetValue(m.formFields[ui.FFAuthDet].Value)
	}
	m.formKeyEditor.CursorEnd()
	m.formKeyEditorOriginal = m.formFields[ui.FFAuthDet].Value
	m.formEditing = true
	m.formFields[ui.FFAuthDet].Masked = false
	m.formSecretRevealed = false
	m.overlay = OverlayKeyEditor

	width, height := m.privateKeyPopupSize()
	return m.preparePrivateKeyPopupEditor(width-4, max(3, height-6))
}

func (m *Model) closePrivateKeyEditor(discard bool) {
	if discard {
		m.formKeyEditor.SetValue(m.formKeyEditorOriginal)
	}
	m.formFields[ui.FFAuthDet].SetValue(m.formKeyEditor.Value())
	m.formSecretRevealed = false
	m.formEditing = false
	m.formKeyEditor.Blur()
	m.formFields[ui.FFAuthDet].Masked = true
	m.overlay = OverlayAddHost
}

func (m *Model) syncFormKeyFieldFromEditor() {
	m.formFields[ui.FFAuthDet].SetValue(m.formKeyEditor.Value())
}

func (m *Model) updateFormKeyEditor(msg tea.Msg) tea.Cmd {
	updated, cmd := m.formKeyEditor.Update(msg)
	m.formKeyEditor = updated
	m.syncFormKeyFieldFromEditor()
	return cmd
}

func (m Model) formKeyEditorView() string {
	view := strings.TrimRight(m.formKeyEditor.View(), "\n")
	if strings.TrimSpace(view) == "" {
		return ""
	}
	return view
}

func (m Model) handlePrivateKeyEditorKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.closePrivateKeyEditor(true)
		m.ensureFormFocusVisible()
		return m, nil
	}

	switch msg.String() {
	case "ctrl+s":
		normalized := normalizePrivateKey(m.formKeyEditor.Value())
		if err := ssh.ValidatePrivateKey(normalized); err != nil {
			m.err = fmt.Errorf("\u26A0 Invalid private key: %v", err)
			return m, nil
		}
		m.formKeyEditor.SetValue(normalized)
		m.formFields[ui.FFAuthDet].SetValue(normalized)
		m.closePrivateKeyEditor(false)
		m.err = fmt.Errorf("\u2713 private key saved")
		m.ensureFormFocusVisible()
		return m, nil
	case "ctrl+y":
		secret := strings.TrimSpace(m.formKeyEditor.Value())
		if secret == "" {
			m.err = fmt.Errorf("no private key to copy")
			return m, nil
		}
		if err := copyTokenToClipboard(secret); err != nil {
			m.err = fmt.Errorf("failed to copy private key: %v", err)
		} else {
			m.err = fmt.Errorf("\u2713 private key copied to clipboard")
		}
		return m, nil
	}

	cmd := m.updateFormKeyEditor(msg)
	return m, cmd
}
