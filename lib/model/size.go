package model

type Size struct {
	Lines int
	Files int
	Bytes int
	Other map[string]int
}

func NewSize() *Size {
	return &Size{
		Lines: -1,
		Files: -1,
		Bytes: -1,
		Other: map[string]int{},
	}
}

func (s *Size) Add(other *Size) {
	s.Lines = add(s.Lines, other.Lines)
	s.Files = add(s.Files, other.Files)
	s.Bytes = add(s.Bytes, other.Bytes)

	for k, o := range other.Other {
		v, _ := s.Other[k]
		s.Other[k] = v + o
	}
}

func (s *Size) IsEmpty() bool {
	return s.Lines == -1 && s.Files == -1 && s.Bytes == -1 && len(s.Other) == 0
}

func (s *Size) Clear() {
	s.Lines = 0
	s.Files = 0
	s.Bytes = 0
	s.Other = map[string]int{}
}
