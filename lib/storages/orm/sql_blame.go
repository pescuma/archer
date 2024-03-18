package orm

import "github.com/pescuma/archer/lib/model"

type sqlBlame struct {
	Code    *int
	Comment *int
	Blank   *int
}

func newSqlBlame(b *model.Blame) *sqlBlame {
	return &sqlBlame{
		Code:    encodeMetric(b.Code),
		Comment: encodeMetric(b.Comment),
		Blank:   encodeMetric(b.Blank),
	}
}

func (s *sqlBlame) ToModel() *model.Blame {
	return &model.Blame{
		Code:    decodeMetric(s.Code),
		Comment: decodeMetric(s.Comment),
		Blank:   decodeMetric(s.Blank),
	}
}
