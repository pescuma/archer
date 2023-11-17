package server

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/model"
	"github.com/pescuma/archer/lib/archer/utils"
	"golang.org/x/exp/constraints"
)

type GridParams struct {
	Sort   string `form:"sort"`
	Asc    *bool  `form:"asc"`
	Offset *int   `form:"offset"`
	Limit  *int   `form:"limit"`
}

var errorNotFound error

func init() {
	errorNotFound = fmt.Errorf("not found")
}

func sendError(c *gin.Context, err error) {
	switch err {
	case errorNotFound:
		c.String(http.StatusNotFound, "")
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func get(f func() (any, error)) func(c *gin.Context) {
	return func(c *gin.Context) {
		result, err := f()
		if err != nil {
			sendError(c, err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func getP[P any](f func(*P) (any, error)) func(c *gin.Context) {
	return func(c *gin.Context) {
		var params P

		err := c.ShouldBindQuery(&params)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result, err := f(&params)
		if err != nil {
			sendError(c, err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func patchP[P any](f func(*P) (any, error)) func(c *gin.Context) {
	return func(c *gin.Context) {
		var params P

		err := c.ShouldBindUri(&params)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err = c.ShouldBindJSON(&params)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result, err := f(&params)
		if err != nil {
			sendError(c, err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func sortBy[T any, R constraints.Ordered](col []T, get func(T) R, asc bool) error {
	if asc {
		sort.Slice(col, func(i, j int) bool {
			return get(col[i]) <= get(col[j])
		})
	} else {
		sort.Slice(col, func(i, j int) bool {
			return get(col[i]) >= get(col[j])
		})
	}
	return nil
}

func paginate[T any](col []T, offset, limit *int) []T {
	if offset != nil {
		if *offset > len(col) {
			return []T{}
		}

		col = col[*offset:]
	}

	if limit != nil && *limit < len(col) {
		col = col[:*limit]
	}

	return col
}

func prepareToSearch(s string) string {
	s = strings.TrimSpace(s)

	if s != "" {
		s = strings.ToLower(s)
	}

	return s
}

func encodeMetric(v int) *int {
	return utils.IIf(v == -1, nil, &v)
}

func encodeDate(v time.Time) *time.Time {
	empty := time.Time{}
	return utils.IIf(v == empty, nil, &v)
}

func (s *server) toSize(i *model.Size) gin.H {
	return gin.H{
		"lines": encodeMetric(i.Lines),
		"files": encodeMetric(i.Files),
		"bytes": encodeMetric(i.Bytes),
		"other": i.Other,
	}
}

func (s *server) toChanges(i *model.Changes) gin.H {
	return gin.H{
		"total":         encodeMetric(i.Total),
		"in6Months":     encodeMetric(i.In6Months),
		"linesModified": encodeMetric(i.LinesModified),
		"linesAdded":    encodeMetric(i.LinesAdded),
		"linesDeleted":  encodeMetric(i.LinesDeleted),
	}
}
