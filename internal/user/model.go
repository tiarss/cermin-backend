package user

import "time"

type User struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	Name         string    `gorm:"size:100;not null"`
	Email        string    `gorm:"size:150;not null;uniqueIndex"`
	PasswordHash *string   `gorm:"type:text"`
	AuthProvider string    `gorm:"size:30;not null;default:local"`
	GoogleID     *string   `gorm:"size:255;uniqueIndex"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
}

type PublicUser struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	AuthProvider string `json:"auth_provider"`
}

func ToPublicUser(user *User) PublicUser {
	return PublicUser{
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		AuthProvider: user.AuthProvider,
	}
}
