package router

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cygreenenv/greenhouse-panel/internal/constants"
	"github.com/cygreenenv/greenhouse-panel/internal/model"
	"github.com/cygreenenv/greenhouse-panel/internal/repository"
	"github.com/cygreenenv/greenhouse-panel/internal/service"
	ws "github.com/cygreenenv/greenhouse-panel/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"time"
)

func setupCalibrationRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:router_cal?mode=memory&cache=shared"), &gorm.Config{})
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
	sensor := model.Sensor{GreenhouseID: g.ID, Name: "温度", Type: constants.SensorTemperature, Unit: "°C", Status: constants.StatusOnline}
	if err = db.Create(&sensor).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&model.Threshold{SensorID: sensor.ID, MinValue: 15, MaxValue: 32}).Error; err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gh := repository.NewGreenhouseRepository(db)
	sr := repository.NewSensorRepository(db)
	ar := repository.NewAlertRepository(db)
	hub := ws.NewHub()
	auth := service.NewAuthService("test-secret")
	mon := service.NewMonitoringService(gh, sr, ar, logger, hub)
	cal := service.NewCalibrationService(sr, logger, hub)
	alerts := service.NewAlertService(ar, logger)
	reports := service.NewReportService(sr, ar)
	r := New(Dependencies{Logger: logger, Auth: auth, Monitoring: mon, Calibration: cal, Alerts: alerts, Reports: reports, Hub: hub})
	return r, db
}

func doPut(t *testing.T, r http.Handler, path, token string, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var parsed map[string]any
	raw, _ := io.ReadAll(w.Body)
	_ = json.Unmarshal(raw, &parsed)
	return w.Code, parsed
}

func TestCalibrationHTTPFlow(t *testing.T) {
	r, db := setupCalibrationRouter(t)

	// 1) 无令牌 → 401 且说明原因
	code, body := doPut(t, r, "/api/v1/sensors/1/calibration", "", `{"offset":1}`)
	if code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d body=%v", code, body)
	}
	if msg, _ := body["message"].(string); msg == "" {
		t.Fatal("401 response must explain the reason")
	}

	// 2) 非管理员令牌（viewer 角色，用相同密钥签发）→ 403
	viewerClaims := jwt.MapClaims{"sub": "viewer", "role": "viewer", "exp": time.Now().Add(time.Hour).Unix()}
	viewerToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, viewerClaims).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	code, body = doPut(t, r, "/api/v1/sensors/1/calibration", viewerToken, `{"offset":1}`)
	if code != http.StatusForbidden {
		t.Fatalf("want 403, got %d body=%v", code, body)
	}

	// 3) 管理员登录令牌保存合法偏移 → 200
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin","password":"admin123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginReq)
	var loginBody struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginW.Body.Bytes(), &loginBody); err != nil || loginBody.Data.Token == "" {
		t.Fatalf("login failed: %s", loginW.Body.String())
	}
	token := loginBody.Data.Token

	// 3) 合法偏移 → 200
	code, body = doPut(t, r, "/api/v1/sensors/1/calibration", token, `{"offset":2.5}`)
	if code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%v", code, body)
	}

	// 4) 超范围 → 400 且消息含允许范围
	code, body = doPut(t, r, "/api/v1/sensors/1/calibration", token, `{"offset":99}`)
	if code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%v", code, body)
	}
	msg, _ := body["message"].(string)
	if !bytes.Contains([]byte(msg), []byte("允许范围")) {
		t.Fatalf("range message must explain bounds, got %q", msg)
	}
	// 越界保存失败后原偏移仍是 2.5
	var sensor model.Sensor
	if err := db.First(&sensor, 1).Error; err != nil {
		t.Fatal(err)
	}
	if sensor.CalibrationOffset != 2.5 {
		t.Fatalf("offset must stay 2.5 after rejected save, got %v", sensor.CalibrationOffset)
	}

	// 5) 传感器不存在 → 404
	code, body = doPut(t, r, "/api/v1/sensors/999/calibration", token, `{"offset":1}`)
	if code != http.StatusNotFound {
		t.Fatalf("want 404, got %d body=%v", code, body)
	}

	// 6) 归零 → 恢复原值
	code, _ = doPut(t, r, "/api/v1/sensors/1/calibration", token, `{"offset":0}`)
	if code != http.StatusOK {
		t.Fatalf("want 200 reset, got %d", code)
	}
	if err := db.First(&sensor, 1).Error; err != nil {
		t.Fatal(err)
	}
	if sensor.CalibrationOffset != 0 {
		t.Fatalf("offset reset to 0, got %v", sensor.CalibrationOffset)
	}

	// 7) 缺字段 → 400 校验失败
	code, _ = doPut(t, r, "/api/v1/sensors/1/calibration", token, `{}`)
	if code != http.StatusBadRequest {
		t.Fatalf("want 400 validation, got %d", code)
	}

	// 8) 设置偏移 2.5，写入原始读数 20，最新读数/历史/报告均应体现校准值 22.5
	if _, body = doPut(t, r, "/api/v1/sensors/1/calibration", token, `{"offset":2.5}`); body == nil {
		t.Fatal("set offset failed")
	}
	ingestReq := httptest.NewRequest(http.MethodPost, "/api/v1/readings", bytes.NewBufferString(`{"sensorId":1,"value":20}`))
	ingestReq.Header.Set("Content-Type", "application/json")
	ingestReq.Header.Set("Authorization", "Bearer "+token)
	ingestW := httptest.NewRecorder()
	r.ServeHTTP(ingestW, ingestReq)
	if ingestW.Code != http.StatusCreated {
		t.Fatalf("ingest want 201, got %d %s", ingestW.Code, ingestW.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/readings/latest?greenhouse_id=1", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	var latest struct {
		Data []struct {
			Value           float64 `json:"value"`
			CalibratedValue float64 `json:"calibratedValue"`
		} `json:"data"`
	}
	if err := json.Unmarshal(getW.Body.Bytes(), &latest); err != nil || len(latest.Data) == 0 {
		t.Fatalf("latest readings failed: %s", getW.Body.String())
	}
	if latest.Data[0].Value != 20 || latest.Data[0].CalibratedValue != 22.5 {
		t.Fatalf("want raw 20 / calibrated 22.5, got %+v", latest.Data[0])
	}

	reportReq := httptest.NewRequest(http.MethodGet, "/api/v1/reports/environment?greenhouse_id=1&range=day", nil)
	reportW := httptest.NewRecorder()
	r.ServeHTTP(reportW, reportReq)
	reportBody := reportW.Body.String()
	if reportW.Code != http.StatusOK {
		t.Fatalf("report want 200, got %d %s", reportW.Code, reportBody)
	}
	if !bytes.Contains([]byte(reportBody), []byte(`"average":22.5`)) {
		t.Fatalf("report must use calibrated value 22.5, got %s", reportBody)
	}
}
