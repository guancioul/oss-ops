package model

type PRStatus string

const (
	PRStatusOpen     PRStatus = "open"
	PRStatusMerged   PRStatus = "merged"
	PRStatusRejected PRStatus = "rejected"
)

type PR struct {
	Number int
	URL    string
	Status PRStatus
}
