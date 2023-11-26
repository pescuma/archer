package model

type FileContents struct {
	FileID UUID
	Lines  []*FileLine
}

func NewFileContents(fileID UUID) *FileContents {
	return &FileContents{FileID: fileID}
}

func (fc *FileContents) AppendLine() *FileLine {
	result := &FileLine{
		Line: len(fc.Lines) + 1,
	}

	fc.Lines = append(fc.Lines, result)

	return result
}
