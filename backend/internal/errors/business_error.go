package apperrors

import (
	"fmt"
	"net/http"
)

type BusinessError struct {
	Code    int
	Message string
	Status  int
}

func (e *BusinessError) Error() string { return e.Message }
func New(code int, message string, status int) *BusinessError {
	return &BusinessError{Code: code, Message: message, Status: status}
}

var (
	ErrNotFound     = New(40401, "资源不存在", http.StatusNotFound)
	ErrValidation   = New(40001, "请求参数不合法", http.StatusBadRequest)
	ErrUnauthorized = New(40101, "认证失败", http.StatusUnauthorized)
	ErrInternal     = New(50001, "服务器内部错误", http.StatusInternalServerError)
	// ErrSensorNotFound 校准或写入时传感器不存在。
	ErrSensorNotFound = New(40402, "传感器不存在", http.StatusNotFound)
	// ErrForbidden 已登录但不具备管理员权限。
	ErrForbidden = New(40301, "没有权限执行该操作", http.StatusForbidden)
	// ErrCalibrationRange 偏移超出该传感器类型允许的范围，消息由调用方补充具体范围。
	ErrCalibrationRange = New(40002, "校准偏移超出允许范围", http.StatusBadRequest)
)

// NewCalibrationRangeError 生成带具体允许范围说明的偏移越界错误。
func NewCalibrationRangeError(sensorType string, limit float64) *BusinessError {
	return New(ErrCalibrationRange.Code, fmt.Sprintf("校准偏移超出允许范围：%s 传感器的偏移必须在 %.2f 到 %.2f 之间", sensorType, -limit, limit), http.StatusBadRequest)
}
