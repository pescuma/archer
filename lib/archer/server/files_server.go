package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/samber/lo"
)

func (s *server) initFiles(r *gin.Engine) {
	r.GET("/api/files", s.listFiles)
	r.GET("/api/files/:id", s.getFile)
	r.GET("/api/stats/count/files", s.countFiles)
	r.GET("/api/stats/monthly/files", s.getFilesMonthlyStats)
}

func (s *server) listFiles(c *gin.Context) {
}

func (s *server) countFiles(c *gin.Context) {
	files := s.files.ListFiles()

	c.JSON(http.StatusOK, gin.H{
		"total":   len(files),
		"deleted": lo.CountBy(files, func(file *model.File) bool { return !file.Exists }),
	})
}

func (s *server) getFile(c *gin.Context) {
}

func (s *server) getFilesMonthlyStats(c *gin.Context) {
	s1 := lo.GroupBy(s.files.ListFiles(), func(file *model.File) string {
		y, m, _ := file.FirstSeen.Date()
		return fmt.Sprintf("%04d-%02d", y, m)
	})
	s2 := lo.MapValues(s1, func(files []*model.File, _ string) int {
		return len(files)
	})

	c.JSON(http.StatusOK, s2)
}
