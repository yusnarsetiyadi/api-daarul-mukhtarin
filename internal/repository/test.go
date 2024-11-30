package repository

import (
	"daarul_mukhtarin/internal/abstraction"

	"gorm.io/gorm"
)

type Test interface {
}

type test struct {
	abstraction.Repository
}

func NewTest(db *gorm.DB) *test {
	return &test{
		abstraction.Repository{
			Db: db,
		},
	}
}
