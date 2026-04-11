package tui

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/solerf/gg/github"
)

var (
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("30")).Bold(true)
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).MaxHeight(3)

	finishMsgStyle = highlightStyle.Height(1)

	highlightStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("30"))

	listHighlightStyle = highlightStyle.
				BorderForeground(lipgloss.Color("30")).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				Padding(0, 0, 0, 0)

	listInfoStyle = listHighlightStyle.
			Foreground(lipgloss.Color("245")).
			Padding(0, 0, 0, 4)

	listInfoPaddingTopStyle = listHighlightStyle.PaddingTop(1)

	listInfoPaddingBottomStyle = listInfoStyle.PaddingBottom(1)

	listNotHighlightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("251"))

	normalStyle = lipgloss.NewStyle()
)

type errMsg struct{ cause error }

func (e errMsg) Error() string {
	return e.cause.Error()
}

type repositoriesFetchedMsg []list.Item
type repositoryClonedMsg string

type triggerSpinnerMsg string

type item struct {
	r            github.Repository
	renderedName string
}

func newItem(r github.Repository) item {
	return item{
		r:            r,
		renderedName: normalStyle.Render(r.Name),
	}
}

func (i item) FilterValue() string {
	return i.r.Name
}

type itemDelegate struct{}

func (d itemDelegate) Height() int  { return 1 }
func (d itemDelegate) Spacing() int { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i := listItem.(item)
	content := i.renderedName
	if index == m.Index() {
		info := lipgloss.JoinVertical(lipgloss.Left,
			listInfoStyle.Render(i.r.FullName),
			lipgloss.JoinHorizontal(lipgloss.Left, listInfoStyle.Render("Last Update: "), listNotHighlightStyle.Render(i.r.LastUpdate.Format(time.RFC3339))),
			lipgloss.JoinHorizontal(lipgloss.Left, listInfoStyle.Render("Visibility: "), listNotHighlightStyle.Render(i.r.Visibility)),
			lipgloss.JoinHorizontal(lipgloss.Left, listInfoStyle.Render("Clone: "), listNotHighlightStyle.Render(i.r.CloneUrl)),
			lipgloss.JoinHorizontal(lipgloss.Left, listInfoPaddingBottomStyle.Render("Web: "), listNotHighlightStyle.Render(i.r.HtmlUrl)),
		)
		content = fmt.Sprintf("%s\n%s", listInfoPaddingTopStyle.Render(i.r.Name), info)
	}
	_, _ = fmt.Fprint(w, content)
}

type Model struct {
	curDir   string
	ghClient *github.Client

	spinnerModel spinner.Model
	loading      bool
	status       string

	repositories list.Model

	textInputModel textinput.Model

	targetRepository *github.Repository
	targetDirToClone string
	GitCloneFinished bool

	err error

	wWidth  int
	wHeight int
}

func NewModel(curDir string, ghClient *github.Client) (Model, error) {
	spin := spinner.New(spinner.WithStyle(spinnerStyle))
	return Model{
		curDir:       curDir,
		ghClient:     ghClient,
		spinnerModel: spin,
	}, nil
}

func initRepositoriesAndTextInput(items []list.Item, width int, height int) (list.Model, textinput.Model) {
	listModel := list.New(items, itemDelegate{}, width, height/3)
	listModel.SetShowTitle(false)
	listModel.SetShowHelp(false)
	listModel.SetShowStatusBar(false)
	listModel.SetShowTitle(false)
	listModel.SetFilteringEnabled(true)
	listModel.SetShowFilter(true)

	txtInput := textinput.New()
	txtInput.CharLimit = 100
	txtInput.Width = 100
	txtInput.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("30"))
	txtInput.Cursor.SetMode(cursor.CursorBlink)
	txtInput.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("30"))
	txtInput.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("30"))
	return listModel, txtInput
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(triggerSpinnerCmd("Loading..."), fetchGitHubCmd(m.ghClient))
}

func (m Model) Update(receivedMsg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := receivedMsg.(type) {

	case tea.KeyMsg:
		if !m.loading {
			switch keypress := msg.String(); keypress {

			case "esc":
				if m.textInputModel.Focused() {
					// undo selection
					m.textInputModel.Blur()
					m.textInputModel.Placeholder = ""
					m.textInputModel.SetValue("")
					m.targetRepository = nil
					return m, nil
				}

				var cmd tea.Cmd
				m.repositories, cmd = m.repositories.Update(receivedMsg)
				return m, cmd

			case "enter", "tab":
				if m.textInputModel.Focused() && keypress == "tab" {
					// complete with placeholder
					m.textInputModel.SetValue(m.textInputModel.Placeholder)
					return m, nil
				}

				if m.textInputModel.Focused() && keypress == "enter" {
					// finish and trigger gh clone
					return m, tea.Sequence(triggerSpinnerCmd("Cloning..."), cloneGitHubCmd(m.targetRepository, m.textInputModel.Value()))
				}

				// otherwise update selection from list
				i := m.repositories.Items()[m.repositories.GlobalIndex()].(item)
				m.targetRepository = &i.r
				m.textInputModel.Placeholder = fmt.Sprintf("%v/%v", m.curDir, i.r.Name)
				m.textInputModel.Focus()
				return m, nil

			default:
				var cmd tea.Cmd
				if m.textInputModel.Focused() && keypress != "ctrl+c" {
					m.textInputModel, cmd = m.textInputModel.Update(receivedMsg)
				} else {
					m.repositories, cmd = m.repositories.Update(receivedMsg)
				}
				return m, cmd
			}
		}
	case repositoriesFetchedMsg:
		m.repositories, m.textInputModel = initRepositoriesAndTextInput(msg, m.wWidth, m.wHeight)
		m.loading = false
		m.status = ""
		return m, nil

	case repositoryClonedMsg:
		m.GitCloneFinished = true
		m.loading = false
		m.targetDirToClone = m.textInputModel.Value()
		return m, tea.Quit

	case triggerSpinnerMsg:
		m.loading = true
		m.status = string(msg)
		return m, m.spinnerModel.Tick

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinnerModel, cmd = m.spinnerModel.Update(msg)
			return m, cmd
		}
		return m, nil

	case errMsg:
		m.loading = false
		m.status = ""
		m.err = msg.cause
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.wWidth = msg.Width
		m.wHeight = msg.Height
		return m, nil
	}

	if !m.loading {
		var cmd tea.Cmd
		if m.textInputModel.Focused() {
			// if focused we send the msgs (keys pressed) to be updated
			m.textInputModel, cmd = m.textInputModel.Update(receivedMsg)
			return m, cmd
		}
		m.repositories, cmd = m.repositories.Update(receivedMsg)
		return m, cmd
	}
	return m, nil
}

func (m Model) View() string {
	if m.GitCloneFinished {
		return fmt.Sprintf("Cloned at %v, exiting...", finishMsgStyle.Render(m.targetDirToClone))
	}

	if m.err != nil {
		return fmt.Sprintf("%v\n", errStyle.Width(m.wWidth).Render(m.err.Error()))
	}

	if m.loading {
		return fmt.Sprintf("%v %v", m.spinnerModel.View(), m.status)
	}

	targetRepoView := ""
	if m.targetRepository != nil {
		targetRepoView = fmt.Sprintf("\n\nClone [%s] at %s", highlightStyle.Render(m.targetRepository.Name), m.textInputModel.View())
	}
	return fmt.Sprintf("%s%s", m.repositories.View(), targetRepoView)
}

func triggerSpinnerCmd(status string) tea.Cmd {
	return func() tea.Msg {
		return triggerSpinnerMsg(status)
	}
}

func fetchGitHubCmd(ghClient *github.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		ghRepositories, err := ghClient.ListRepositories(ctx)
		if err != nil {
			return errMsg{err}
		}

		if len(ghRepositories) == 0 {
			return nil
		}

		items := make([]list.Item, len(ghRepositories))
		for i, r := range ghRepositories {
			items[i] = newItem(r)
		}
		return repositoriesFetchedMsg(items)
	}
}

func cloneGitHubCmd(ghRepo *github.Repository, targetDir string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancelF := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancelF()

		if err := github.Clone(ctx, ghRepo, targetDir); err != nil {
			return errMsg{err}
		}
		return repositoryClonedMsg(targetDir)
	}
}
