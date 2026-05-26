package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/guancioul/oss-radar/internal/data"
	"github.com/guancioul/oss-radar/internal/theme"
	"github.com/guancioul/oss-radar/internal/ui/screens"
)

var dashCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Open the TUI dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDashboard(dataDir)
	},
}

func init() {
	rootCmd.AddCommand(dashCmd)
}

type viewState int

const (
	viewPipeline viewState = iota
	viewIssue
)

type appModel struct {
	pipeline screens.PipelineModel
	viewer   screens.ViewerModel
	state    viewState
	dataDir  string
	theme    theme.Theme
}

func (m *appModel) reload() {
	issues := data.LoadIssues(m.dataDir)
	metrics := data.ComputeMetrics(issues)
	m.pipeline = m.pipeline.WithReloadedData(issues, metrics)
}

func (m appModel) Init() tea.Cmd { return nil }

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.pipeline.Resize(msg.Width, msg.Height)
		if m.state == viewIssue {
			m.viewer.Resize(msg.Width, msg.Height)
		}
		pm, cmd := m.pipeline.Update(msg)
		m.pipeline = pm
		return m, cmd

	case screens.PipelineClosedMsg:
		return m, tea.Quit

	case screens.PipelineOpenReportMsg:
		m.viewer = screens.NewViewerModel(m.theme, msg.Path, msg.Title, m.pipeline.Width(), m.pipeline.Height())
		m.state = viewIssue
		return m, nil

	case screens.ViewerClosedMsg:
		m.state = viewPipeline
		return m, nil

	case screens.PipelineUpdateStatusMsg:
		data.UpdateIssueStatus(m.dataDir, msg.IssueURL, msg.NewStatus)
		m.reload()
		return m, nil

	case screens.PipelineRefreshMsg:
		m.reload()
		return m, nil

	case screens.PipelineOpenURLMsg:
		return m, func() tea.Msg {
			var cmd *exec.Cmd
			switch runtime.GOOS {
			case "darwin":
				cmd = exec.Command("open", msg.URL)
			default:
				cmd = exec.Command("xdg-open", msg.URL)
			}
			_ = cmd.Run()
			return nil
		}

	default:
		if m.state == viewIssue {
			vm, cmd := m.viewer.Update(msg)
			m.viewer = vm
			return m, cmd
		}
		pm, cmd := m.pipeline.Update(msg)
		m.pipeline = pm
		return m, cmd
	}
}

func (m appModel) View() string {
	if m.state == viewIssue {
		return m.viewer.View()
	}
	return m.pipeline.View()
}

func runDashboard(dir string) error {
	issues := data.LoadIssues(dir)
	if issues == nil {
		fmt.Fprintf(os.Stderr, "No issues.json found in %s. Run `oss-radar scan` first.\n", dir)
		return nil
	}
	metrics := data.ComputeMetrics(issues)
	t := theme.NewTheme("auto")
	pm := screens.NewPipelineModel(t, issues, metrics, dir, 120, 40)

	m := appModel{
		pipeline: pm,
		dataDir:  dir,
		theme:    t,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
