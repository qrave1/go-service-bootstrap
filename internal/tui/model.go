package tui

import (
	"fmt"
	"github.com/qrave1/go-service-bootstrap/internal/generator"
	"io"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type state int

const (
	stateProjectName state = iota
	stateOptions
	stateGenerating
	stateFinished
)

// A message to indicate generation is complete
type generationFinishedMsg struct {
	err error
}

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
	err         error
}

func InitialModel() Model {
	ti := textinput.New()
	ti.Placeholder = "my-awesome-service"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

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
		err:         nil,
	}

	l := list.New(items, itemDelegate{selected: &m.selected}, 0, 0)
	l.Title = "Select your service options (space to select, enter to confirm)"
	m.options = l

	return m
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state == stateProjectName {
			if msg.Type == tea.KeyEnter {
				m.state = stateOptions
				return m, nil
			}
		} else if m.state == stateOptions {
			switch msg.String() {
			case " ":
				if i, ok := m.options.SelectedItem().(choice); ok {
					if _, exists := m.selected[i.title]; exists {
						delete(m.selected, i.title)
					} else {
						m.selected[i.title] = struct{}{}
					}
				}
			case "enter":
				if i, ok := m.options.SelectedItem().(choice); ok && i.title == "Done" {
					m.state = stateGenerating
					cfg := generator.NewConfig(m.projectName.Value(), m.selected)
					return m, runGenerator(cfg)
				}
			}
		}

		if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEsc {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.options.SetSize(msg.Width-h, msg.Height-v)

	case generationFinishedMsg:
		m.state = stateFinished
		m.err = msg.err
		return m, tea.Quit // Quit after generation is done
	}

	switch m.state {
	case stateProjectName:
		m.projectName, cmd = m.projectName.Update(msg)
	case stateOptions:
		m.options, cmd = m.options.Update(msg)

	}

	return m, cmd
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	switch m.state {
	case stateProjectName:
		return docStyle.Render(fmt.Sprintf(
			"What is the name of your project?\n\n%s\n\n(press esc to quit)",
			m.projectName.View(),
		))
	case stateOptions:
		return docStyle.Render(m.options.View())
	case stateGenerating:
		return docStyle.Render("Generating your service...")
	case stateFinished:
		return docStyle.Render("Project generated successfully! You can now exit.")
	default:
		return "Unknown state"
	}
}

func newItemDelegate(selected *map[string]struct{}) list.ItemDelegate {
	return itemDelegate{selected: selected}
}

type itemDelegate struct {
	list.DefaultDelegate
	selected *map[string]struct{}
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	c, ok := item.(choice)
	if !ok {
		return
	}

	str := c.title
	if d.selected != nil {
		if _, exists := (*d.selected)[c.title]; exists {
			str = fmt.Sprintf("[x] %s", str)
		} else {
			str = fmt.Sprintf("[ ] %s", str)
		}
	}

	// Render the item
	fn := d.Styles.NormalTitle.Render
	if index == m.Index() {
		fn = d.Styles.SelectedTitle.Render
	}

	fmt.Fprint(w, fn(str))
}
