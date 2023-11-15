package server

import (
	"net/http"
	"reflect"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
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

func bind[T any](c *gin.Context) (T, error) {
	var result T

	err := c.BindQuery(&result)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	return result, err
}

func sortBy(col []gin.H, field string, asc bool) {
	stringType := reflect.TypeOf("")
	timeType := reflect.TypeOf(time.Time{})

	sort.Slice(col, func(i, j int) bool {
		v1 := reflect.ValueOf(col[i][field])
		v2 := reflect.ValueOf(col[j][field])

		result := false
		if v1.CanInt() {
			result = v1.Int() <= v2.Int()

		} else if v1.CanFloat() {
			result = v1.Float() <= v2.Float()

		} else if v1.CanUint() {
			result = v1.Uint() <= v2.Uint()

		} else if v1.CanConvert(stringType) {
			result = v1.Interface().(string) <= v2.Interface().(string)

		} else if v1.CanConvert(timeType) {
			result = v1.Interface().(time.Time).Before(v2.Interface().(time.Time))

		} else {
			panic("unknown type: " + v1.Type().String())
		}

		if !asc {
			result = !result
		}

		return result
	})
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
