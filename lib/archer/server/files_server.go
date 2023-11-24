package server

import (
	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/samber/lo"
)

func (s *server) initFiles(r *gin.Engine) {
	r.GET("/api/files", getP[ListParams](s.filesList))
	r.GET("/api/files/:id", get(s.fileGet))
	r.GET("/api/stats/count/files", getP[StatsParams](s.statsCountFiles))
	r.GET("/api/stats/seen/files", getP[StatsParams](s.statsSeenFiles))
}

func (s *server) filesList(params *ListParams) (any, error) {
	files, err := s.listFiles(params.FilterFile, params.FilterProject, params.FilterRepo, params.FilterPerson)
	if err != nil {
		return nil, err
	}

	err = s.sortFiles(files, params.Sort, params.Asc)
	if err != nil {
		return nil, err
	}

	total := len(files)

	files = paginate(files, params.Offset, params.Limit)

	var result []gin.H
	for _, r := range files {
		result = append(result, s.toFile(r))
	}

	return gin.H{
		"data":  result,
		"total": total,
	}, nil
}

func (s *server) statsCountFiles(params *StatsParams) (any, error) {
	files, err := s.listFiles(params.FilterFile, params.FilterProject, params.FilterRepo, params.FilterPerson)
	if err != nil {
		return nil, err
	}

	return gin.H{
		"total":   len(files),
		"deleted": lo.CountBy(files, func(file *model.File) bool { return !file.Exists }),
	}, nil
}

func (s *server) fileGet() (any, error) {
	return nil, nil
}

func (s *server) statsSeenFiles(params *StatsParams) (any, error) {
	files, err := s.listFiles(params.FilterFile, params.FilterProject, params.FilterRepo, params.FilterPerson)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]int)
	for _, f := range files {
		y, m, _ := f.FirstSeen.Date()
		s.incSeenStats(result, y, m, "firstSeen")

		y, m, _ = f.LastSeen.Date()
		s.incSeenStats(result, y, m, "lastSeen")
	}

	return result, nil
}
