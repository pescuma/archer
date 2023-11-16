package server

import (
	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/model"
)

func (s *server) toPersonReference(author *model.Person) gin.H {
	return gin.H{
		"id":     author.ID,
		"name":   author.Name,
		"emails": author.ListEmails(),
	}
}
