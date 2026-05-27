package model

type Issue struct {
	Number     int      `yaml:"number"      json:"number"`
	Repo       string   `yaml:"repo"        json:"repo"`
	Title      string   `yaml:"title"       json:"title"`
	URL        string   `yaml:"url"         json:"url"`
	Labels     []string `yaml:"labels"      json:"labels"`
	Status     string   `yaml:"status"      json:"status"`
	Score      float64  `yaml:"score"       json:"score"`
	UpdatedAt  string   `yaml:"updated_at"  json:"updated_at"`
	FoundAt    string   `yaml:"found_at"    json:"found_at"`
	Notes      string   `yaml:"notes,omitempty"       json:"notes,omitempty"`
	PRNumber   int      `yaml:"pr_number,omitempty"   json:"pr_number,omitempty"`
	PRURL      string   `yaml:"pr_url,omitempty"      json:"pr_url,omitempty"`
	AIVerdict  string   `yaml:"ai_verdict,omitempty"  json:"ai_verdict,omitempty"`
	AIReason   string   `yaml:"ai_reason,omitempty"   json:"ai_reason,omitempty"`
	TimeEst    string   `yaml:"time_est,omitempty"    json:"time_est,omitempty"`
	ReportPath string   `yaml:"report_path,omitempty" json:"report_path,omitempty"`
}

type PipelineMetrics struct {
	Total      int
	ByStatus   map[string]int
	AvgScore   float64
	InProgress int
	Merged     int
}
