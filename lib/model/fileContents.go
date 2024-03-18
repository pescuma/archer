package model

type FileContents struct {
	FileID ID
	Lines  []*FileLine
}

func NewFileContents(fileID ID) *FileContents {
	return &FileContents{FileID: fileID}
}

func (fc *FileContents) AppendLine() *FileLine {
	result := &FileLine{
		Line: len(fc.Lines) + 1,
	}

	fc.Lines = append(fc.Lines, result)

	return result
}
