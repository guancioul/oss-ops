package screens

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/guancioul/oss-radar/internal/model"
	"github.com/guancioul/oss-radar/internal/theme"
)

// ProgressClosedMsg is emitted when the progress screen is dismissed.
type ProgressClosedMsg struct{}

// OSSProgressMetrics holds OSS-specific analytics.
type OSSProgressMetrics struct {
	Total      int
	InProgress int
	Merged     int
	Candidate  int
	Evaluating int
	Skip       int
	AvgScore   float64
	// Monthly merged counts (last 3 months)
	MonthlyMerged []MonthCount
}

// MonthCount represents merged PR count for a given month.
type MonthCount struct {
	Label string
	Count int
}

// ProgressModel implements the OSS progress analytics screen.
type ProgressModel struct {
	metrics      model.PipelineMetrics
	scrollOffset int
	width        int
	height       int
	theme        theme.Theme
}

// NewProgressModel creates a new progress screen.
func NewProgressModel(t theme.Theme, metrics model.PipelineMetrics, width, height int) ProgressModel {
	return ProgressModel{
		metrics: metrics,
		width:   width,
		height:  height,
		theme:   t,
	}
}

// Init implements tea.Model.
func (m ProgressModel) Init() tea.Cmd {
	return nil
}

// Resize updates dimensions.
func (m *ProgressModel) Resize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles input for the progress screen.
func (m ProgressModel) Update(msg tea.Msg) (ProgressModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return ProgressClosedMsg{} }
		case "down", "j":
			m.scrollOffset++
		case "up", "k":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		case "pgdown", "ctrl+d":
			m.scrollOffset += m.height / 2
		case "pgup", "ctrl+u":
			m.scrollOffset -= m.height / 2
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// View renders the progress screen.
func (m ProgressModel) View() string {
	header := m.renderHeader()
	pipeline := m.renderPipelineStats()
	scores := m.renderScoreDistribution()
	help := m.renderHelp()

	body := lipgloss.JoinVertical(lipgloss.Left,
		pipeline,
		"",
		scores,
	)

	// Apply scroll
	bodyLines := strings.Split(body, "\n")
	offset := m.scrollOffset
	if offset >= len(bodyLines) {
		offset = len(bodyLines) - 1
	}
	if offset < 0 {
		offset = 0
	}
	if offset > 0 {
		bodyLines = bodyLines[offset:]
	}

	availHeight := m.height - 4
	if availHeight < 3 {
		availHeight = 3
	}
	if len(bodyLines) > availHeight {
		bodyLines = bodyLines[:availHeight]
	}

	body = strings.Join(bodyLines, "\n")

	return lipgloss.JoinVertical(lipgloss.Left, header, body, help)
}

func (m ProgressModel) renderHeader() string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Text).
		Background(m.theme.Surface).
		Width(m.width).
		Padding(0, 2)

	title := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Mauve).Render("OSS PROGRESS")

	right := lipgloss.NewStyle().Foreground(m.theme.Subtext)
	info := right.Render(fmt.Sprintf("%d total | %.1f avg score", m.metrics.Total, m.metrics.AvgScore))

	gap := m.width - lipgloss.Width(title) - lipgloss.Width(info) - 4
	if gap < 1 {
		gap = 1
	}

	return style.Render(title + strings.Repeat(" ", gap) + info)
}

func (m ProgressModel) renderPipelineStats() string {
	padStyle := lipgloss.NewStyle().Padding(0, 2)
	sectionTitle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Sky)

	var lines []string
	lines = append(lines, padStyle.Render(sectionTitle.Render("Pipeline Breakdown")))

	if m.metrics.Total == 0 {
		dimStyle := lipgloss.NewStyle().Foreground(m.theme.Subtext)
		lines = append(lines, padStyle.Render(dimStyle.Render("No data")))
		return strings.Join(lines, "\n")
	}

	// Bar chart by status
	statusOrder := []string{"in-progress", "evaluating", "candidate", "merged", "skip"}
	statusColors := map[string]lipgloss.Color{
		"in-progress": m.theme.Green,
		"evaluating":  m.theme.Sky,
		"candidate":   m.theme.Blue,
		"merged":      m.theme.Mauve,
		"skip":        m.theme.Red,
	}
	statusLabels := map[string]string{
		"in-progress": "In-Progress",
		"evaluating":  "Evaluating",
		"candidate":   "Candidate",
		"merged":      "Merged",
		"skip":        "Skip",
	}

	maxCount := 0
	for _, st := range statusOrder {
		if c := m.metrics.ByStatus[st]; c > maxCount {
			maxCount = c
		}
	}

	labelW := 12
	barMaxW := m.width - labelW - 16
	if barMaxW < 10 {
		barMaxW = 10
	}

	for _, st := range statusOrder {
		count := m.metrics.ByStatus[st]
		barW := 0
		if maxCount > 0 {
			barW = count * barMaxW / maxCount
		}
		if barW < 1 && count > 0 {
			barW = 1
		}

		color := m.theme.Text
		if c, ok := statusColors[st]; ok {
			color = c
		}

		barStyle := lipgloss.NewStyle().Foreground(color)
		labelStyle := lipgloss.NewStyle().Foreground(m.theme.Text).Width(labelW)
		countStyle := lipgloss.NewStyle().Foreground(m.theme.Subtext)

		bar := barStyle.Render(strings.Repeat("█", barW))
		lbl := labelStyle.Render(statusLabels[st])
		cnt := countStyle.Render(fmt.Sprintf("  %d", count))

		lines = append(lines, padStyle.Render(lbl+bar+cnt))
	}

	// Summary line
	dimStyle := lipgloss.NewStyle().Foreground(m.theme.Subtext)
	summary := dimStyle.Render(fmt.Sprintf(
		"%d in-progress | %d merged | %d candidate pipeline depth",
		m.metrics.InProgress, m.metrics.Merged, m.metrics.ByStatus["candidate"],
	))
	lines = append(lines, padStyle.Render(""))
	lines = append(lines, padStyle.Render(summary))

	return strings.Join(lines, "\n")
}

func (m ProgressModel) renderScoreDistribution() string {
	padStyle := lipgloss.NewStyle().Padding(0, 2)
	sectionTitle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Sky)

	var lines []string
	lines = append(lines, padStyle.Render(sectionTitle.Render("Score Distribution (0-100)")))

	// We need to compute score buckets from metrics
	// Since we only have AvgScore in metrics, show informational text
	dimStyle := lipgloss.NewStyle().Foreground(m.theme.Subtext)
	avgStyle := lipgloss.NewStyle().Foreground(m.theme.Yellow).Bold(true)

	lines = append(lines, padStyle.Render(
		dimStyle.Render("Average score: ")+avgStyle.Render(fmt.Sprintf("%.1f", m.metrics.AvgScore)),
	))

	return strings.Join(lines, "\n")
}

func (m ProgressModel) renderHelp() string {
	style := lipgloss.NewStyle().
		Foreground(m.theme.Subtext).
		Background(m.theme.Surface).
		Width(m.width).
		Padding(0, 1)

	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Text)
	descStyle := lipgloss.NewStyle().Foreground(m.theme.Subtext)

	brand := lipgloss.NewStyle().Foreground(m.theme.Overlay).Render("oss-radar")

	keys := keyStyle.Render("↑↓") + descStyle.Render(" scroll  ") +
		keyStyle.Render("PgUp/Dn") + descStyle.Render(" page  ") +
		keyStyle.Render("Esc") + descStyle.Render(" back")

	gap := m.width - lipgloss.Width(keys) - lipgloss.Width(brand) - 2
	if gap < 1 {
		gap = 1
	}

	return style.Render(keys + strings.Repeat(" ", gap) + brand)
}
