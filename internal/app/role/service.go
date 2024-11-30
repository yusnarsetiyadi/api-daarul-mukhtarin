package role

import (
	"daarul_mukhtarin/internal/abstraction"
	"daarul_mukhtarin/internal/dto"
	"daarul_mukhtarin/internal/factory"
	"daarul_mukhtarin/internal/model"
	"daarul_mukhtarin/internal/repository"
	"daarul_mukhtarin/pkg/constant"
	"daarul_mukhtarin/pkg/util/response"
	"daarul_mukhtarin/pkg/util/trxmanager"
	"errors"
	"net/http"

	"gorm.io/gorm"
)

type Service interface {
	Find(ctx *abstraction.Context) (map[string]interface{}, error)
	Update(ctx *abstraction.Context, payload *dto.RoleUpdateRequest) (map[string]interface{}, error)
}

type service struct {
	RoleRepository repository.Role

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		RoleRepository: f.RoleRepository,

		DB: f.Db,
	}
}

func (s *service) Find(ctx *abstraction.Context) (map[string]interface{}, error) {
	if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
		return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
	}
	data, err := s.RoleRepository.Find(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	count, err := s.RoleRepository.Count(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	var res []map[string]interface{}
	for _, v := range data {
		res = append(res, map[string]interface{}{
			"id":        v.ID,
			"name":      v.Name,
			"is_delete": v.IsDelete,
		})
	}
	return map[string]interface{}{
		"count": count,
		"data":  res,
	}, nil
}

func (s *service) Update(ctx *abstraction.Context, payload *dto.RoleUpdateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
		}

		roleData, err := s.RoleRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if roleData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "role not found")
		}

		newRoleData := new(model.RoleEntityModel)
		newRoleData.Context = ctx
		newRoleData.ID = payload.ID
		if payload.Name != nil {
			newRoleData.Name = *payload.Name
		}

		if err = s.RoleRepository.Update(ctx, newRoleData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "success update!",
	}, nil
}
