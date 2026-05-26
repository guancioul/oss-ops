package model

type Issue struct {
	Number    int      `json:"number"`
	Repo      string   `json:"repo"`       // "owner/repo"
	Title     string   `json:"title"`
	URL       string   `json:"url"`
	Labels    []string `json:"labels"`
	Status    string   `json:"status"`     // candidate/evaluating/in-progress/merged/skip
	Score     float64  `json:"score"`      // 0-100 rule-based
	UpdatedAt string   `json:"updated_at"` // ISO date YYYY-MM-DD
	FoundAt   string   `json:"found_at"`   // when first discovered
	Notes     string   `json:"notes,omitempty"`
	PRNumber  int      `json:"pr_number,omitempty"`
	PRURL     string   `json:"pr_url,omitempty"`
	AIVerdict string   `json:"ai_verdict,omitempty"` // yes/maybe/no
	AIReason  string   `json:"ai_reason,omitempty"`
	TimeEst   string   `json:"time_est,omitempty"`
}

type PipelineMetrics struct {
	Total      int
	ByStatus   map[string]int
	AvgScore   float64
	InProgress int
	Merged     int
}
