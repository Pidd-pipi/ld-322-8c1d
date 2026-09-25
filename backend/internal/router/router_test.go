package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cygreenenv/greenhouse-panel/internal/config"
	"github.com/cygreenenv/greenhouse-panel/internal/model"
	"github.com/cygreenenv/greenhouse-panel/internal/repository"
	"github.com/cygreenenv/greenhouse-panel/internal/service"
	ws "github.com/cygreenenv/greenhouse-panel/internal/websocket"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const testSecret = "router-test-secret"

func setupTestEngine(t *testing.T) (*httptest.Server, uint) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:calibration_router?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(model.All()...); err != nil {
		t.Fatal(err)
	}
	g := model.Greenhouse{Name: "路由测试温室"}
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
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	hub := ws.NewHub()
	auth := service.NewAuthService(testSecret)
	sensors := repository.NewSensorRepository(db)
	alerts := repository.NewAlertRepository(db)
	monitoring := service.NewMonitoringService(repository.NewGreenhouseRepository(db), sensors, alerts, logger, hub)
	engine := New(Dependencies{
		Config:     config.Config{},
		Logger:     logger,
		Auth:       auth,
		Monitoring: monitoring,
		Alerts:     service.NewAlertService(alerts, logger),
		Control:    service.NewControlService(repository.NewDeviceRepository(db), logger, hub),
		Reports:    service.NewReportService(sensors, alerts),
		Hub:        hub,
	})
	return httptest.NewServer(engine), sensor.ID
}

func adminToken(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{"sub": "admin", "role": "admin", "exp": time.Now().Add(time.Hour).Unix()}
	value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func viewerToken(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{"sub": "viewer", "role": "viewer", "exp": time.Now().Add(time.Hour).Unix()}
	value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func putCalibration(t *testing.T, baseURL string, sensorID uint, token, body string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/v1/sensors/%d/calibration", baseURL, sensorID), bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var parsed map[string]any
	if err = json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, parsed
}

func TestCalibrationEndpoint(t *testing.T) {
	server, sensorID := setupTestEngine(t)
	defer server.Close()

	t.Run("未登录返回401", func(t *testing.T) {
		status, body := putCalibration(t, server.URL, sensorID, "", `{"offset":1}`)
		if status != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d: %v", status, body)
		}
	})
	t.Run("非管理员返回403并说明原因", func(t *testing.T) {
		status, body := putCalibration(t, server.URL, sensorID, viewerToken(t), `{"offset":1}`)
		if status != http.StatusForbidden || body["message"] != "没有权限执行此操作，需要管理员角色" {
			t.Fatalf("want 403 with reason, got %d: %v", status, body)
		}
	})
	t.Run("超出允许范围返回400并说明区间", func(t *testing.T) {
		status, body := putCalibration(t, server.URL, sensorID, adminToken(t), `{"offset":99}`)
		if status != http.StatusBadRequest || body["message"] != "校准偏移超出允许范围：温度允许 -10.0 ~ 10.0 °C" {
			t.Fatalf("want 400 with range reason, got %d: %v", status, body)
		}
	})
	t.Run("传感器不存在返回404", func(t *testing.T) {
		status, body := putCalibration(t, server.URL, 9999, adminToken(t), `{"offset":1}`)
		if status != http.StatusNotFound || body["message"] != "传感器不存在" {
			t.Fatalf("want 404 with reason, got %d: %v", status, body)
		}
	})
	t.Run("缺少offset字段返回400", func(t *testing.T) {
		status, _ := putCalibration(t, server.URL, sensorID, adminToken(t), `{}`)
		if status != http.StatusBadRequest {
			t.Fatalf("want 400, got %d", status)
		}
	})
	t.Run("管理员保存成功且可清零恢复", func(t *testing.T) {
		status, body := putCalibration(t, server.URL, sensorID, adminToken(t), `{"offset":-1.5}`)
		if status != http.StatusOK {
			t.Fatalf("want 200, got %d: %v", status, body)
		}
		data := body["data"].(map[string]any)
		if data["calibrationOffset"] != -1.5 {
			t.Fatalf("want offset -1.5, got %v", data["calibrationOffset"])
		}
		status, body = putCalibration(t, server.URL, sensorID, adminToken(t), `{"offset":0}`)
		if status != http.StatusOK {
			t.Fatalf("want 200, got %d: %v", status, body)
		}
		if body["data"].(map[string]any)["calibrationOffset"] != 0.0 {
			t.Fatalf("want offset reset to 0, got %v", body["data"])
		}
	})
}
