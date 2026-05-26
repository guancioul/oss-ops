package screens

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/guancioul/oss-radar/internal/data"
	"github.com/guancioul/oss-radar/internal/model"
	"github.com/guancioul/oss-radar/internal/theme"
)

// PipelineClosedMsg is emitted when the pipeline screen is dismissed.
type PipelineClosedMsg struct{}

// PipelineOpenReportMsg is emitted when an issue should be opened in a viewer.
type PipelineOpenReportMsg struct {
	Path  string
	Title string
}

// PipelineOpenURLMsg is emitted when a URL should be opened in browser.
type PipelineOpenURLMsg struct {
	URL string
}

// PipelineUpdateStatusMsg requests a status update for an issue.
type PipelineUpdateStatusMsg struct {
	DataDir   string
	IssueURL  string
	NewStatus string
}

// PipelineRefreshMsg requests a full reload from disk.
type PipelineRefreshMsg struct{}

// PipelineOpenProgressMsg is emitted when the progress screen should open.
type PipelineOpenProgressMsg struct{}

// Sort modes
const (
	sortScore  = "score"
	sortDate   = "date"
	sortRepo   = "repo"
	sortStatus = "status"
)

// Filter modes
const (
	filterAll        = "all"
	filterCandidate  = "candidate"
	filterEvaluating = "evaluating"
	filterInProgress = "in-progress"
	filterMerged     = "merged"
	filterSkip       = "skip"
)

type pipelineTab struct {
	filter string
	label  string
}

var pipelineTabs = []pipelineTab{
	{filterAll, "ALL"},
	{filterCandidate, "CANDIDATE"},
	{filterEvaluating, "EVALUATING"},
	{filterInProgress, "IN-PROGRESS"},
	{filterMerged, "MERGED"},
	{filterSkip, "SKIP"},
}

var sortCycle = []string{sortScore, sortDate, sortRepo, sortStatus}

var statusOptions = []string{"candidate", "evaluating", "in-progress", "merged", "skip"}

// statusGroupOrder defines display order for grouped view.
var statusGroupOrder = []string{"in-progress", "evaluating", "candidate", "merged", "skip"}

// PipelineModel implements the OSS radar pipeline dashboard screen.
type PipelineModel struct {
	issues       []model.Issue
	filtered     []model.Issue
	metrics      model.PipelineMetrics
	cursor       int
	scrollOffset int
	sortMode     string
	activeTab    int
	viewMode     string // "grouped" or "flat"
	width, height int
	theme        theme.Theme
	dataDir      string
	// Status picker sub-state
	statusPicker bool
	statusCursor int
	// Search sub-state
	searchInput bool
	searchQuery string
}

// NewPipelineModel creates a new pipeline screen.
func NewPipelineModel(t theme.Theme, issues []model.Issue, metrics model.PipelineMetrics, dataDir string, width, height int) PipelineModel {
	m := PipelineModel{
		issues:    issues,
		metrics:   metrics,
		sortMode:  sortScore,
		activeTab: 0,
		viewMode:  "grouped",
		width:     width,
		height:    height,
		theme:     t,
		dataDir:   dataDir,
	}
	m.applyFilterAndSort()
	return m
}

// Init implements tea.Model.
func (m PipelineModel) Init() tea.Cmd {
	return nil
}

// Resize updates dimensions.
func (m *PipelineModel) Resize(width, height int) {
	m.width = width
	m.height = height
}

// Width returns the current width.
func (m PipelineModel) Width() int { return m.width }

// Height returns the current height.
func (m PipelineModel) Height() int { return m.height }

// WithReloadedData rebuilds the pipeline with fresh data while preserving UI state.
func (m PipelineModel) WithReloadedData(issues []model.Issue, metrics model.PipelineMetrics) PipelineModel {
	selectedURL := ""
	if iss, ok := m.CurrentIssue(); ok {
		selectedURL = iss.URL
	}

	reloaded := NewPipelineModel(m.theme, issues, metrics, m.dataDir, m.width, m.height)
	reloaded.sortMode = m.sortMode
	reloaded.activeTab = m.activeTab
	reloaded.viewMode = m.viewMode
	reloaded.searchQuery = m.searchQuery
	reloaded.searchInput = m.searchInput
	reloaded.applyFilterAndSort()

	for i, iss := range reloaded.filtered {
		if selectedURL != "" && iss.URL == selectedURL {
			reloaded.cursor = i
			reloaded.adjustScroll()
			return reloaded
		}
	}

	if len(reloaded.filtered) == 0 {
		reloaded.cursor = 0
		reloaded.scrollOffset = 0
		return reloaded
	}

	if m.cursor >= len(reloaded.filtered) {
		reloaded.cursor = len(reloaded.filtered) - 1
	} else if m.cursor > 0 {
		reloaded.cursor = m.cursor
	}
	reloaded.adjustScroll()
	return reloaded
}

// CurrentIssue returns the currently selected issue, if any.
func (m PipelineModel) CurrentIssue() (model.Issue, bool) {
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return model.Issue{}, false
	}
	return m.filtered[m.cursor], true
}

// Update handles input for the pipeline screen.
func (m PipelineModel) Update(msg tea.Msg) (PipelineModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.statusPicker {
			return m.handleStatusPicker(msg)
		}
		if m.searchInput {
			return m.handleSearchInput(msg)
		}
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}
	return m, nil
}

func (m PipelineModel) handleKey(msg tea.KeyMsg) (PipelineModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.searchQuery != "" {
			m.searchQuery = ""
			m.applyFilterAndSort()
			m.cursor = 0
			m.scrollOffset = 0
			return m, nil
		}
		return m, nil

	case "q":
		return m, func() tea.Msg { return PipelineClosedMsg{} }

	case "/":
		m.searchInput = true
		return m, nil

	case "down", "j":
		if len(m.filtered) > 0 {
			m.cursor++
			if m.cursor >= len(m.filtered) {
				m.cursor = len(m.filtered) - 1
			}
			m.adjustScroll()
		}

	case "up", "k":
		if len(m.filtered) > 0 {
			m.cursor--
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.adjustScroll()
		}

	case "s":
		for i, s := range sortCycle {
			if s == m.sortMode {
				m.sortMode = sortCycle[(i+1)%len(sortCycle)]
				break
			}
		}
		m.applyFilterAndSort()
		m.cursor = 0
		m.scrollOffset = 0

	case "f", "right", "l":
		m.activeTab++
		if m.activeTab >= len(pipelineTabs) {
			m.activeTab = 0
		}
		m.applyFilterAndSort()
		m.cursor = 0
		m.scrollOffset = 0

	case "left", "h":
		m.activeTab--
		if m.activeTab < 0 {
			m.activeTab = len(pipelineTabs) - 1
		}
		m.applyFilterAndSort()
		m.cursor = 0
		m.scrollOffset = 0

	case "v":
		if m.viewMode == "grouped" {
			m.viewMode = "flat"
		} else {
			m.viewMode = "grouped"
		}

	case "enter", "o":
		// Open issue URL in browser
		if iss, ok := m.CurrentIssue(); ok && iss.URL != "" {
			u := iss.URL
			return m, func() tea.Msg {
				return PipelineOpenURLMsg{URL: u}
			}
		}

	case "r":
		return m, func() tea.Msg { return PipelineRefreshMsg{} }

	case "c":
		if len(m.filtered) > 0 {
			m.statusPicker = true
			m.statusCursor = 0
		}

	case "g":
		if len(m.filtered) > 0 {
			m.cursor = 0
			m.scrollOffset = 0
		}

	case "G":
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
			m.adjustScroll()
		}

	case "pgdown", "ctrl+d":
		if len(m.filtered) > 0 {
			halfPage := m.height / 2
			if halfPage < 1 {
				halfPage = 1
			}
			m.cursor += halfPage
			if m.cursor >= len(m.filtered) {
				m.cursor = len(m.filtered) - 1
			}
			m.adjustScroll()
		}

	case "pgup", "ctrl+u":
		if len(m.filtered) > 0 {
			halfPage := m.height / 2
			if halfPage < 1 {
				halfPage = 1
			}
			m.cursor -= halfPage
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.adjustScroll()
		}
	}

	return m, nil
}

func (m PipelineModel) handleSearchInput(msg tea.KeyMsg) (PipelineModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchInput = false
		m.searchQuery = ""
		m.applyFilterAndSort()
		m.cursor = 0
		m.scrollOffset = 0
		return m, nil

	case "enter":
		m.searchInput = false
		return m, nil

	case "backspace":
		if len(m.searchQuery) > 0 {
			runes := []rune(m.searchQuery)
			m.searchQuery = string(runes[:len(runes)-1])
			m.applyFilterAndSort()
			m.cursor = 0
			m.scrollOffset = 0
		}
		return m, nil

	case "ctrl+u":
		m.searchQuery = ""
		m.applyFilterAndSort()
		m.cursor = 0
		m.scrollOffset = 0
		return m, nil
	}

	if r := msg.Runes; len(r) > 0 {
		m.searchQuery += strings.ToLower(string(r))
		m.applyFilterAndSort()
		m.cursor = 0
		m.scrollOffset = 0
		return m, nil
	}
	return m, nil
}

func (m PipelineModel) handleStatusPicker(msg tea.KeyMsg) (PipelineModel, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.statusPicker = false
		return m, nil

	case "down", "j":
		m.statusCursor++
		if m.statusCursor >= len(statusOptions) {
			m.statusCursor = len(statusOptions) - 1
		}

	case "up", "k":
		m.statusCursor--
		if m.statusCursor < 0 {
			m.statusCursor = 0
		}

	case "enter":
		m.statusPicker = false
		if iss, ok := m.CurrentIssue(); ok {
			newStatus := statusOptions[m.statusCursor]
			dir := m.dataDir
			issURL := iss.URL
			return m, func() tea.Msg {
				return PipelineUpdateStatusMsg{
					DataDir:   dir,
					IssueURL:  issURL,
					NewStatus: newStatus,
				}
			}
		}
	}
	return m, nil
}

// matchesSearch checks whether an issue matches the search query.
func matchesSearch(iss model.Issue, query string) bool {
	if query == "" {
		return true
	}
	q := strings.ToLower(query)
	if strings.Contains(strings.ToLower(iss.Title), q) {
		return true
	}
	if strings.Contains(strings.ToLower(iss.Repo), q) {
		return true
	}
	if strings.Contains(strings.ToLower(iss.Notes), q) {
		return true
	}
	for _, label := range iss.Labels {
		if strings.Contains(strings.ToLower(label), q) {
			return true
		}
	}
	return false
}

// applyFilterAndSort rebuilds the filtered list from issues.
func (m *PipelineModel) applyFilterAndSort() {
	var filtered []model.Issue

	currentFilter := pipelineTabs[m.activeTab].filter
	for _, iss := range m.issues {
		if !matchesSearch(iss, m.searchQuery) {
			continue
		}
		norm := data.NormalizeStatus(iss.Status)
		switch currentFilter {
		case filterAll:
			filtered = append(filtered, iss)
		default:
			if norm == currentFilter {
				filtered = append(filtered, iss)
			}
		}
	}

	// Sort
	switch m.sortMode {
	case sortScore:
		sort.SliceStable(filtered, func(i, j int) bool {
			return filtered[i].Score > filtered[j].Score
		})
	case sortDate:
		sort.SliceStable(filtered, func(i, j int) bool {
			return filtered[i].UpdatedAt > filtered[j].UpdatedAt
		})
	case sortRepo:
		sort.SliceStable(filtered, func(i, j int) bool {
			return strings.ToLower(filtered[i].Repo) < strings.ToLower(filtered[j].Repo)
		})
	case sortStatus:
		sort.SliceStable(filtered, func(i, j int) bool {
			return data.StatusPriority(filtered[i].Status) < data.StatusPriority(filtered[j].Status)
		})
	}

	// In grouped mode, always sort by status priority first
	if m.viewMode == "grouped" {
		sort.SliceStable(filtered, func(i, j int) bool {
			pi := data.StatusPriority(filtered[i].Status)
			pj := data.StatusPriority(filtered[j].Status)
			if pi != pj {
				return pi < pj
			}
			switch m.sortMode {
			case sortScore:
				return filtered[i].Score > filtered[j].Score
			case sortDate:
				return filtered[i].UpdatedAt > filtered[j].UpdatedAt
			case sortRepo:
				return strings.ToLower(filtered[i].Repo) < strings.ToLower(filtered[j].Repo)
			default:
				return filtered[i].Score > filtered[j].Score
			}
		})
	}

	m.filtered = filtered
}

// chromeRowsFixed returns the number of fixed chrome rows.
func (m PipelineModel) chromeRowsFixed() int {
	rows := 7 // header + tabs(2) + metrics + sortbar + help + preview baseline
	if m.searchInput || m.searchQuery != "" {
		rows++
	}
	return rows
}

const previewBudgetApprox = 5

// adjustScroll updates scrollOffset so the cursor stays visible.
func (m *PipelineModel) adjustScroll() {
	availHeight := m.height - m.chromeRowsFixed() - previewBudgetApprox
	if availHeight < 5 {
		availHeight = 5
	}
	line := m.cursorLineEstimate()
	margin := 3

	if line >= m.scrollOffset+availHeight-margin {
		m.scrollOffset = line - availHeight + margin + 1
	}
	if line < m.scrollOffset+margin {
		m.scrollOffset = line - margin
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func (m PipelineModel) cursorLineEstimate() int {
	if m.viewMode != "grouped" {
		return m.cursor
	}
	line := 0
	prevStatus := ""
	for i, iss := range m.filtered {
		norm := data.NormalizeStatus(iss.Status)
		if norm != prevStatus {
			line++ // group header
			prevStatus = norm
		}
		if i == m.cursor {
			return line
		}
		line++
	}
	return line
}

// -- View --

// View renders the pipeline screen.
func (m PipelineModel) View() string {
	header := m.renderHeader()
	tabs := m.renderTabs()
	metricsBar := m.renderMetrics()
	sortBar := m.renderSortBar()
	searchBar := m.renderSearchBar()
	body := m.renderBody()
	preview := m.renderPreview()
	help := m.renderHelp()

	// Apply scroll to body
	bodyLines := strings.Split(body, "\n")
	if m.scrollOffset > 0 && m.scrollOffset < len(bodyLines) {
		bodyLines = bodyLines[m.scrollOffset:]
	}

	// Calculate available height for body
	previewLines := strings.Count(preview, "\n") + 1
	availHeight := m.height - m.chromeRowsFixed() - previewLines
	if availHeight < 3 {
		availHeight = 3
	}
	if len(bodyLines) > availHeight {
		bodyLines = bodyLines[:availHeight]
	}
	body = strings.Join(bodyLines, "\n")

	// Status picker overlay
	if m.statusPicker {
		body = m.overlayStatusPicker(body)
	}

	sections := []string{header, tabs, metricsBar, sortBar}
	if searchBar != "" {
		sections = append(sections, searchBar)
	}
	sections = append(sections, body, preview, help)
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m PipelineModel) renderSearchBar() string {
	if !m.searchInput && m.searchQuery == "" {
		return ""
	}

	style := lipgloss.NewStyle().
		Foreground(m.theme.Text).
		Width(m.width).
		Padding(0, 2)

	prompt := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Blue).Render("/")
	queryStyle := lipgloss.NewStyle().Foreground(m.theme.Text)
	hintStyle := lipgloss.NewStyle().Foreground(m.theme.Subtext)

	display := queryStyle.Render(m.searchQuery)
	if m.searchInput {
		display += lipgloss.NewStyle().Foreground(m.theme.Blue).Render("█")
	}

	tabFiltered := m.countForFilter(pipelineTabs[m.activeTab].filter)
	matchInfo := hintStyle.Render(fmt.Sprintf("  %d/%d matching", len(m.filtered), tabFiltered))

	hint := ""
	if m.searchInput {
		hint = hintStyle.Render("   Enter: keep   Esc: cancel   Ctrl+U: clear")
	} else {
		hint = hintStyle.Render("   Esc: clear   /: edit")
	}

	return style.Render(prompt + " " + display + matchInfo + hint)
}

func (m PipelineModel) renderHeader() string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Text).
		Background(m.theme.Surface).
		Width(m.width).
		Padding(0, 2)

	right := lipgloss.NewStyle().Foreground(m.theme.Subtext)
	info := right.Render(fmt.Sprintf("%d issues | %d in-progress | %d merged",
		m.metrics.Total, m.metrics.InProgress, m.metrics.Merged))

	title := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Blue).Render("OSS RADAR")
	gap := m.width - lipgloss.Width(title) - lipgloss.Width(info) - 4
	if gap < 1 {
		gap = 1
	}

	return style.Render(title + strings.Repeat(" ", gap) + info)
}

func (m PipelineModel) renderTabs() string {
	var tabs []string
	var underParts []string

	for i, tab := range pipelineTabs {
		count := m.countForFilter(tab.filter)
		label := fmt.Sprintf(" %s (%d) ", tab.label, count)

		if i == m.activeTab {
			style := lipgloss.NewStyle().
				Bold(true).
				Foreground(m.theme.Blue).
				Padding(0, 0)
			tabs = append(tabs, style.Render(label))
			underParts = append(underParts, strings.Repeat("━", lipgloss.Width(label)))
		} else {
			style := lipgloss.NewStyle().
				Foreground(m.theme.Subtext).
				Padding(0, 0)
			tabs = append(tabs, style.Render(label))
			underParts = append(underParts, strings.Repeat("─", lipgloss.Width(label)))
		}
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	underline := lipgloss.NewStyle().Foreground(m.theme.Overlay).Render(strings.Join(underParts, ""))

	padStyle := lipgloss.NewStyle().Padding(0, 1)
	return padStyle.Render(row) + "\n" + padStyle.Render(underline)
}

func (m PipelineModel) countForFilter(filter string) int {
	count := 0
	for _, iss := range m.issues {
		norm := data.NormalizeStatus(iss.Status)
		switch filter {
		case filterAll:
			count++
		default:
			if norm == filter {
				count++
			}
		}
	}
	return count
}

func (m PipelineModel) renderMetrics() string {
	style := lipgloss.NewStyle().
		Background(m.theme.Surface).
		Width(m.width).
		Padding(0, 2)

	var parts []string
	statusColors := m.statusColorMap()

	for _, status := range statusGroupOrder {
		count, ok := m.metrics.ByStatus[status]
		if !ok || count == 0 {
			continue
		}
		color := statusColors[status]
		s := lipgloss.NewStyle().Foreground(color)
		parts = append(parts, s.Render(fmt.Sprintf("%s:%d", statusLabel(status), count)))
	}

	return style.Render(strings.Join(parts, "  "))
}

func (m PipelineModel) renderSortBar() string {
	style := lipgloss.NewStyle().
		Foreground(m.theme.Subtext).
		Width(m.width).
		Padding(0, 2)

	sortLabelText := fmt.Sprintf("[Sort: %s]", m.sortMode)
	viewLabel := fmt.Sprintf("[View: %s]", m.viewMode)
	count := fmt.Sprintf("%d shown", len(m.filtered))

	return style.Render(fmt.Sprintf("%s  %s  %s", sortLabelText, viewLabel, count))
}

func (m PipelineModel) renderBody() string {
	if len(m.filtered) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(m.theme.Subtext).
			Padding(1, 2)
		return emptyStyle.Render("No issues match this filter")
	}

	var lines []string
	prevStatus := ""
	padStyle := lipgloss.NewStyle().Padding(0, 2)

	for i, iss := range m.filtered {
		norm := data.NormalizeStatus(iss.Status)

		if m.viewMode == "grouped" && norm != prevStatus {
			count := m.countByNormStatus(norm)
			headerStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(m.theme.Subtext)
			lines = append(lines, padStyle.Render(
				headerStyle.Render(fmt.Sprintf("── %s (%d) %s",
					strings.ToUpper(statusLabel(norm)), count,
					strings.Repeat("─", max(0, m.width-30-len(statusLabel(norm)))))),
			))
			prevStatus = norm
		}

		selected := i == m.cursor
		line := m.renderIssueLine(iss, selected)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (m PipelineModel) renderIssueLine(iss model.Issue, selected bool) string {
	padStyle := lipgloss.NewStyle().Padding(0, 2)

	// Column widths
	scoreW := 5  // "85   "
	repoW := 25
	statusW := 12
	labelsW := 20
	timeEstW := 10
	// Title gets remaining space
	titleW := m.width - scoreW - repoW - statusW - labelsW - timeEstW - 14
	if titleW < 15 {
		titleW = 15
	}

	// Score with color
	scoreStyle := m.scoreStyle(iss.Score)
	scoreText := fmt.Sprintf("%.0f", iss.Score)
	score := lipgloss.NewStyle().Width(scoreW).Render(scoreStyle.Render(scoreText))

	// Repo (truncate)
	repoText := truncateRunes(iss.Repo, repoW)
	repoStyle := lipgloss.NewStyle().Foreground(m.theme.Sky).Width(repoW)

	// Title (truncate)
	titleText := truncateRunes(iss.Title, titleW)
	titleStyle := lipgloss.NewStyle().Foreground(m.theme.Text).Width(titleW)

	// Status with color
	norm := data.NormalizeStatus(iss.Status)
	statusColor := m.statusColorMap()[norm]
	statusStyle := lipgloss.NewStyle().Foreground(statusColor).Width(statusW)
	statusText := statusStyle.Render(statusLabel(norm))

	// Labels (first 2)
	labelsText := ""
	if len(iss.Labels) > 0 {
		shown := iss.Labels
		if len(shown) > 2 {
			shown = shown[:2]
		}
		labelsText = strings.Join(shown, ",")
	}
	labelsStyle := lipgloss.NewStyle().Foreground(m.theme.Yellow).Width(labelsW)

	// Time estimate
	timeEstStyle := lipgloss.NewStyle().Foreground(m.theme.Subtext).Width(timeEstW)
	timeEstText := truncateRunes(iss.TimeEst, timeEstW)

	line := fmt.Sprintf(" %s %s %s %s %s %s",
		score,
		repoStyle.Render(repoText),
		titleStyle.Render(titleText),
		statusText,
		labelsStyle.Render(truncateRunes(labelsText, labelsW)),
		timeEstStyle.Render(timeEstText),
	)

	if selected {
		selStyle := lipgloss.NewStyle().
			Background(m.theme.Overlay).
			Width(m.width - 4)
		return padStyle.Render(selStyle.Render(line))
	}
	return padStyle.Render(line)
}

func (m PipelineModel) renderPreview() string {
	iss, ok := m.CurrentIssue()
	if !ok {
		return ""
	}

	padStyle := lipgloss.NewStyle().Padding(0, 2)
	divider := lipgloss.NewStyle().Foreground(m.theme.Overlay)

	var lines []string
	lines = append(lines, padStyle.Render(divider.Render(strings.Repeat("─", m.width-4))))

	labelStyle := lipgloss.NewStyle().Foreground(m.theme.Sky).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(m.theme.Text)
	dimStyle := lipgloss.NewStyle().Foreground(m.theme.Subtext)

	// AI verdict if available
	if iss.AIVerdict != "" {
		verdictColor := m.theme.Subtext
		switch iss.AIVerdict {
		case "yes":
			verdictColor = m.theme.Green
		case "maybe":
			verdictColor = m.theme.Yellow
		case "no":
			verdictColor = m.theme.Red
		}
		verdictStyle := lipgloss.NewStyle().Foreground(verdictColor).Bold(true)
		lines = append(lines, padStyle.Render(
			labelStyle.Render("AI: ")+verdictStyle.Render(strings.ToUpper(iss.AIVerdict))+" "+valueStyle.Render(iss.AIReason)))
	}

	// Labels
	if len(iss.Labels) > 0 {
		lines = append(lines, padStyle.Render(
			labelStyle.Render("Labels: ")+dimStyle.Render(strings.Join(iss.Labels, ", "))))
	}

	// PR URL if in-progress
	if iss.PRURL != "" {
		lines = append(lines, padStyle.Render(
			labelStyle.Render("PR: ")+valueStyle.Render(iss.PRURL)))
	}

	// Notes
	if iss.Notes != "" {
		notes := truncateRunes(iss.Notes, m.width-10)
		lines = append(lines, padStyle.Render(dimStyle.Render(notes)))
	}

	// Time estimate
	if iss.TimeEst != "" {
		lines = append(lines, padStyle.Render(
			labelStyle.Render("Est: ")+dimStyle.Render(iss.TimeEst)))
	}

	// URL hint
	lines = append(lines, padStyle.Render(dimStyle.Render(fmt.Sprintf("Enter/o: open %s", iss.URL))))

	return strings.Join(lines, "\n")
}

func (m PipelineModel) renderHelp() string {
	style := lipgloss.NewStyle().
		Foreground(m.theme.Subtext).
		Background(m.theme.Surface).
		Width(m.width).
		Padding(0, 1)

	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Text)
	descStyle := lipgloss.NewStyle().Foreground(m.theme.Subtext)

	if m.statusPicker {
		return style.Render(
			keyStyle.Render("↑↓/jk") + descStyle.Render(" navigate  ") +
				keyStyle.Render("Enter") + descStyle.Render(" confirm  ") +
				keyStyle.Render("Esc") + descStyle.Render(" cancel"))
	}

	if m.searchInput {
		return style.Render(
			keyStyle.Render("type") + descStyle.Render(" filter live  ") +
				keyStyle.Render("Enter") + descStyle.Render(" keep  ") +
				keyStyle.Render("Ctrl+U") + descStyle.Render(" clear  ") +
				keyStyle.Render("Esc") + descStyle.Render(" cancel"))
	}

	brand := lipgloss.NewStyle().Foreground(m.theme.Overlay).Render("oss-radar")

	keys := keyStyle.Render("↑↓/jk") + descStyle.Render(" nav  ") +
		keyStyle.Render("←→/hl") + descStyle.Render(" tabs  ") +
		keyStyle.Render("/") + descStyle.Render(" search  ") +
		keyStyle.Render("s") + descStyle.Render(" sort  ") +
		keyStyle.Render("r") + descStyle.Render(" refresh  ") +
		keyStyle.Render("Enter/o") + descStyle.Render(" open URL  ") +
		keyStyle.Render("c") + descStyle.Render(" change status  ") +
		keyStyle.Render("v") + descStyle.Render(" view  ") +
		keyStyle.Render("q") + descStyle.Render(" quit")

	gap := m.width - lipgloss.Width(keys) - lipgloss.Width(brand) - 2
	if gap < 1 {
		gap = 1
	}

	return style.Render(keys + strings.Repeat(" ", gap) + brand)
}

func (m PipelineModel) overlayStatusPicker(body string) string {
	bodyLines := strings.Split(body, "\n")

	pickerWidth := 30
	padStyle := lipgloss.NewStyle().Padding(0, 2)
	borderStyle := lipgloss.NewStyle().
		Foreground(m.theme.Blue).
		Bold(true)

	var picker []string
	picker = append(picker, padStyle.Render(borderStyle.Render("Change status:")))

	for i, opt := range statusOptions {
		style := lipgloss.NewStyle().Foreground(m.theme.Text).Width(pickerWidth)
		if i == m.statusCursor {
			style = style.Background(m.theme.Overlay).Bold(true)
		}
		prefix := "  "
		if i == m.statusCursor {
			prefix = "> "
		}
		picker = append(picker, padStyle.Render(style.Render(prefix+opt)))
	}

	bodyLines = append(bodyLines, picker...)
	return strings.Join(bodyLines, "\n")
}

// -- Helpers --

func (m PipelineModel) scoreStyle(score float64) lipgloss.Style {
	switch {
	case score >= 80:
		return lipgloss.NewStyle().Foreground(m.theme.Green).Bold(true)
	case score >= 60:
		return lipgloss.NewStyle().Foreground(m.theme.Yellow)
	case score >= 40:
		return lipgloss.NewStyle().Foreground(m.theme.Text)
	default:
		return lipgloss.NewStyle().Foreground(m.theme.Red)
	}
}

func (m PipelineModel) statusColorMap() map[string]lipgloss.Color {
	return map[string]lipgloss.Color{
		"in-progress": m.theme.Green,
		"candidate":   m.theme.Blue,
		"evaluating":  m.theme.Sky,
		"merged":      m.theme.Mauve,
		"skip":        m.theme.Red,
	}
}

func (m PipelineModel) countByNormStatus(status string) int {
	count := 0
	for _, iss := range m.filtered {
		if data.NormalizeStatus(iss.Status) == status {
			count++
		}
	}
	return count
}

// truncateRunes truncates a string to at most maxRunes runes, appending "..." if truncated.
func truncateRunes(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-3]) + "..."
}

func statusLabel(norm string) string {
	switch norm {
	case "in-progress":
		return "In-Progress"
	case "candidate":
		return "Candidate"
	case "evaluating":
		return "Evaluating"
	case "merged":
		return "Merged"
	case "skip":
		return "Skip"
	default:
		return norm
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
