package model

type ProjectDirectory struct {
	RelativePath string
	Type         ProjectDirectoryType
	ID           UUID

	Files map[string]*ProjectFile
	Size  *Size
}

func NewProjectDirectory(relativePath string) *ProjectDirectory {
	return &ProjectDirectory{
		RelativePath: relativePath,
		ID:           NewUUID("d"),
		Files:        map[string]*ProjectFile{},
		Size:         NewSize(),
	}
}

func (d *ProjectDirectory) GetFile(relativePath string) *ProjectFile {
	result, ok := d.Files[relativePath]

	if !ok {
		result = NewProjectFile(relativePath)
		d.Files[relativePath] = result
	}

	return result
}
