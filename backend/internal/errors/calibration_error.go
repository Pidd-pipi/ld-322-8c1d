package apperrors

import (
	"fmt"
	"net/http"
)

// ErrCalibrationUnsupported 表示传感器类型没有配置校准偏移范围。
var ErrCalibrationUnsupported = New(40003, "该传感器类型不支持校准偏移", http.StatusBadRequest)

// NewCalibrationOutOfRange 构造校准偏移超出允许范围的错误，消息中说明允许区间。
func NewCalibrationOutOfRange(label, unit string, min, max float64) *BusinessError {
	return New(40002, fmt.Sprintf("校准偏移超出允许范围：%s允许 %.1f ~ %.1f %s", label, min, max, unit), http.StatusBadRequest)
}
