package models

import (
	"fmt"
	"strings"
	"time"
)

// PrimaryKey 是全局自增主键的类型别名。
type PrimaryKey = uint64

// Varchar 将字符串原样返回，语义上表示数据库 VARCHAR 赋值。
func Varchar(s string) string { return s }

// Role 是受约束的字符串类型，表示用户权限等级。
type Role string

const (
	RoleStudent Role = "student"
	RoleAdmin   Role = "admin"
)

// User 对应数据库 users 表。
type User struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Username      string    `gorm:"size:64;uniqueIndex"      json:"username"`
	Email         string    `gorm:"size:128;uniqueIndex"     json:"email"`
	PasswordHash  string    `gorm:"size:255"                 json:"-"`
	Role          Role      `gorm:"size:32"                  json:"role"`
	RememberToken string    `gorm:"size:255"                 json:"-"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// Valid 报告该角色是否为受支持的值之一。
func (r Role) Valid() bool {
	switch r {
	case RoleStudent, RoleAdmin:
		return true
	default:
		return false
	}
}

// ParseRole 对原始角色值进行规范化并验证。
func ParseRole(value string) (Role, error) {
	role := Role(strings.ToLower(strings.TrimSpace(value)))
	if role.Valid() {
		return role, nil
	}
	return "", fmt.Errorf("unknown role: %s", value)
}
