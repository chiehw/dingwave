package api

import (
	"strconv"

	"dingtalk/internal/database"
	"dingtalk/internal/service"

	"github.com/labstack/echo/v4"
)

type ConversationHandler struct {
	conversationService *service.ConversationService
}

func NewConversationHandler(conversationService *service.ConversationService) *ConversationHandler {
	return &ConversationHandler{conversationService: conversationService}
}

type ConversationsHomeResponse struct {
	CurrentUser *database.CurrentUser       `json:"current_user"`
	Top         service.ConversationSection `json:"top"`
	Single      service.ConversationSection `json:"single"`
	Group       service.ConversationSection `json:"group"`
}

func (h *ConversationHandler) GetConversationsHome(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 5
	}

	data, err := h.conversationService.GetHome(limit)
	if err != nil {
		return Error(c, 500, err.Error())
	}

	resp := ConversationsHomeResponse{
		CurrentUser: data.CurrentUser,
		Top:         data.Top,
		Single:      data.Single,
		Group:       data.Group,
	}

	return Success(c, resp)
}

type ConversationsResponse struct {
	Total int64                   `json:"total"`
	Page  int                     `json:"page"`
	Size  int                     `json:"size"`
	Items []database.Conversation `json:"items"`
}

func (h *ConversationHandler) GetConversations(c echo.Context) error {
	convType, _ := strconv.Atoi(c.QueryParam("type"))
	p := parsePagination(c, 20)
	order := c.QueryParam("order")
	if order == "" {
		order = "time"
	}

	items, total, err := h.conversationService.List(convType, p.page, p.size, order)
	if err != nil {
		return Error(c, 500, err.Error())
	}

	resp := ConversationsResponse{
		Total: total,
		Page:  p.page,
		Size:  p.size,
		Items: items,
	}

	return Success(c, resp)
}
