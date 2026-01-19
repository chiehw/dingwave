package service

import (
	"dingtalk/internal/database"

	"gorm.io/gorm"
)

type ConversationService struct {
	db *gorm.DB
}

func NewConversationService(db *gorm.DB) *ConversationService {
	return &ConversationService{db: db}
}

type ConversationSection struct {
	Total int64                   `json:"total"`
	Items []database.Conversation `json:"items"`
}

type ConversationsHomeData struct {
	CurrentUser *database.CurrentUser `json:"current_user"`
	Top         ConversationSection   `json:"top"`
	Single      ConversationSection   `json:"single"`
	Group       ConversationSection   `json:"group"`
}

func (s *ConversationService) GetHome(limit int) (ConversationsHomeData, error) {
	var data ConversationsHomeData

	var currentUser database.CurrentUser
	s.db.First(&currentUser)
	data.CurrentUser = &currentUser

	s.db.Model(&database.Conversation{}).Where("is_top = ?", true).Count(&data.Top.Total)
	s.db.Where("is_top = ?", true).Order("last_message_at DESC").Limit(limit).Find(&data.Top.Items)

	s.db.Model(&database.Conversation{}).Where("type = ?", database.ConversationTypeSingle).Count(&data.Single.Total)
	s.db.Where("type = ?", database.ConversationTypeSingle).Order("last_message_at DESC").Limit(limit).Find(&data.Single.Items)

	s.db.Model(&database.Conversation{}).Where("type = ?", database.ConversationTypeGroup).Count(&data.Group.Total)
	s.db.Where("type = ?", database.ConversationTypeGroup).Order("last_message_at DESC").Limit(limit).Find(&data.Group.Items)

	return data, nil
}

func (s *ConversationService) List(convType int, page, size int, order string) ([]database.Conversation, int64, error) {
	offset := (page - 1) * size
	var total int64
	var items []database.Conversation

	query := s.db.Model(&database.Conversation{})
	if convType == 0 {
		query = query.Where("is_top = ?", true)
	} else {
		query = query.Where("type = ?", convType)
	}

	query.Count(&total)

	if order == "count" {
		query = query.Order("message_count DESC")
	} else {
		query = query.Order("last_message_at DESC")
	}

	query.Limit(size).Offset(offset).Find(&items)
	return items, total, nil
}
