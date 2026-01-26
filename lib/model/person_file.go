package model

type PersonFile struct {
	PersonID ID
	FileID   ID

	SeenAt *SeenAt
}

func NewPersonFile(personID ID, fileID ID) *PersonFile {
	return &PersonFile{
		PersonID: personID,
		FileID:   fileID,
		SeenAt:   NewSeenAt(),
	}
}
