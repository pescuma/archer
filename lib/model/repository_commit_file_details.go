package model

type RepositoryCommitFileDetails struct {
	FileID    ID
	Hash      string
	OldIDs    map[ID]ID
	OldHashes map[ID]string
}

func NewRepositoryCommitFileDetails(fileID ID) *RepositoryCommitFileDetails {
	return &RepositoryCommitFileDetails{
		FileID:    fileID,
		OldIDs:    make(map[ID]ID),
		OldHashes: make(map[ID]string),
	}
}
