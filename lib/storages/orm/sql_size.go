package orm

import "github.com/pescuma/archer/lib/model"

type sqlSize struct {
	Lines *int
	Files *int
	Bytes *int
	Other map[string]int `gorm:"serializer:json"`
}

func newSqlSize(s *model.Size) *sqlSize {
	return &sqlSize{
		Lines: encodeMetric(s.Lines),
		Files: encodeMetric(s.Files),
		Bytes: encodeMetric(s.Bytes),
		Other: encodeMap(s.Other),
	}
}

func (s *sqlSize) ToModel() *model.Size {
	return &model.Size{
		Lines: decodeMetric(s.Lines),
		Files: decodeMetric(s.Files),
		Bytes: decodeMetric(s.Bytes),
		Other: decodeMap(s.Other),
	}
}
