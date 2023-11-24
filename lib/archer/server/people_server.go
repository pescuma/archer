package server

import (
	"github.com/gin-gonic/gin"
)

func (s *server) initPeople(r *gin.Engine) {
	r.GET("/api/people", getP[ListParams](s.peopleList))
	r.GET("/api/people/:id", get(s.personGet))
	r.GET("/api/stats/count/people", getP[StatsParams](s.statsCountPeople))
	r.GET("/api/stats/seen/people", getP[StatsParams](s.statsSeenPeople))
}

func (s *server) peopleList(params *ListParams) (any, error) {
	people, err := s.listPeople(params.FilterFile, params.FilterProject, params.FilterRepo, params.FilterPerson)
	if err != nil {
		return nil, err
	}

	err = s.sortPeople(people, params.Sort, params.Asc)
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

func (s *server) statsCountPeople(params *StatsParams) (any, error) {
	people, err := s.listPeople(params.FilterFile, params.FilterProject, params.FilterRepo, params.FilterPerson)
	if err != nil {
		return nil, err
	}

	return gin.H{
		"total": len(people),
	}, nil
}

func (s *server) personGet() (any, error) {
	return nil, nil
}

func (s *server) statsSeenPeople(params *StatsParams) (any, error) {
	people, err := s.listPeople(params.FilterFile, params.FilterProject, params.FilterRepo, params.FilterPerson)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]int)
	for _, f := range people {
		y, m, _ := f.FirstSeen.Date()
		s.incSeenStats(result, y, m, "firstSeen")

		y, m, _ = f.LastSeen.Date()
		s.incSeenStats(result, y, m, "lastSeen")
	}

	return result, nil
}
