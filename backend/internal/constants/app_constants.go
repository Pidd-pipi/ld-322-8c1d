package constants

const (
	APIPrefix       = "/api/v1"
	HealthPath      = "/healthz"
	WebSocketPath   = "/ws"
	DefaultPage     = 1
	DefaultPageSize = 100
	MaxPageSize     = 500
	StatusOnline    = "online"
	StatusOff       = "off"
	StatusOn        = "on"
	AlertPending    = "pending"
	AlertHandled    = "handled"
	RoleAdmin       = "admin"
	SuccessMessage  = "ok"
	EventAlert      = "alert.created"
	EventDevice     = "device.updated"
	// EventSensorCalibrated 校准偏移保存成功后广播，通知各终端刷新校准值。
	EventSensorCalibrated = "sensor.calibrated"
	// ContextClaims 是 JWT 声明在 gin.Context 中的键。
	ContextClaims = "jwt_claims"
)
