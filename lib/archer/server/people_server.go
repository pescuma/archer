package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/samber/lo"
)

type PeopleFilters struct {
	FilterSearch string `form:"search"`
}

type PeopleListParams struct {
	GridParams
	PeopleFilters
}

type StatsPeopleParams struct {
	PeopleFilters
}

func (s *server) initPeople(r *gin.Engine) {
	r.GET("/api/people", getP[PeopleListParams](s.listPeople))
	r.GET("/api/people/:id", get(s.getPerson))
	r.GET("/api/stats/count/people", getP[StatsPeopleParams](s.countPeople))
	r.GET("/api/stats/seen/people", getP[StatsPeopleParams](s.getPeopleSeenStats))
}

func (s *server) countPeople(params *StatsPeopleParams) (any, error) {
	people := s.people.ListPeople()

	people = s.filterPeople(people, params.FilterSearch)

	return gin.H{
		"total": len(people),
	}, nil
}

func (s *server) listPeople(params *PeopleListParams) (any, error) {
	people := s.people.ListPeople()

	people = s.filterPeople(people, params.FilterSearch)

	err := s.sortPeople(people, params.Sort, params.Asc)
	if err != nil {
		return nil, err
	}

	total := len(people)

	people = paginate(people, params.Offset, params.Limit)

	var result []gin.H
	for _, r := range people {
		result = append(result, s.toPerson(r))
	}

	return gin.H{
		"data":  result,
		"total": total,
	}, nil
}

func (s *server) getPerson() (any, error) {
	return nil, nil
}

func (s *server) getPeopleSeenStats(params *StatsPeopleParams) (any, error) {
	people := s.people.ListPeople()

	people = s.filterPeople(people, params.FilterSearch)

	s1 := lo.GroupBy(people, func(person *model.Person) string {
		y, m, _ := person.FirstSeen.Date()
		return fmt.Sprintf("%04d-%02d", y, m)
	})

	s2 := lo.MapValues(s1, func(people []*model.Person, _ string) int {
		return len(people)
	})

	return s2, nil
}
