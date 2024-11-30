package notifikasi

import (
	"daarul_mukhtarin/internal/middleware"

	"github.com/labstack/echo/v4"
)

func (h *handler) Route(v *echo.Group) {
	v.GET("", h.Find, middleware.Authentication)
	v.PUT("/set-read/:id", h.SetRead, middleware.Authentication)
}
