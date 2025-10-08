package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/qrave1/go-service-bootstrap/internal/generator"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateProjectName state = iota
	stateConfig
	stateHTTPFramework
	stateDatabase
	stateTaskRunner
	stateFeatures // Multi-select
	stateSummary
	stateGenerating
	stateFinished
)

type keyMap struct {
	Enter key.Binding
	Back  key.Binding
	Quit  key.Binding
}

var keys = keyMap{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("backspace", "left"),
		key.WithHelp("backspace/left", "go back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("ctrl+c/q", "quit"),
	),
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Enter, k.Back, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Enter, k.Back, k.Quit}}
}

var (
	titleStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	questionStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)
	helpStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).PaddingLeft(2)
)

type choice struct {
	title string
	desc  string
}

func (i choice) FilterValue() string { return i.title }
func (i choice) Title() string       { return i.title }
func (i choice) Description() string { return i.desc }

type itemDelegate struct {
	m Model
}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(choice)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.Title())

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

func newItemDelegate(m Model) list.ItemDelegate {
	return itemDelegate{m: m}
}

type generationFinishedMsg struct{}

func runGenerator(projectName, config, httpFramework, database, taskRunner string, features map[string]struct{}) tea.Cmd {
	return func() tea.Msg {
		cfg := generator.Config{
			ProjectName: projectName,
		}

		switch config {
		case "YAML":
			cfg.HasYAML = true
		case ".env":
			cfg.HasEnv = true
		}

		switch httpFramework {
		case "Echo":
			cfg.IsEcho = true
		case "Fiber":
			cfg.IsFiber = true
		}

		switch database {
		case "PostgreSQL":
			cfg.HasPostgres = true
			cfg.HasDB = true
		case "MySQL":
			cfg.HasMysql = true
			cfg.HasDB = true
		case "SQLite":
			cfg.HasSqlite = true
			cfg.HasDB = true
		}

		switch taskRunner {
		case "Makefile":
			cfg.HasMakefile = true
		case "Taskfile":
			cfg.HasTaskfile = true
		}

		for f := range features {
			switch f {
			case "gorilla/websocket":
				cfg.HasWebSocket = true
			case "Enable HTML templates":
				cfg.HasHTML = true
			case "Telebot":
				cfg.HasTelegram = true
			}
		}

		_ = generator.Generate(cfg) // TODO: handle error
		return generationFinishedMsg{}
	}
}

type Model struct {
	state     state
	list      list.Model
	textInput textinput.Model
	spinner   spinner.Model
	help      help.Model
	keys      keyMap
	err       error

	// User choices
	projectName   string
	config        string
	httpFramework string
	database      string
	taskRunner    string
	features      map[string]struct{}
}

func (m Model) choicesForState() []list.Item {
	switch m.state {
	case stateConfig:
		return []list.Item{choice{title: "YAML"}, choice{title: ".env"}}
	case stateHTTPFramework:
		return []list.Item{choice{title: "Echo"}, choice{title: "Fiber"}}
	case stateDatabase:
		return []list.Item{choice{title: "PostgreSQL"}, choice{title: "MySQL"}, choice{title: "SQLite"}, choice{title: "None"}}
	case stateTaskRunner:
		return []list.Item{choice{title: "Makefile"}, choice{title: "Taskfile"}}
	case stateFeatures:
		return []list.Item{choice{title: "gorilla/websocket"}, choice{title: "Enable HTML templates"}, choice{title: "Telebot"}}
	default:
		return nil
	}
}

func InitialModel() Model {
	ti := textinput.New()
	ti.Placeholder = "my-awesome-service"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	s := spinner.New()
	s.Spinner = spinner.Pulse
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := Model{
		state:     stateProjectName,
		textInput: ti,
		features:  make(map[string]struct{}),
		spinner:   s,
		help:      help.New(),
		keys:      keys,
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *Model) nextState() tea.Cmd {
	m.state++
	if m.state >= stateSummary {
		return nil
	}

	choices := m.choicesForState()
	m.list = list.New(choices, newItemDelegate(*m), 0, 0)
	m.list.Title = "Choose an option"
	m.list.Styles.Title = titleStyle
	m.list.Styles.HelpStyle = helpStyle
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Back) && m.state != stateProjectName:
			m.state--
			if m.state == stateProjectName {
				m.textInput.Focus()
				m.list.SetItems(nil)
			} else {
				m.list.SetItems(m.choicesForState())
			}
			return m, nil
		}
	case generationFinishedMsg:
		m.state = stateFinished
		return m, tea.Quit
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	switch m.state {
	case stateProjectName:
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)

		if _, ok := msg.(tea.KeyMsg); ok && key.Matches(msg.(tea.KeyMsg), m.keys.Enter) {
			if m.textInput.Value() != "" {
				m.projectName = m.textInput.Value()
				m.textInput.Blur()
				cmds = append(cmds, m.nextState())
			}
		}
	case stateConfig, stateHTTPFramework, stateDatabase, stateTaskRunner, stateFeatures:
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)

		if _, ok := msg.(tea.KeyMsg); ok && key.Matches(msg.(tea.KeyMsg), m.keys.Enter) {
			if m.list.SelectedItem() != nil {
				selectedChoice := m.list.SelectedItem().(choice).title
				switch m.state {
				case stateConfig:
					m.config = selectedChoice
				case stateHTTPFramework:
					m.httpFramework = selectedChoice
				case stateDatabase:
					m.database = selectedChoice
				case stateTaskRunner:
					m.taskRunner = selectedChoice
				case stateFeatures:
					// Handle single-select for features
					m.features[selectedChoice] = struct{}{}
				}
				cmds = append(cmds, m.nextState())
			}
		}
	case stateSummary:
		if _, ok := msg.(tea.KeyMsg); ok && key.Matches(msg.(tea.KeyMsg), m.keys.Enter) {
			m.state = stateGenerating
			cmds = append(cmds, m.spinner.Tick, runGenerator(m.projectName, m.config, m.httpFramework, m.database, m.taskRunner, m.features))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	var s string

	switch m.state {
	case stateProjectName:
		s = fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			titleStyle.Render("What is the project name?"),
			m.textInput.View(),
			m.help.View(m.keys),
		)
	case stateConfig, stateHTTPFramework, stateDatabase, stateTaskRunner, stateFeatures:
		m.list.SetSize(m.list.Width(), m.list.Height())
		s = lipgloss.JoinVertical(lipgloss.Left, m.list.View(), m.help.View(m.keys))
	case stateSummary:
		var featuresList []string
		for f := range m.features {
			featuresList = append(featuresList, f)
		}
		s = fmt.Sprintf(
			"%s\n\n%s: %s\n%s: %s\n%s: %s\n%s: %s\n%s: %s\n%s: %s\n\n%s",
			titleStyle.Render("Summary"),
			questionStyle.Render("Project Name"), m.projectName,
			questionStyle.Render("Config Type"), m.config,
			questionStyle.Render("HTTP Framework"), m.httpFramework,
			questionStyle.Render("Database"), m.database,
			questionStyle.Render("Task Runner"), m.taskRunner,
			questionStyle.Render("Features"), strings.Join(featuresList, ", "),
			m.help.View(m.keys),
		)
	case stateGenerating:
		s = fmt.Sprintf(
			"%s %s\n\n%s",
			m.spinner.View(), titleStyle.Render("Generating project..."),
			m.help.View(m.keys),
		)
	case stateFinished:
		s = fmt.Sprintf(
			"%s\n\n%s",
			titleStyle.Render("Project generated successfully!"),
			m.help.View(m.keys),
		)
	}

	return s
}

// ... (keyMap, delegate, etc. remain mostly the same) ...
