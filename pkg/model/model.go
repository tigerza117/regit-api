package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

type Users []User

type User struct {
	ID uuid.UUID `json:"-" gorm:"type:string;size:36;primaryKey;"`

	UserID string `json:"-"`

	Email     string `json:"-"`
	FirstName string `json:"-"`
	LastName  string `json:"-"`
	NickName  string `json:"-"`

	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}

type UserResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func (m *User) Response() *UserResponse {
	return &UserResponse{
		ID:   m.ID,
		Name: m.NickName,
	}
}

type Messages []*Message

func (m Messages) Response() []*MessageResponse {
	return lo.Map[*Message, *MessageResponse](m, func(item *Message, index int) *MessageResponse {
		return item.Response()
	})
}

type Message struct {
	ID     uuid.UUID `json:"-" gorm:"type:string;size:36;primaryKey;"`
	UserID uuid.UUID `json:"-"`
	User   *User     `json:"-"`

	Message string `json:"message"`

	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (m *Message) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return
}

type MessageResponse struct {
	ID      uuid.UUID `json:"id"`
	Message string    `json:"message"`
}

func (m *Message) Response() *MessageResponse {
	return &MessageResponse{
		ID:      m.ID,
		Message: m.Message,
	}
}
