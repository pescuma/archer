package model

type MonthlyStatsLine struct {
	ID UUID

	Month        string
	RepositoryID UUID
	AuthorID     UUID
	CommitterID  UUID
	FileID       UUID
	ProjectID    *UUID

	Changes *Changes
	Blame   *Blame
}

func NewMonthlyStatsLine(month string, repositoryID UUID, authorID UUID, committerID UUID, fileID UUID, projectID *UUID) *MonthlyStatsLine {
	return &MonthlyStatsLine{
		ID:           NewUUID("sl"),
		Month:        month,
		RepositoryID: repositoryID,
		AuthorID:     authorID,
		CommitterID:  committerID,
		FileID:       fileID,
		ProjectID:    projectID,
		Changes:      NewChanges(),
		Blame:        NewBlame(),
	}
}
