package handler

import (
	"github.com/cygreenenv/greenhouse-panel/internal/dto"
	apperrors "github.com/cygreenenv/greenhouse-panel/internal/errors"
	"github.com/cygreenenv/greenhouse-panel/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type SensorHandler struct {
	service   *service.MonitoringService
	validator *validator.Validate
}

func NewSensorHandler(s *service.MonitoringService, v *validator.Validate) *SensorHandler {
	return &SensorHandler{s, v}
}
func (h *SensorHandler) Create(c *gin.Context) {
	var req dto.SensorRequest
	if err := c.ShouldBindJSON(&req); err != nil || h.validator.Struct(req) != nil || req.MinValue >= req.MaxValue {
		Fail(c, apperrors.ErrValidation)
		return
	}
	sensor, err := h.service.CreateSensor(req.GreenhouseID, req.Name, req.Type, req.MinValue, req.MaxValue)
	if err != nil {
		Fail(c, err)
		return
	}
	Created(c, sensor)
}

// UpdateCalibration 保存传感器校准偏移；无权限、超范围或传感器不存在时返回具体原因。
func (h *SensorHandler) UpdateCalibration(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.CalibrationRequest
	if err := c.ShouldBindJSON(&req); err != nil || h.validator.Struct(req) != nil {
		Fail(c, apperrors.ErrValidation)
		return
	}
	sensor, err := h.service.UpdateCalibration(id, *req.Offset)
	if err != nil {
		Fail(c, err)
		return
	}
	Success(c, sensor)
}
