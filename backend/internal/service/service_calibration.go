package service

import (
	"github.com/cygreenenv/greenhouse-panel/internal/constants"
	apperrors "github.com/cygreenenv/greenhouse-panel/internal/errors"
	"github.com/cygreenenv/greenhouse-panel/internal/model"
	"github.com/cygreenenv/greenhouse-panel/internal/repository"
	ws "github.com/cygreenenv/greenhouse-panel/internal/websocket"
	"log/slog"
	"math"
)

type CalibrationService struct {
	sensorRepo *repository.SensorRepository
	logger     *slog.Logger
	hub        *ws.Hub
}

func NewCalibrationService(s *repository.SensorRepository, l *slog.Logger, h *ws.Hub) *CalibrationService {
	return &CalibrationService{sensorRepo: s, logger: l, hub: h}
}

// CalibratedValue 返回原始读数叠加偏移后的校准值。偏移为 0 时即原值。
func CalibratedValue(raw, offset float64) float64 {
	return raw + offset
}

// ApplyCalibration 就地为批量读数（需已预载 Sensor）填充校准值，
// 供最新读数、历史趋势等接口统一调用。
func ApplyCalibration(rows []model.SensorReading) {
	for i := range rows {
		rows[i].CalibratedValue = CalibratedValue(rows[i].Value, rows[i].Sensor.CalibrationOffset)
	}
}

// UpdateOffset 保存传感器校准偏移，仅管理员可用；
// 偏移必须落在该传感器类型允许的范围内，保存失败不会破坏原有偏移。
func (s *CalibrationService) UpdateOffset(sensorID uint, offset float64, role string) (*model.Sensor, error) {
	if role != constants.RoleAdmin {
		return nil, apperrors.ErrForbidden
	}
	if math.IsNaN(offset) || math.IsInf(offset, 0) {
		return nil, apperrors.ErrValidation
	}
	sensor, err := s.sensorRepo.Get(sensorID)
	if err != nil {
		return nil, err
	}
	limit, ok := constants.MaxCalibrationOffset[sensor.Type]
	if !ok {
		s.logger.Warn("calibration limit missing for sensor type", "sensorId", sensorID, "type", sensor.Type)
		return nil, apperrors.ErrValidation
	}
	if offset < -limit || offset > limit {
		return nil, apperrors.NewCalibrationRangeError(constants.SensorLabels[sensor.Type], limit)
	}
	updated, err := s.sensorRepo.UpdateCalibration(sensorID, offset)
	if err != nil {
		return nil, err
	}
	s.logger.Info("calibration offset saved",
		"sensorId", sensorID, "type", sensor.Type,
		"oldOffset", sensor.CalibrationOffset, "newOffset", offset)
	s.hub.Broadcast(constants.EventSensorCalibrated, updated)
	return updated, nil
}
