package auth

import (
	"context"
	"daarul_mukhtarin/internal/abstraction"
	"daarul_mukhtarin/internal/config"
	"daarul_mukhtarin/internal/dto"
	"daarul_mukhtarin/internal/factory"
	"daarul_mukhtarin/internal/model"
	modelToken "daarul_mukhtarin/internal/model/token"
	"daarul_mukhtarin/internal/repository"
	"daarul_mukhtarin/pkg/constant"
	"daarul_mukhtarin/pkg/gomail"
	"daarul_mukhtarin/pkg/util/aescrypt"
	"daarul_mukhtarin/pkg/util/encoding"
	"daarul_mukhtarin/pkg/util/general"
	"daarul_mukhtarin/pkg/util/response"
	"daarul_mukhtarin/pkg/util/trxmanager"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service interface {
	Login(ctx *abstraction.Context, payload *dto.AuthLoginRequest) (map[string]interface{}, error)
	Logout(ctx *abstraction.Context) (map[string]interface{}, error)
	RefreshToken(ctx *abstraction.Context) (map[string]interface{}, error)
	SendEmailForgotPassword(ctx *abstraction.Context, payload *dto.AuthSendEmailForgotPasswordRequest) (map[string]interface{}, error)
	ValidationResetPassword(ctx *abstraction.Context, payload *dto.AuthValidationResetPasswordRequest) (string, error)
}

type service struct {
	UserRepository repository.User

	DB      *gorm.DB
	DbRedis *redis.Client
}

func NewService(f *factory.Factory) Service {
	return &service{
		UserRepository: f.UserRepository,

		DB:      f.Db,
		DbRedis: f.DbRedis,
	}
}

func (s *service) encryptTokenClaims(v int) (encryptedString string, err error) {
	encryptedString, err = aescrypt.EncryptAES(fmt.Sprint(v), config.Get().JWT.SecretKey)
	return
}

func (s *service) Login(ctx *abstraction.Context, payload *dto.AuthLoginRequest) (map[string]interface{}, error) {
	var (
		err   error
		data  = new(model.UserEntityModel)
		token string
	)
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		data, err = s.UserRepository.FindByEmail(ctx, payload.Email)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if data == nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "email or password is incorrect")
		}

		if err = bcrypt.CompareHashAndPassword([]byte(data.Password), []byte(payload.Password)); err != nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "email or password is incorrect")
		}

		if data.IsLocked {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "this account is locked")
		}

		var encryptedUserID string
		if encryptedUserID, err = s.encryptTokenClaims(data.ID); err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		var encryptedUserRoleID string
		if encryptedUserRoleID, err = s.encryptTokenClaims(data.RoleId); err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		var encryptedUserDivisiID string
		if encryptedUserDivisiID, err = s.encryptTokenClaims(data.DivisiId); err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		encodedEmail := encoding.Encode(data.Email)

		tokenClaims := &modelToken.TokenClaims{
			ID:       encryptedUserID,
			RoleID:   encryptedUserRoleID,
			DivisiID: encryptedUserDivisiID,
			Email:    encodedEmail,
			Exp:      time.Now().Add(time.Duration(1 * time.Hour)).Unix(),
		}
		authToken := modelToken.NewAuthToken(tokenClaims)
		token, err = authToken.Token()
		if err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		if err := s.UserRepository.UpdateLoginFrom(ctx, &data.ID, payload.LoginFrom).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"token": token,
		"data": map[string]interface{}{
			"id":         data.ID,
			"name":       data.Name,
			"email":      data.Email,
			"created_at": data.CreatedAt,
			"updated_at": data.UpdatedAt,
			"role": map[string]interface{}{
				"id":   data.Role.ID,
				"name": data.Role.Name,
			},
			"divisi": map[string]interface{}{
				"id":   data.Divisi.ID,
				"name": data.Divisi.Name,
			},
		},
	}, nil
}

func (s *service) Logout(ctx *abstraction.Context) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		if err := s.UserRepository.UpdateLoginFrom(ctx, &ctx.Auth.ID, "").Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"message": "success logout!",
	}, nil
}

func (s *service) RefreshToken(ctx *abstraction.Context) (map[string]interface{}, error) {
	var token string
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		data, err := s.UserRepository.FindById(ctx, ctx.Auth.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		var encryptedUserID string
		if encryptedUserID, err = s.encryptTokenClaims(data.ID); err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		var encryptedUserRoleID string
		if encryptedUserRoleID, err = s.encryptTokenClaims(data.RoleId); err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		var encryptedUserDivisiID string
		if encryptedUserDivisiID, err = s.encryptTokenClaims(data.DivisiId); err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		encodedEmail := encoding.Encode(data.Email)

		tokenClaims := &modelToken.TokenClaims{
			ID:       encryptedUserID,
			RoleID:   encryptedUserRoleID,
			DivisiID: encryptedUserDivisiID,
			Email:    encodedEmail,
			Exp:      time.Now().Add(time.Duration(1 * time.Hour)).Unix(),
		}
		authToken := modelToken.NewAuthToken(tokenClaims)
		token, err = authToken.Token()
		if err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"token": token,
	}, nil
}

func (s *service) SendEmailForgotPassword(ctx *abstraction.Context, payload *dto.AuthSendEmailForgotPasswordRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		data, err := s.UserRepository.FindByEmail(ctx, payload.Email)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if data == nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "email not found")
		}

		eksternalToken := new(modelToken.AuthEksternalToken)
		eksternalToken.UserId = data.ID
		token, err := eksternalToken.GenerateTokenEksternal()
		if err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		s.DbRedis.Set(context.Background(), *token, *token, 0)

		if err = gomail.SendMail(data.Email, "Forgot Password for SelarasHomeId", general.ParseTemplateEmail("./assets/html/forgot_password.html", struct {
			NAME  string
			EMAIL string
			LINK  string
		}{
			NAME:  data.Name,
			EMAIL: data.Email,
			LINK:  constant.BASE_URL + "/auth/validation/reset-password/" + *token,
		})); err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"message": "success send email forgot password!",
	}, nil
}

func (s *service) ValidationResetPassword(ctx *abstraction.Context, payload *dto.AuthValidationResetPasswordRequest) (string, error) {
	userData := new(model.UserEntityModel)
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		_, err := s.DbRedis.Get(context.Background(), payload.Token).Result()
		if err == redis.Nil {
			return errors.New("your token is invalid")
		} else {
			s.DbRedis.Del(context.Background(), payload.Token)
		}

		data, err := modelToken.ValidateTokenEksternal(payload.Token)
		if err != nil {
			return errors.New("your token is invalid")
		}

		userData, err = s.UserRepository.FindById(ctx, data.UserId)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if userData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "user not found")
		}

		passwordString := general.GeneratePassword(8, 1, 1, 1, 1)
		password := []byte(passwordString)
		hashedPassword, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
		if err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		newUserData := new(model.UserEntityModel)
		newUserData.Context = ctx
		newUserData.ID = userData.ID
		newUserData.Password = string(hashedPassword)

		if err = s.UserRepository.Update(ctx, newUserData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		if err = gomail.SendMail(userData.Email, "Reset Password for SelarasHomeId", general.ParseTemplateEmail("./assets/html/reset_password_admin.html", struct {
			NAME      string
			RESETNAME string
			EMAIL     string
			PASSWORD  string
			LINK      string
		}{
			NAME:      userData.Name,
			RESETNAME: "System",
			EMAIL:     userData.Email,
			PASSWORD:  passwordString,
			LINK:      constant.BASE_URL,
		})); err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		return nil
	}); err != nil {
		return "", err
	}

	return userData.Email, nil
}
