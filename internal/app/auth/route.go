package auth

import (
	"daarul_mukhtarin/internal/middleware"

	"github.com/labstack/echo/v4"
)

func (h *handler) Route(v *echo.Group) {
	v.POST("/login", h.Login)
	v.POST("/logout", h.Logout, middleware.Logout)
	v.POST("/refresh-token", h.RefreshToken, middleware.RefreshToken)
	v.POST("/send-email/forgot-password", h.SendEmailForgotPassword, middleware.ResetPasswordIpCheck)
	v.GET("/validation/reset-password/:token", h.ValidationResetPassword)
}
