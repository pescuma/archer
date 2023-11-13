package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/samber/lo"
)

func (s *server) initPeople(r *gin.Engine) {
	r.GET("/api/people", s.listPeople)
	r.GET("/api/people/:id", s.getPerson)
	r.GET("/api/stats/count/people", s.countPeople)
	r.GET("/api/stats/seen/people", s.getPeopleSeenStats)
}

func (s *server) listPeople(c *gin.Context) {
}

func (s *server) countPeople(c *gin.Context) {
	people := s.people.ListPeople()

	c.JSON(http.StatusOK, gin.H{
		"total": len(people),
	})
}

func (s *server) getPerson(c *gin.Context) {
}

func (s *server) getPeopleSeenStats(c *gin.Context) {
	s1 := lo.GroupBy(s.people.ListPeople(), func(person *model.Person) string {
		y, m, _ := person.FirstSeen.Date()
		return fmt.Sprintf("%04d-%02d", y, m)
	})
	s2 := lo.MapValues(s1, func(people []*model.Person, _ string) int {
		return len(people)
	})

	c.JSON(http.StatusOK, s2)
}
