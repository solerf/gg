package tui

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/solerf/gg/github"
)

const (
	loadingStatus = "Loading..."
	cloningStatus = "Cloning %v at %v"
)

var (
	viewBaseStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	viewBoldStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("57")).Bold(true)
	viewErrStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

type errMsg struct{ cause error }

func (e errMsg) Error() string {
	return e.cause.Error()
}

type updateTableMsg *repositoryTable
type triggerSpinnerMsg string
type repositoryClonedMsg string

type Model struct {
	homeDir  string
	gitUser  string
	ghClient *github.Client

	repositoryTable *repositoryTable

	selectedRepository *github.Repository
	cloneTo            string

	loading       bool
	loadingStatus string
	Done          bool

	err error

	tableModel     table.Model
	spinnerModel   spinner.Model
	textinputModel textinput.Model
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(triggerSpinnerCmd(loadingStatus), startCmd(m.gitUser, m.ghClient))
}

func (m Model) Update(receivedMsg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := receivedMsg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case sortNameField, sortLangField, sortDateField:
			m.repositoryTable.SortInPlace(keypress)
			return m, tea.Sequence(triggerSpinnerCmd(loadingStatus), updateTableCmd(m.repositoryTable))

		case "tab":
			if m.textinputModel.Focused() {
				m.textinputModel.SetValue(m.textinputModel.Placeholder)
			}

			if m.tableModel.Cursor() != -1 && !m.textinputModel.Focused() {
				m.selectedRepository = &m.repositoryTable.repositories[m.tableModel.Cursor()]
				m.textinputModel.Cursor.SetMode(cursor.CursorBlink)
				m.textinputModel.Focus()
			}

			return m, nil

		case "esc":
			m.selectedRepository = nil
			m.textinputModel.Cursor.SetMode(cursor.CursorHide)
			m.textinputModel.Blur()
			m.textinputModel.Reset()
			return m, nil

		case "enter":
			// this should clone the repo to the dir
			m.cloneTo = strings.TrimSpace(m.textinputModel.Value())
			return m, tea.Sequence(
				triggerSpinnerCmd(fmt.Sprintf(cloningStatus, m.selectedRepository.Name, m.cloneTo)),
				cloneCmd(m.selectedRepository, m.cloneTo),
			)

		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case triggerSpinnerMsg:
		m.loading = true
		m.loadingStatus = string(msg)
		return m, m.spinnerModel.Tick

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinnerModel, cmd = m.spinnerModel.Update(msg)
		return m, cmd

	case errMsg:
		m.loading = false
		m.loadingStatus = ""
		m.err = msg.cause
		return m, nil

	case updateTableMsg:
		// return nothing just go through
		updateTableView(&m, msg)

	case repositoryClonedMsg:
		m.Done = true
		m.loading = false
		m.loadingStatus = ""
		return m, tea.Quit
	}

	var cmd tea.Cmd
	if m.textinputModel.Focused() {
		// if focused we send the msgs (keys pressed) to be updated
		m.textinputModel, cmd = m.textinputModel.Update(receivedMsg)
		return m, cmd
	}

	// if not a msg related to the table it will just be ignored
	// it is at the end to ensure navigation work
	m.tableModel, cmd = m.tableModel.Update(receivedMsg)
	return m, cmd
}

func updateTableView(m *Model, repositoryTable *repositoryTable) {
	m.repositoryTable = repositoryTable
	m.selectedRepository = nil

	m.tableModel.SetRows(m.repositoryTable.rows)
	m.tableModel.SetCursor(0)

	m.loading = false
	m.loadingStatus = ""
}

func (m Model) View() string {
	if m.Done {
		return fmt.Sprintf("Cloned at %v, exiting...", viewBoldStyle.Height(1).Render(path.Join(m.cloneTo, m.selectedRepository.Name)))
	}

	if m.loading {
		return fmt.Sprintf("%v %v", m.spinnerModel.View(), m.loadingStatus)
	}

	var view strings.Builder
	view.WriteString(fmt.Sprintf("%v\n", viewBaseStyle.Render(m.tableModel.View())))

	if m.err != nil {
		view.WriteString(fmt.Sprintf("%v\n", viewErrStyle.Width(100).MaxHeight(3).Render(m.err.Error())))
	}

	if m.selectedRepository != nil {
		view.WriteString(
			fmt.Sprintf(
				"Clone, %s, at\n%s",
				viewBoldStyle.Render(m.selectedRepository.Name),
				m.textinputModel.View(),
			),
		)
	}
	return view.String()
}

func NewModel(debug bool, curDir, homeDir, gitUser string, ghClient *github.Client) (*Model, error) {
	tbl := table.New(
		table.WithColumns(repositoryTableCols),
		table.WithFocused(true),
		table.WithHeight(15),
	)
	sty := table.DefaultStyles()
	sty.Header = sty.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	sty.Selected = sty.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	tbl.SetStyles(sty)

	spin := spinner.New(spinner.WithStyle(
		lipgloss.NewStyle().Foreground(lipgloss.Color("57")).Bold(true),
	))

	ti := textinput.New()
	ti.CharLimit = 80
	ti.Width = 80
	ti.Placeholder = curDir
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("57"))
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("57"))
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("57"))

	return &Model{
		homeDir:        homeDir,
		gitUser:        gitUser,
		ghClient:       ghClient,
		tableModel:     tbl,
		spinnerModel:   spin,
		textinputModel: ti,
	}, nil
}

func startCmd(gitUser string, ghClient *github.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		repositories, err := ghClient.ListRepositories(ctx, gitUser)
		if err != nil {
			return errMsg{err}
		}

		repoTable := newRepositoriesTable(repositories)
		return updateTableCmd(repoTable)()
	}
}

func updateTableCmd(repoTable *repositoryTable) tea.Cmd {
	return func() tea.Msg {
		return updateTableMsg(repoTable)
	}
}

func triggerSpinnerCmd(status string) tea.Cmd {
	return func() tea.Msg {
		return triggerSpinnerMsg(status)
	}
}

func cloneCmd(repository *github.Repository, destination string) tea.Cmd {
	return func() tea.Msg {
		if err := github.Clone(repository, destination); err != nil {
			return errMsg{err}
		}
		return repositoryClonedMsg(path.Join(destination, repository.Name))
	}
}
