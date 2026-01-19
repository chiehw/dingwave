package api

import (
	"strconv"

	"dingtalk/internal/database"
	"dingtalk/internal/service"

	"github.com/labstack/echo/v4"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

type UsersResponse struct {
	Total int64           `json:"total"`
	Page  int             `json:"page"`
	Size  int             `json:"size"`
	Items []database.User `json:"items"`
}

func (h *UserHandler) GetUsers(c echo.Context) error {
	p := parsePagination(c, 50)

	items, total, err := h.userService.List(p.page, p.size)
	if err != nil {
		return Error(c, 500, err.Error())
	}

	resp := UsersResponse{
		Total: total,
		Page:  p.page,
		Size:  p.size,
		Items: items,
	}

	return Success(c, resp)
}

func (h *UserHandler) SearchUsers(c echo.Context) error {
	q := c.QueryParam("q")
	if q == "" {
		return Error(c, 400, "q is required")
	}
	size, _ := strconv.Atoi(c.QueryParam("size"))
	if size <= 0 {
		size = 20
	}

	users, err := h.userService.Search(q, size)
	if err != nil {
		return Error(c, 500, err.Error())
	}

	return Success(c, users)
}
