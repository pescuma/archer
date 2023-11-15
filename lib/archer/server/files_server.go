package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/samber/lo"
)

func (s *server) initFiles(r *gin.Engine) {
	r.GET("/api/files", get(s.listFiles))
	r.GET("/api/files/:id", get(s.getFile))
	r.GET("/api/stats/count/files", get(s.countFiles))
	r.GET("/api/stats/seen/files", get(s.getFilesSeenStats))
}

func (s *server) listFiles() (any, error) {
	return nil, nil
}

func (s *server) countFiles() (any, error) {
	files := s.files.ListFiles()

	return gin.H{
		"total":   len(files),
		"deleted": lo.CountBy(files, func(file *model.File) bool { return !file.Exists }),
	}, nil
}

func (s *server) getFile() (any, error) {
	return nil, nil
}

func (s *server) getFilesSeenStats() (any, error) {
	s1 := lo.GroupBy(s.files.ListFiles(), func(file *model.File) string {
		y, m, _ := file.FirstSeen.Date()
		return fmt.Sprintf("%04d-%02d", y, m)
	})
	s2 := lo.MapValues(s1, func(files []*model.File, _ string) int {
		return len(files)
	})

	return s2, nil
}
