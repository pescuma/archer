package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/samber/lo"
)

func (s *server) initPeople(r *gin.Engine) {
	r.GET("/api/people", get(s.listPeople))
	r.GET("/api/people/:id", get(s.getPerson))
	r.GET("/api/stats/count/people", get(s.countPeople))
	r.GET("/api/stats/seen/people", get(s.getPeopleSeenStats))
}

func (s *server) listPeople() (any, error) {
	return nil, nil
}

func (s *server) countPeople() (any, error) {
	people := s.people.ListPeople()

	return gin.H{
		"total": len(people),
	}, nil
}

func (s *server) getPerson() (any, error) {
	return nil, nil
}

func (s *server) getPeopleSeenStats() (any, error) {
	s1 := lo.GroupBy(s.people.ListPeople(), func(person *model.Person) string {
		y, m, _ := person.FirstSeen.Date()
		return fmt.Sprintf("%04d-%02d", y, m)
	})
	s2 := lo.MapValues(s1, func(people []*model.Person, _ string) int {
		return len(people)
	})

	return s2, nil
}
