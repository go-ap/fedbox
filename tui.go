//go:build ssh

package fedbox

import (
	"git.sr.ht/~mariusor/storage-all"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	vocab "github.com/go-ap/activitypub"
)

type Model struct {
	Actor   *vocab.Actor
	Storage storage.FullStorage

	renderer *lipgloss.Renderer
	Form     *huh.Form
	Style    lipgloss.Style
	w, h     int
	loggedIn bool
}

func TUIModel(base *Base, acc *vocab.Actor, r *lipgloss.Renderer, w, h int) *Model {
	return &Model{Actor: acc, Storage: base.Storage, renderer: r, w: w, h: h}
}

func (m *Model) Init() tea.Cmd {
	r := m.renderer

	m.Style = r.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(1, 2).
		BorderForeground(lipgloss.Color("#444444")).
		Foreground(lipgloss.Color("#7571F9"))

	custom := huh.ThemeBase()
	custom.Blurred.Title = r.NewStyle().
		Foreground(lipgloss.Color("#444"))
	custom.Blurred.TextInput.Prompt = r.NewStyle().
		Foreground(lipgloss.Color("#444"))
	custom.Blurred.TextInput.Text = r.NewStyle().
		Foreground(lipgloss.Color("#444"))
	custom.Focused.TextInput.Cursor = r.NewStyle().
		Foreground(lipgloss.Color("#7571F9"))
	custom.Focused.Base = r.NewStyle().
		Padding(0, 1).
		Border(lipgloss.ThickBorder(), false).
		BorderLeft(true).
		BorderForeground(lipgloss.Color("#7571F9"))

	name := ""
	if m.Actor != nil {
		name = vocab.PreferredNameOf(m.Actor)
	}

	m.Form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Username").Key("username").Value(&name),
			huh.NewInput().Title("Password").EchoMode(huh.EchoModePassword),
		),
	)
	m.Form.WithTheme(custom)

	return m.Form.Init()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if m.Form == nil {
		return m, nil
	}

	if m.Form != nil {
		f, cmd := m.Form.Update(msg)
		m.Form = f.(*huh.Form)
		cmds = append(cmds, cmd)
	}

	m.loggedIn = m.Form.State == huh.StateCompleted
	if m.Form.State == huh.StateAborted {
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Interrupt
		case "q":
			return m, tea.Quit
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if m.Form == nil {
		return "Starting..."
	}
	if m.loggedIn {
		return m.Style.Render("Welcome, " + m.Form.GetString("username") + "!")
	}
	return m.Form.View()
}
