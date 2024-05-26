package server

import (
	"github.com/gin-gonic/gin"
)

func (s *server) initFiles(r *gin.Engine) {
	r.GET("/api/files", getP[ListParams](s.filesList))
	r.GET("/api/files/:id", get(s.fileGet))
	r.GET("/api/stats/count/files", getP[StatsParams](s.statsCountFiles))
	r.GET("/api/stats/seen/files", getP[StatsParams](s.statsSeenFiles))
}

func (s *server) filesList(params *ListParams) (any, error) {
	files, err := s.listFiles(&params.Filters)
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
	files, err := s.listFiles(&params.Filters)
	if err != nil {
		return nil, err
	}

	exists := 0
	deleted := 0
	lines := 0
	for _, file := range files {
		if file.Exists {
			lines += file.Size.Lines
			exists++
		} else {
			deleted++
		}
	}

	return gin.H{
		"exists":  exists,
		"deleted": deleted,
		"lines":   lines,
	}, nil
}

func (s *server) fileGet() (any, error) {
	return nil, nil
}

func (s *server) statsSeenFiles(params *StatsParams) (any, error) {
	files, err := s.listFiles(&params.Filters)
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
