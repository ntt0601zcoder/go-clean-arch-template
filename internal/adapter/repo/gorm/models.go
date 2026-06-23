// Package gormrepo implements the AccountRepository driven port on top of GORM
// (PostgreSQL). It lives in the adapter layer, so it may import infra and
// generated code directly; the core only ever sees the outbound.AccountRepository
// interface.
package gormrepo

import "time"

// AccountModel is the GORM persistence representation of domain.Account. It is
// deliberately separate from the entity so storage concerns (column tags,
// soft-delete column, int-encoded status) never leak into the domain.
type AccountModel struct {
	ID           string     `gorm:"column:id;primaryKey"`
	Email        string     `gorm:"column:email;uniqueIndex"`
	FirstName    string     `gorm:"column:first_name"`
	LastName     string     `gorm:"column:last_name"`
	PasswordHash string     `gorm:"column:password_hash"`
	Status       int        `gorm:"column:status"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"`
	DeletedAt    *time.Time `gorm:"column:deleted_at"`
}

// TableName pins the table name so GORM does not pluralise/guess it.
func (AccountModel) TableName() string { return "accounts" }
