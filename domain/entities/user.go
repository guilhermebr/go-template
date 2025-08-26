package entities

import (
	"time"

	"github.com/gofrs/uuid/v5"
)

type AccountType string

const (
	AccountTypeUser       AccountType = "user"
	AccountTypeAdmin      AccountType = "admin"
	AccountTypeSuperAdmin AccountType = "super_admin"
)

func (a AccountType) String() string {
	return string(a)
}

type User struct {
	ID             uuid.UUID   `json:"id" db:"id"`
	Email          string      `json:"email" db:"email"`
	AuthProvider   string      `json:"auth_provider" db:"auth_provider"`
	AuthProviderID string      `json:"-" db:"auth_provider_id"`
	AccountType    AccountType `json:"account_type" db:"account_type"`
	CreatedAt      time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at" db:"updated_at"`
}

func (u *User) IsValid() bool {
	return u.Email != "" && u.AuthProvider != "" && u.ID != uuid.Nil
}

type UserStats struct {
	TotalUsers      int64
	AdminUsers      int64
	SuperAdminUsers int64
	RegularUsers    int64
	RecentSignups   int64
}

type ListUsersParams struct {
	Limit  int32
	Offset int32
}
