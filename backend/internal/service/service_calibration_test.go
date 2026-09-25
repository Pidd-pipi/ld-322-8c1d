package service

import (
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	apperrors "github.com/cygreenenv/greenhouse-panel/internal/errors"
	"github.com/cygreenenv/greenhouse-panel/internal/model"
	"github.com/cygreenenv/greenhouse-panel/internal/repository"
	ws "github.com/cygreenenv/greenhouse-panel/internal/websocket"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newCalibrationTestService(t *testing.T, dsn string) (*MonitoringService, *gorm.DB, model.Sensor) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(model.All()...); err != nil {
		t.Fatal(err)
	}
	g := model.Greenhouse{Name: "校准温室"}
	if err = db.Create(&g).Error; err != nil {
		t.Fatal(err)
	}
	sensor := model.Sensor{GreenhouseID: g.ID, Name: "温度传感器", Type: "temperature", Unit: "°C", Status: "online"}
	if err = db.Create(&sensor).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&model.Threshold{SensorID: sensor.ID, MinValue: 10, MaxValue: 30}).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewMonitoringService(repository.NewGreenhouseRepository(db), repository.NewSensorRepository(db), repository.NewAlertRepository(db), slog.New(slog.NewTextHandler(io.Discard, nil)), ws.NewHub())
	return svc, db, sensor
}

func TestUpdateCalibrationAppliesToLatestHistoryAndAlerts(t *testing.T) {
	svc, db, sensor := newCalibrationTestService(t, "file:calibration_svc?mode=memory&cache=shared")
	if _, err := svc.UpdateCalibration(sensor.ID, 5); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Ingest(sensor.ID, 20); err != nil {
		t.Fatal(err)
	}
	latest, err := svc.Latest(sensor.GreenhouseID)
	if err != nil {
		t.Fatal(err)
	}
	if len(latest) != 1 || latest[0].Value != 25 {
		t.Fatalf("latest should use calibrated value 25, got %#v", latest)
	}
	if latest[0].RawValue == nil || *latest[0].RawValue != 20 {
		t.Fatalf("latest should keep raw value 20, got %#v", latest[0].RawValue)
	}
	history, err := svc.History(sensor.GreenhouseID, nil, time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].Value != 25 || history[0].RawValue == nil || *history[0].RawValue != 20 {
		t.Fatalf("history should use calibrated value and keep raw, got %#v", history)
	}
	// 原始值 26 在阈值 [10,30] 内，校准后 31 越界，应触发报警且报警值为校准值。
	_, alert, err := svc.Ingest(sensor.ID, 26)
	if err != nil {
		t.Fatal(err)
	}
	if alert == nil || alert.Value != 31 {
		t.Fatalf("alert should be evaluated on calibrated value 31, got %#v", alert)
	}
	// 偏移调回 0 后恢复原始值。
	if _, err = svc.UpdateCalibration(sensor.ID, 0); err != nil {
		t.Fatal(err)
	}
	latest, err = svc.Latest(sensor.GreenhouseID)
	if err != nil {
		t.Fatal(err)
	}
	if len(latest) != 1 || latest[0].Value != 26 {
		t.Fatalf("offset reset to 0 should restore raw value 26, got %#v", latest)
	}
	var stored model.SensorReading
	if err = db.First(&stored, latest[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Value != 26 {
		t.Fatalf("stored reading must stay raw, got %v", stored.Value)
	}
}

func TestUpdateCalibrationRejectsOutOfRange(t *testing.T) {
	svc, db, sensor := newCalibrationTestService(t, "file:calibration_range?mode=memory&cache=shared")
	_, err := svc.UpdateCalibration(sensor.ID, 99)
	var business *apperrors.BusinessError
	if !errors.As(err, &business) || business.Code != 40002 {
		t.Fatalf("want calibration out-of-range error, got %v", err)
	}
	var stored model.Sensor
	if err = db.First(&stored, sensor.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.CalibrationOffset != 0 {
		t.Fatalf("failed save must not change stored offset, got %v", stored.CalibrationOffset)
	}
}

func TestUpdateCalibrationSensorNotFound(t *testing.T) {
	svc, _, _ := newCalibrationTestService(t, "file:calibration_missing?mode=memory&cache=shared")
	_, err := svc.UpdateCalibration(9999, 1)
	if !errors.Is(err, apperrors.ErrSensorNotFound) {
		t.Fatalf("want sensor-not-found error, got %v", err)
	}
}

func TestReportUsesCalibratedValues(t *testing.T) {
	svc, db, sensor := newCalibrationTestService(t, "file:calibration_report?mode=memory&cache=shared")
	if _, err := svc.UpdateCalibration(sensor.ID, 2); err != nil {
		t.Fatal(err)
	}
	for _, value := range []float64{20, 22} {
		if _, _, err := svc.Ingest(sensor.ID, value); err != nil {
			t.Fatal(err)
		}
	}
	reports := NewReportService(repository.NewSensorRepository(db), repository.NewAlertRepository(db))
	report, err := reports.Generate(sensor.GreenhouseID, "day", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	metric := report.Metrics["temperature"]
	if metric.Average != 23 || metric.Min != 22 || metric.Max != 24 {
		t.Fatalf("report should use calibrated values, got %#v", metric)
	}
}
