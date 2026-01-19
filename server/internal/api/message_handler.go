package api

import (
	"strconv"

	"dingtalk/internal/database"
	"dingtalk/internal/service"

	"github.com/labstack/echo/v4"
)

type MessageHandler struct {
	messageService *service.MessageService
	userService    *service.UserService
}

func NewMessageHandler(messageService *service.MessageService, userService *service.UserService) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
		userService:    userService,
	}
}

type MessagesResponse struct {
	HasMore bool               `json:"has_more"`
	Items   []database.Message `json:"items"`
}

func (h *MessageHandler) GetConversationMessages(c echo.Context) error {
	cid := c.Param("cid")
	before, _ := strconv.ParseInt(c.QueryParam("before"), 10, 64)
	after, _ := strconv.ParseInt(c.QueryParam("after"), 10, 64)
	size, _ := strconv.Atoi(c.QueryParam("size"))
	if size <= 0 {
		size = 50
	}

	items, hasMore, err := h.messageService.GetConversationMessages(cid, before, after, size)
	if err != nil {
		return Error(c, 500, err.Error())
	}

	h.populateMessageSenders(items)

	resp := MessagesResponse{
		HasMore: hasMore,
		Items:   items,
	}

	return Success(c, resp)
}

type SearchMessagesResponse struct {
	Total int64                       `json:"total"`
	Page  int                         `json:"page"`
	Size  int                         `json:"size"`
	Items []service.SearchResultItem `json:"items"`
}

func (h *MessageHandler) SearchMessagesGlobal(c echo.Context) error {
	q := c.QueryParam("q")
	if q == "" {
		return Error(c, 400, "q is required")
	}
	p := parsePagination(c, 20)

	items, total, err := h.messageService.SearchGlobal(q, p.page, p.size)
	if err != nil {
		return Error(c, 500, err.Error())
	}

	resp := SearchMessagesResponse{
		Total: total,
		Page:  p.page,
		Size:  p.size,
		Items: items,
	}

	return Success(c, resp)
}

type SearchConvMessagesResponse struct {
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Size  int                `json:"size"`
	Items []database.Message `json:"items"`
}

func (h *MessageHandler) SearchConversationMessages(c echo.Context) error {
	cid := c.Param("cid")
	q := c.QueryParam("q")
	if q == "" {
		return Error(c, 400, "q is required")
	}
	p := parsePagination(c, 20)

	items, total, err := h.messageService.SearchInConversation(cid, q, p.page, p.size)
	if err != nil {
		return Error(c, 500, err.Error())
	}

	h.populateMessageSenders(items)

	resp := SearchConvMessagesResponse{
		Total: total,
		Page:  p.page,
		Size:  p.size,
		Items: items,
	}

	return Success(c, resp)
}

func (h *MessageHandler) populateMessageSenders(messages []database.Message) {
	var ids []int64
	for i := range messages {
		ids = append(ids, messages[i].SenderID)
	}

	userMap, _ := h.userService.GetUsersByIDs(ids)
	for i := range messages {
		messages[i].SenderName = userMap[messages[i].SenderID]
		messages[i].ContentTypeStr = messages[i].ContentType.String()
	}
}
