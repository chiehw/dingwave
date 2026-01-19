package service

import (
	"dingtalk/internal/database"

	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) List(page, size int) ([]database.User, int64, error) {
	offset := (page - 1) * size
	var total int64
	var items []database.User

	s.db.Model(&database.User{}).Count(&total)
	s.db.Limit(size).Offset(offset).Find(&items)

	return items, total, nil
}

func (s *UserService) Search(query string, size int) ([]database.User, error) {
	var users []database.User
	s.db.Where("nickname LIKE ? OR email LIKE ?", "%"+query+"%", "%"+query+"%").Limit(size).Find(&users)
	return users, nil
}

func (s *UserService) GetUsersByIDs(ids []int64) (map[int64]string, error) {
	userMap := make(map[int64]string)
	for _, id := range ids {
		if _, exists := userMap[id]; !exists {
			var user database.User
			if err := s.db.First(&user, id).Error; err == nil {
				userMap[id] = user.Nickname
			}
		}
	}
	return userMap, nil
}
