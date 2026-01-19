package api

import (
	"strconv"

	"github.com/labstack/echo/v4"
)

type pagination struct {
	page   int
	size   int
	offset int
}

func parsePagination(c echo.Context, defaultSize int) pagination {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page <= 0 {
		page = 1
	}
	size, _ := strconv.Atoi(c.QueryParam("size"))
	if size <= 0 {
		size = defaultSize
	}
	return pagination{
		page:   page,
		size:   size,
		offset: (page - 1) * size,
	}
}
