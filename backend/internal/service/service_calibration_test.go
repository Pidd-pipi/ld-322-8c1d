package service

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/cygreenenv/greenhouse-panel/internal/constants"
	apperrors "github.com/cygreenenv/greenhouse-panel/internal/errors"
	"github.com/cygreenenv/greenhouse-panel/internal/model"
	"github.com/cygreenenv/greenhouse-panel/internal/repository"
	ws "github.com/cygreenenv/greenhouse-panel/internal/websocket"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newCalibrationFixture(t *testing.T) (*gorm.DB, *CalibrationService, model.Sensor) {
	t.Helper()
	dsn := fmt.Sprintf("file:cal_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(model.All()...); err != nil {
		t.Fatal(err)
	}
	g := model.Greenhouse{Name: "校准测试温室"}
	if err = db.Create(&g).Error; err != nil {
		t.Fatal(err)
	}
	sensor := model.Sensor{GreenhouseID: g.ID, Name: "温度传感器", Type: constants.SensorTemperature, Unit: "°C", Status: constants.StatusOnline}
	if err = db.Create(&sensor).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&model.Threshold{SensorID: sensor.ID, MinValue: 10, MaxValue: 35}).Error; err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewCalibrationService(repository.NewSensorRepository(db), logger, ws.NewHub())
	return db, svc, sensor
}

func TestUpdateCalibrationOffset(t *testing.T) {
	_, svc, sensor := newCalibrationFixture(t)

	cases := []struct {
		name    string
		id      uint
		offset  float64
		role    string
		wantErr error
	}{
		{name: "管理员保存合法偏移", id: sensor.ID, offset: 1.5, role: constants.RoleAdmin, wantErr: nil},
		{name: "归零恢复原值", id: sensor.ID, offset: 0, role: constants.RoleAdmin, wantErr: nil},
		{name: "负向偏移合法", id: sensor.ID, offset: -5, role: constants.RoleAdmin, wantErr: nil},
		{name: "偏移超过上限", id: sensor.ID, offset: 5.01, role: constants.RoleAdmin, wantErr: apperrors.ErrCalibrationRange},
		{name: "偏移低于下限", id: sensor.ID, offset: -6, role: constants.RoleAdmin, wantErr: apperrors.ErrCalibrationRange},
		{name: "非管理员无权限", id: sensor.ID, offset: 1, role: "viewer", wantErr: apperrors.ErrForbidden},
		{name: "传感器不存在", id: 9999, offset: 1, role: constants.RoleAdmin, wantErr: apperrors.ErrRecordNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			updated, err := svc.UpdateOffset(tc.id, tc.offset, tc.role)
			if errorCode(err) != errorCode(tc.wantErr) {
				t.Fatalf("want error %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr == nil && updated.CalibrationOffset != tc.offset {
				t.Fatalf("want offset %.2f, got %.2f", tc.offset, updated.CalibrationOffset)
			}
		})
	}
}

func errorCode(err error) int {
	if err == nil {
		return 0
	}
	var business *apperrors.BusinessError
	if errors.As(err, &business) {
		return business.Code
	}
	return -1
}

func TestFailedSaveKeepsPreviousOffset(t *testing.T) {
	db, svc, sensor := newCalibrationFixture(t)

	if _, err := svc.UpdateOffset(sensor.ID, 2, constants.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	// 越界保存必须被拒绝，原偏移保持不变。
	if _, err := svc.UpdateOffset(sensor.ID, 99, constants.RoleAdmin); err == nil {
		t.Fatal("expected range error")
	}
	var current model.Sensor
	if err := db.First(&current, sensor.ID).Error; err != nil {
		t.Fatal(err)
	}
	if current.CalibrationOffset != 2 {
		t.Fatalf("offset must remain 2 after failed save, got %.2f", current.CalibrationOffset)
	}
}

func TestAlertUsesCalibratedValue(t *testing.T) {
	db, calibrationSvc, sensor := newCalibrationFixture(t)

	if _, err := calibrationSvc.UpdateOffset(sensor.ID, -5, constants.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if value := CalibratedValue(25, -5); value != 20 {
		t.Fatalf("want calibrated value 20, got %.1f", value)
	}

	// 阈值为 [10,35]，原始 14.9°C 在范围内本不该报警；
	// 叠加 -5 偏移后校准值为 9.9°C，低于下限，必须按校准值触发报警。
	monitoring := NewMonitoringService(
		repository.NewGreenhouseRepository(db),
		repository.NewSensorRepository(db),
		repository.NewAlertRepository(db),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		ws.NewHub(),
	)
	_, alert, err := monitoring.Ingest(sensor.ID, 14.9)
	if err != nil {
		t.Fatal(err)
	}
	if alert == nil {
		t.Fatal("expected alert based on calibrated value, got nil")
	}
	if alert.Value != 9.9 {
		t.Fatalf("alert must persist calibrated value 9.9, got %.2f", alert.Value)
	}

	// 原始读数仍然保留为 14.9。
	var stored model.SensorReading
	if err = db.Last(&stored, "sensor_id = ?", sensor.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Value != 14.9 {
		t.Fatalf("raw reading must remain 14.9, got %.2f", stored.Value)
	}
}
