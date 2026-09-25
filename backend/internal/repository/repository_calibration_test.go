package repository

import (
	"errors"
	"testing"

	apperrors "github.com/cygreenenv/greenhouse-panel/internal/errors"
	"github.com/cygreenenv/greenhouse-panel/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSensorRepositoryUpdateCalibrationOffset(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:calibration_repo?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(model.All()...); err != nil {
		t.Fatal(err)
	}
	g := model.Greenhouse{Name: "校准仓库温室"}
	if err = db.Create(&g).Error; err != nil {
		t.Fatal(err)
	}
	s := model.Sensor{GreenhouseID: g.ID, Name: "温度", Type: "temperature", Unit: "°C", Status: "online"}
	if err = db.Create(&s).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewSensorRepository(db)
	if err = repo.UpdateCalibrationOffset(s.ID, -1.5); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.Get(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.CalibrationOffset != -1.5 {
		t.Fatalf("want offset -1.5, got %v", stored.CalibrationOffset)
	}
	if err = repo.UpdateCalibrationOffset(9999, 1); !errors.Is(err, apperrors.ErrRecordNotFound) {
		t.Fatalf("want record-not-found for missing sensor, got %v", err)
	}
}
