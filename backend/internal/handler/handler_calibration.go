package handler

import (
	"github.com/cygreenenv/greenhouse-panel/internal/constants"
	"github.com/cygreenenv/greenhouse-panel/internal/dto"
	apperrors "github.com/cygreenenv/greenhouse-panel/internal/errors"
	"github.com/cygreenenv/greenhouse-panel/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
)

type CalibrationHandler struct {
	service   *service.CalibrationService
	validator *validator.Validate
}

func NewCalibrationHandler(s *service.CalibrationService, v *validator.Validate) *CalibrationHandler {
	return &CalibrationHandler{service: s, validator: v}
}

// Update 保存某传感器的校准偏移，仅管理员可操作。
func (h *CalibrationHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.CalibrationRequest
	if err := c.ShouldBindJSON(&req); err != nil || h.validator.Struct(req) != nil {
		Fail(c, apperrors.ErrValidation)
		return
	}
	claims, exists := c.Get(constants.ContextClaims)
	if !exists {
		Fail(c, apperrors.ErrUnauthorized)
		return
	}
	role, _ := claims.(jwt.MapClaims)["role"].(string)
	sensor, err := h.service.UpdateOffset(id, *req.Offset, role)
	if err != nil {
		Fail(c, err)
		return
	}
	Success(c, sensor)
}
