package model

type MonthlyStatsLine struct {
	ID UUID

	Month        string
	RepositoryID UUID
	AuthorID     UUID
	CommitterID  UUID
	ProjectID    *UUID

	Changes *Changes
	Blame   *Blame
}

func NewMonthlyStatsLine(month string, repositoryID UUID, authorID UUID, committerID UUID, projectID *UUID) *MonthlyStatsLine {
	return &MonthlyStatsLine{
		ID:           NewUUID("sl"),
		Month:        month,
		RepositoryID: repositoryID,
		AuthorID:     authorID,
		CommitterID:  committerID,
		ProjectID:    projectID,
		Changes:      NewChanges(),
		Blame:        NewBlame(),
	}
}
