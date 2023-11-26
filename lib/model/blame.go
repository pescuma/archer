package model

type Blame struct {
	Code    int
	Comment int
	Blank   int
}

func NewBlame() *Blame {
	return &Blame{
		Code:    -1,
		Comment: -1,
		Blank:   -1,
	}
}

func (s *Blame) Total() int {
	result := -1
	result = add(result, s.Code)
	result = add(result, s.Comment)
	result = add(result, s.Blank)
	return result
}

func (s *Blame) Add(other *Blame) {
	s.Code = add(s.Code, other.Code)
	s.Comment = add(s.Comment, other.Comment)
	s.Blank = add(s.Blank, other.Blank)
}

func (s *Blame) IsEmpty() bool {
	return s.Code == -1 && s.Comment == -1 && s.Blank == -1
}

func (s *Blame) Clear() {
	s.Code = 0
	s.Comment = 0
	s.Blank = 0
}

func (s *Blame) Reset() {
	s.Code = -1
	s.Comment = -1
	s.Blank = -1
}
