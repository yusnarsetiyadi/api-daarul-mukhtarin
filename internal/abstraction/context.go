package abstraction

import (
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type Context struct {
	echo.Context
	Auth *AuthContext
	Trx  *TrxContext
}

type AuthContext struct {
	ID       int
	RoleID   int
	DivisiID int
	Email    string
}

type TrxContext struct {
	Db *gorm.DB
}
