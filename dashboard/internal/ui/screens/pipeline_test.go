package screens

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/guancioul/oss-radar/internal/model"
	"github.com/guancioul/oss-radar/internal/theme"
)

func tabIndexForFilter(t *testing.T, filter string) int {
	t.Helper()

	for i, tab := range pipelineTabs {
		if tab.filter == filter {
			return i
		}
	}

	t.Fatalf("expected pipeline tabs to include filter %q", filter)
	return -1
}

func TestWithReloadedDataPreservesStateAndSelection(t *testing.T) {
	initialIssues := []model.Issue{
		{
			Number: 1,
			Repo:   "strimzi/strimzi-kafka-operator",
			Title:  "Fix reconciler timeout",
			URL:    "https://github.com/strimzi/strimzi-kafka-operator/issues/1",
			Status: "candidate",
			Score:  75.0,
		},
		{
			Number: 2,
			Repo:   "open-telemetry/otel-arrow",
			Title:  "Add OTLP exporter support",
			URL:    "https://github.com/open-telemetry/otel-arrow/issues/2",
			Status: "evaluating",
			Score:  80.0,
		},
	}

	pm := NewPipelineModel(
		theme.NewTheme("catppuccin-mocha"),
		initialIssues,
		model.PipelineMetrics{Total: len(initialIssues)},
		"..",
		120,
		40,
	)
	pm.sortMode = sortRepo
	pm.activeTab = 0
	pm.viewMode = "flat"
	pm.applyFilterAndSort()
	// In flat+sortRepo order: [0]=open-telemetry (o<s), [1]=strimzi/strimzi-kafka-operator
	// cursor=0 points to the otel-arrow issue (issues/2)
	pm.cursor = 0

	refreshedIssues := []model.Issue{
		initialIssues[0],
		initialIssues[1],
		{
			Number: 3,
			Repo:   "strimzi/strimzi-kafka-bridge",
			Title:  "Improve HTTP bridge",
			URL:    "https://github.com/strimzi/strimzi-kafka-bridge/issues/3",
			Status: "candidate",
			Score:  65.0,
		},
	}

	reloaded := pm.WithReloadedData(refreshedIssues, model.PipelineMetrics{Total: len(refreshedIssues)})

	if reloaded.sortMode != sortRepo {
		t.Fatalf("expected sort mode %q, got %q", sortRepo, reloaded.sortMode)
	}
	if reloaded.viewMode != "flat" {
		t.Fatalf("expected view mode to stay flat, got %q", reloaded.viewMode)
	}
	if got := len(reloaded.filtered); got != 3 {
		t.Fatalf("expected 3 filtered issues after refresh, got %d", got)
	}
	// The otel-arrow issue (issues/2) should still be selected after reload
	if iss, ok := reloaded.CurrentIssue(); !ok || iss.URL != initialIssues[1].URL {
		t.Fatalf("expected selection to stay on otel-arrow issue, got %+v (ok=%v)", iss, ok)
	}
}

func TestRenderIssueLineIncludesRepoAndScore(t *testing.T) {
	pm := NewPipelineModel(
		theme.NewTheme("catppuccin-mocha"),
		nil,
		model.PipelineMetrics{},
		"..",
		120,
		40,
	)

	line := pm.renderIssueLine(model.Issue{
		Number: 42,
		Repo:   "strimzi/strimzi-kafka-operator",
		Title:  "Fix reconciler",
		Status: "candidate",
		Score:  85.0,
	}, false)

	if line == "" {
		t.Fatal("expected non-empty rendered line")
	}
}

func TestSearchFiltersByTitleRepoAndNotes(t *testing.T) {
	issues := []model.Issue{
		{Repo: "strimzi/strimzi-kafka-operator", Title: "Fix reconciler", Status: "candidate", Score: 75.0, Notes: "operator core"},
		{Repo: "open-telemetry/otel-arrow", Title: "Add OTLP exporter", Status: "evaluating", Score: 80.0, Notes: "metrics pipeline"},
		{Repo: "strimzi/strimzi-kafka-bridge", Title: "Improve HTTP bridge", Status: "candidate", Score: 65.0, Notes: "bridge refactor"},
	}

	pm := NewPipelineModel(theme.NewTheme("catppuccin-mocha"), issues, model.PipelineMetrics{Total: len(issues)}, "..", 120, 40)
	pm.activeTab = tabIndexForFilter(t, filterAll)

	// Match by repo substring (case-insensitive).
	pm.searchQuery = "otel-arrow"
	pm.applyFilterAndSort()
	if len(pm.filtered) != 1 || pm.filtered[0].Repo != "open-telemetry/otel-arrow" {
		t.Fatalf("expected 1 match for 'otel-arrow', got %+v", pm.filtered)
	}

	// Match by title substring.
	pm.searchQuery = "reconciler"
	pm.applyFilterAndSort()
	if len(pm.filtered) != 1 || pm.filtered[0].Title != "Fix reconciler" {
		t.Fatalf("expected 1 match for 'reconciler', got %+v", pm.filtered)
	}

	// Match by notes substring.
	pm.searchQuery = "bridge refactor"
	pm.applyFilterAndSort()
	if len(pm.filtered) != 1 || pm.filtered[0].Repo != "strimzi/strimzi-kafka-bridge" {
		t.Fatalf("expected 1 match for notes 'bridge refactor', got %+v", pm.filtered)
	}

	// Empty query restores everything.
	pm.searchQuery = ""
	pm.applyFilterAndSort()
	if len(pm.filtered) != len(issues) {
		t.Fatalf("expected empty query to restore all rows, got %d/%d", len(pm.filtered), len(issues))
	}
}

func TestSearchComposesWithActiveTab(t *testing.T) {
	issues := []model.Issue{
		{Repo: "strimzi/strimzi-kafka-operator", Title: "Fix reconciler", Status: "candidate", Score: 75.0},
		{Repo: "strimzi/strimzi-kafka-operator", Title: "Improve logging", Status: "evaluating", Score: 70.0},
		{Repo: "open-telemetry/otel-arrow", Title: "Add OTLP exporter", Status: "candidate", Score: 80.0},
	}

	pm := NewPipelineModel(theme.NewTheme("catppuccin-mocha"), issues, model.PipelineMetrics{Total: len(issues)}, "..", 120, 40)
	pm.activeTab = tabIndexForFilter(t, filterCandidate)
	pm.searchQuery = "strimzi"
	pm.applyFilterAndSort()

	if len(pm.filtered) != 1 || pm.filtered[0].Title != "Fix reconciler" {
		t.Fatalf("expected candidate+strimzi to leave only reconciler issue, got %+v", pm.filtered)
	}
}

func TestSearchIsCaseInsensitive(t *testing.T) {
	issues := []model.Issue{
		{Repo: "strimzi/strimzi-kafka-operator", Title: "Fix Reconciler", Status: "candidate", Score: 75.0},
	}

	pm := NewPipelineModel(theme.NewTheme("catppuccin-mocha"), issues, model.PipelineMetrics{Total: len(issues)}, "..", 120, 40)
	for _, q := range []string{"reconciler", "RECONCILER", "ReCoNcIlEr"} {
		pm.searchQuery = q
		pm.applyFilterAndSort()
		if len(pm.filtered) != 1 {
			t.Fatalf("expected case-insensitive match for %q, got %d rows", q, len(pm.filtered))
		}
	}
}

func TestSearchEnterCommitsAndEscClearsCommittedQuery(t *testing.T) {
	issues := []model.Issue{
		{Repo: "strimzi/strimzi-kafka-operator", Title: "Fix reconciler", Status: "candidate", Score: 75.0},
		{Repo: "open-telemetry/otel-arrow", Title: "Add OTLP exporter", Status: "evaluating", Score: 80.0},
	}

	pm := NewPipelineModel(theme.NewTheme("catppuccin-mocha"), issues, model.PipelineMetrics{Total: len(issues)}, "..", 120, 40)

	// Open input and type "strimzi".
	pm, _ = pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !pm.searchInput {
		t.Fatal("expected `/` to open search input")
	}
	for _, r := range "strimzi" {
		pm, _ = pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if pm.searchQuery != "strimzi" {
		t.Fatalf("expected query to live-update to 'strimzi', got %q", pm.searchQuery)
	}
	if len(pm.filtered) != 1 || pm.filtered[0].Repo != "strimzi/strimzi-kafka-operator" {
		t.Fatalf("expected live filter to leave only strimzi, got %+v", pm.filtered)
	}

	// Enter commits — input closes, query stays.
	pm, _ = pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if pm.searchInput {
		t.Fatal("expected Enter to close input")
	}
	if pm.searchQuery != "strimzi" {
		t.Fatalf("expected Enter to keep committed query, got %q", pm.searchQuery)
	}

	// Esc on a committed query clears the search and restores the full list.
	pm, _ = pm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if pm.searchQuery != "" {
		t.Fatalf("expected Esc to clear committed query, got %q", pm.searchQuery)
	}
	if len(pm.filtered) != len(issues) {
		t.Fatalf("expected Esc to restore full list, got %d/%d", len(pm.filtered), len(issues))
	}
}

func TestSearchEscInInputCancelsAndClears(t *testing.T) {
	issues := []model.Issue{
		{Repo: "strimzi/strimzi-kafka-operator", Title: "Fix reconciler", Status: "candidate", Score: 75.0},
		{Repo: "open-telemetry/otel-arrow", Title: "Add OTLP", Status: "candidate", Score: 70.0},
		{Repo: "strimzi/strimzi-kafka-bridge", Title: "Bridge fix", Status: "candidate", Score: 65.0},
	}

	pm := NewPipelineModel(theme.NewTheme("catppuccin-mocha"), issues, model.PipelineMetrics{Total: len(issues)}, "..", 120, 40)
	pm.searchInput = true
	pm.searchQuery = "stri"
	pm.applyFilterAndSort()
	if len(pm.filtered) != 2 {
		t.Fatalf("setup expected 2 rows matching 'stri', got %d", len(pm.filtered))
	}

	pm, _ = pm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if pm.searchInput {
		t.Fatal("expected Esc in input mode to close input")
	}
	if pm.searchQuery != "" {
		t.Fatalf("expected Esc in input mode to clear in-progress query, got %q", pm.searchQuery)
	}
	if len(pm.filtered) != len(issues) {
		t.Fatalf("expected Esc to re-expand filtered list to %d rows, got %d", len(issues), len(pm.filtered))
	}
}

func TestSearchResetsCursorOnQueryChange(t *testing.T) {
	issues := []model.Issue{
		{Repo: "strimzi/a", Title: "Issue A", Status: "candidate", Score: 70.0},
		{Repo: "strimzi/b", Title: "Issue B", Status: "candidate", Score: 71.0},
		{Repo: "strimzi/c", Title: "Issue C", Status: "candidate", Score: 72.0},
	}

	pm := NewPipelineModel(theme.NewTheme("catppuccin-mocha"), issues, model.PipelineMetrics{Total: len(issues)}, "..", 120, 40)
	pm.cursor = 2

	pm, _ = pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	pm, _ = pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	if pm.cursor != 0 {
		t.Fatalf("expected cursor to reset to 0 on query change, got %d", pm.cursor)
	}
	if pm.scrollOffset != 0 {
		t.Fatalf("expected scrollOffset to reset to 0 on query change, got %d", pm.scrollOffset)
	}
}

func TestSearchStatePreservedAcrossReload(t *testing.T) {
	initial := []model.Issue{
		{Repo: "strimzi/strimzi-kafka-operator", Title: "Fix reconciler", Status: "candidate", Score: 75.0, URL: "https://github.com/strimzi/strimzi-kafka-operator/issues/1"},
		{Repo: "open-telemetry/otel-arrow", Title: "Add OTLP", Status: "candidate", Score: 70.0, URL: "https://github.com/open-telemetry/otel-arrow/issues/2"},
	}

	pm := NewPipelineModel(theme.NewTheme("catppuccin-mocha"), initial, model.PipelineMetrics{Total: len(initial)}, "..", 120, 40)
	pm.searchQuery = "strimzi"
	pm.applyFilterAndSort()

	refreshed := append([]model.Issue{}, initial...)
	refreshed = append(refreshed, model.Issue{Repo: "strimzi/strimzi-kafka-bridge", Title: "Bridge fix", Status: "candidate", Score: 65.0})

	reloaded := pm.WithReloadedData(refreshed, model.PipelineMetrics{Total: len(refreshed)})

	if reloaded.searchQuery != "strimzi" {
		t.Fatalf("expected refresh to preserve search query, got %q", reloaded.searchQuery)
	}
	if len(reloaded.filtered) != 2 {
		t.Fatalf("expected refresh+search to keep filter applied (2 strimzi matches), got %d", len(reloaded.filtered))
	}
}

func TestSkipTabFiltersCorrectly(t *testing.T) {
	issues := []model.Issue{
		{Repo: "strimzi/a", Title: "Skipped issue", Status: "skip", Score: 30.0},
		{Repo: "strimzi/b", Title: "Active issue", Status: "candidate", Score: 70.0},
		{Repo: "strimzi/c", Title: "In progress", Status: "in-progress", Score: 80.0},
	}

	pm := NewPipelineModel(
		theme.NewTheme("catppuccin-mocha"),
		issues,
		model.PipelineMetrics{Total: len(issues)},
		"..",
		120,
		40,
	)

	pm.activeTab = tabIndexForFilter(t, filterSkip)
	pm.applyFilterAndSort()
	if len(pm.filtered) != 1 || pm.filtered[0].Status != "skip" {
		t.Fatalf("expected skip tab to isolate skip rows, got %+v", pm.filtered)
	}

	pm.activeTab = tabIndexForFilter(t, filterInProgress)
	pm.applyFilterAndSort()
	if len(pm.filtered) != 1 || pm.filtered[0].Status != "in-progress" {
		t.Fatalf("expected in-progress tab to isolate in-progress rows, got %+v", pm.filtered)
	}
}

// Regression: with no committed search query, Esc must NOT close the screen.
func TestEscWithoutQueryIsNoOp(t *testing.T) {
	issues := []model.Issue{
		{Repo: "strimzi/a", Title: "Fix reconciler", Status: "candidate", Score: 75.0},
	}

	pm := NewPipelineModel(theme.NewTheme("catppuccin-mocha"), issues, model.PipelineMetrics{Total: len(issues)}, "..", 120, 40)
	if pm.searchQuery != "" {
		t.Fatalf("setup expected empty search query, got %q", pm.searchQuery)
	}

	pm, cmd := pm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		if msg := cmd(); msg != nil {
			if _, ok := msg.(PipelineClosedMsg); ok {
				t.Fatalf("expected Esc with no query to be a no-op, got PipelineClosedMsg")
			}
			t.Fatalf("expected Esc with no query to return nil cmd, got %T", msg)
		}
	}
	if pm.searchInput {
		t.Fatal("Esc with no query should not toggle searchInput")
	}
}
