package entity

import (
	"context"
	"fmt"
	"gorm.io/gorm"
)

type UserRepo interface {
	GetInfoByID(ctx context.Context, filter *User) (*User, error)
	GetInfoByName(ctx context.Context, username string, user *User) error
}

type User struct {
	gorm.Model
	//ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	//Role     string `json:"role"`
	//IsValid  bool   `json:"is_valid"`
	IsActive bool `json:"is_active"`
	//OTPLevel int    `json:"otp_level"`
}

func (u *User) String() string {
	return fmt.Sprintf("%s(%s)", u.Name, u.Username)
}

func (u *User) TableName() string {
	return "users"
}

type ConnectTokenInfo struct {
	Id     string `json:"id"`
	Secret string `json:"secret"`
	//TypeName    ConnectType  `json:"type"`
	User    *User    `json:"user"`
	Actions []string `json:"actions,omitempty"`
	//Application *Application `json:"application,omitempty"`
	//Asset       *Asset       `json:"asset,omitempty"`
	ExpiredAt int64 `json:"expired_at"`
	//Gateway     Gateway      `json:"gateway,omitempty"`
	//Domain      *Domain      `json:"domain"`

	//CmdFilterRules FilterRules `json:"cmd_filter_rules,omitempty"`
	//SystemUserAuthInfo *SystemUserAuthInfo `json:"system_user"`
}
