package server

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/pescuma/archer/lib/archer/utils"
	"golang.org/x/exp/constraints"
)

func getP[P any](f func(*P) (any, error)) func(c *gin.Context) {
	return func(c *gin.Context) {
		var params P

		err := c.BindQuery(&params)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result, err := f(&params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func get(f func() (any, error)) func(c *gin.Context) {
	return func(c *gin.Context) {
		result, err := f()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

func encodeMetric(v int) *int {
	return utils.IIf(v == -1, nil, &v)
}
