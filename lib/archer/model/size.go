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
	s.Lines += other.Lines
	s.Files += other.Files
	s.Bytes += other.Bytes

	for k, v := range other.Other {
		o, _ := s.Other[k]
		s.Other[k] = o + v
	}
}

func (s *Size) IsEmpty() bool {
	return s.Lines == 0 && s.Files == 0 && s.Bytes == 0 && len(s.Other) == 0
}

func (s *Size) Clear() {
	s.Lines = 0
	s.Files = 0
	s.Bytes = 0
	s.Other = map[string]int{}
}
