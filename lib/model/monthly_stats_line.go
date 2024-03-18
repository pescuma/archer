package model

type MonthlyStatsLine struct {
	ID ID

	Month        string
	RepositoryID UUID
	AuthorID     ID
	CommitterID  ID
	ProjectID    *UUID

	Changes *Changes
	Blame   *Blame
}

func NewMonthlyStatsLine(id ID, month string, repositoryID UUID, authorID ID, committerID ID, projectID *UUID) *MonthlyStatsLine {
	return &MonthlyStatsLine{
		ID:           id,
		Month:        month,
		RepositoryID: repositoryID,
		AuthorID:     authorID,
		CommitterID:  committerID,
		ProjectID:    projectID,
		Changes:      NewChanges(),
		Blame:        NewBlame(),
	}
}
