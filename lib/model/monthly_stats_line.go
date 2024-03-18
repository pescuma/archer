package model

type MonthlyStatsLine struct {
	ID ID

	Month        string
	RepositoryID ID
	AuthorID     ID
	CommitterID  ID
	ProjectID    *ID

	Changes *Changes
	Blame   *Blame
}

func NewMonthlyStatsLine(id ID, month string, repositoryID ID, authorID ID, committerID ID, projectID *ID) *MonthlyStatsLine {
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
