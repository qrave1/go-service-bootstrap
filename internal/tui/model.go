package tui

import (
	"fmt"
	"io"

	"github.com/qrave1/go-service-bootstrap/internal/generator"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	docStyle      = lipgloss.NewStyle().Margin(1, 2)
	titleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFDF5")).Bold(true).Padding(0, 1, 0, 2)
	itemStyle     = lipgloss.NewStyle().PaddingLeft(4)
	selectedStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("47")).Bold(true)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type state int

const (
	stateProjectName state = iota
	stateOptions
	stateGenerating
	stateFinished
)

type generationFinishedMsg struct{ err error }

func runGenerator(cfg generator.Config) tea.Cmd {
	return func() tea.Msg {
		err := generator.Generate(cfg)
		return generationFinishedMsg{err: err}
	}
}

type choice struct {
	title string
	group string
}

func (c choice) Title() string       { return c.title }
func (c choice) Description() string { return c.group }
func (c choice) FilterValue() string { return c.title }

type Model struct {
	state       state
	projectName textinput.Model
	options     list.Model
	selected    map[string]struct{}
	spinner     spinner.Model
	help        help.Model
	keys        keyMap
	err         error
}

func InitialModel() Model {
	ti := textinput.New()
	ti.Placeholder = "my-awesome-service"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	items := []list.Item{
		choice{title: "Echo", group: "HTTP Framework"},
		choice{title: "Fiber", group: "HTTP Framework"},
		choice{title: "PostgreSQL", group: "Database"},
		choice{title: "MySQL", group: "Database"},
		choice{title: "SQLite", group: "Database"},
		choice{title: "gorilla/websocket", group: "WebSocket"},
		choice{title: "Enable HTML templates", group: "Features"},
		choice{title: "Makefile", group: "Task Runner"},
		choice{title: "Taskfile", group: "Task Runner"},
		choice{title: "Done", group: ""},
	}

	m := Model{
		state:       stateProjectName,
		projectName: ti,
		selected:    make(map[string]struct{}),
		spinner:     s,
		help:        help.New(),
		keys:        keys,
		err:         nil,
	}

	l := list.New(items, newItemDelegate(&m), 0, 0)
	l.Title = "Select your service options"
	l.SetShowFilter(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.HelpStyle = helpStyle
	m.options = l

	return m
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.Width = msg.Width
		h, v := docStyle.GetFrameSize()
		m.options.SetSize(msg.Width-h, msg.Height-v)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}

		switch m.state {
		case stateProjectName:
			if key.Matches(msg, m.keys.Confirm) {
				m.state = stateOptions
				return m, nil
			}
		case stateOptions:
			if key.Matches(msg, m.keys.Select) {
				if i, ok := m.options.SelectedItem().(choice); ok {
					if _, exists := m.selected[i.title]; exists {
						delete(m.selected, i.title)
					} else {
						m.selected[i.title] = struct{}{}
					}
				}
			}
			if key.Matches(msg, m.keys.Confirm) {
				if i, ok := m.options.SelectedItem().(choice); ok && i.title == "Done" {
					m.state = stateGenerating
					cfg := generator.NewConfig(m.projectName.Value(), m.selected)
					return m, runGenerator(cfg)
				}
			}
		}

	case generationFinishedMsg:
		m.state = stateFinished
		m.err = msg.err
		return m, tea.Quit

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.state {
	case stateProjectName:
		m.projectName, cmd = m.projectName.Update(msg)
		cmds = append(cmds, cmd)
	case stateOptions:
		m.options, cmd = m.options.Update(msg)
		cmds = append(cmds, cmd)
	case stateGenerating:
		cmds = append(cmds, m.spinner.Tick)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.err != nil {
		return docStyle.Render(fmt.Sprintf("\n%s %s\n", errorStyle.Render("Error:"), m.err))
	}

	switch m.state {
	case stateProjectName:
		return docStyle.Render(fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			titleStyle.Render("What is the name of your project?"),
			m.projectName.View(),
			m.help.View(m.keys),
		))
	case stateOptions:
		return docStyle.Render(m.options.View() + "\n" + m.help.View(m.keys))
	case stateGenerating:
		return docStyle.Render(fmt.Sprintf("%s Generating your service...", m.spinner.View()))
	case stateFinished:
		return docStyle.Render(successStyle.Render("✔ Project generated successfully!"))
	default:
		return "Unknown state"
	}
}

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Help    key.Binding
	Quit    key.Binding
	Select  key.Binding
	Confirm key.Binding
}

var keys = keyMap{
	Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "move up")),
	Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "move down")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
	Quit:    key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "quit")),
	Select:  key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select")),
	Confirm: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select, k.Confirm},
		{k.Help, k.Quit},
	}
}

type itemDelegate struct {
	list.DefaultDelegate
	m *Model
}

func (d itemDelegate) Render(w io.Writer, l list.Model, index int, item list.Item) {
	c, ok := item.(choice)
	if !ok {
		return
	}

	isSelected := false
	if _, ok := d.m.selected[c.title]; ok {
		isSelected = true
	}

	var checkbox string
	if isSelected {
		checkbox = selectedStyle.Render("[x]")
	} else {
		checkbox = itemStyle.Render("[ ]")
	}

	title := c.Title()
	if l.Index() == index {
		title = d.Styles.SelectedTitle.Render(title)
	} else {
		title = d.Styles.NormalTitle.Render(title)
	}

	fmt.Fprintf(w, "%s %s", checkbox, title)
}

func newItemDelegate(m *Model) list.ItemDelegate {
	d := itemDelegate{m: m}

	d.Styles.SelectedTitle = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("229")).Foreground(lipgloss.Color("229")).Padding(0, 0, 0, 1)

	d.Styles.NormalTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(0, 0, 0, 1)

	return d
}
