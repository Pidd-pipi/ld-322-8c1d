package service

import (
	"math"

	"github.com/cygreenenv/greenhouse-panel/internal/model"
)

// CalibratedValue 返回应用校准偏移后的数值（保留 4 位小数以消除浮点误差），
// 数据库中的原始读数保持不变；偏移为 0 时结果与原始值一致。
func CalibratedValue(sensor model.Sensor, raw float64) float64 {
	return math.Round((raw+sensor.CalibrationOffset)*1e4) / 1e4
}

// CalibrateReadings 将读数切片原地转换为校准值，原始值保留在 RawValue 字段中返回。
func CalibrateReadings(rows []model.SensorReading) {
	for i := range rows {
		raw := rows[i].Value
		rows[i].Value = CalibratedValue(rows[i].Sensor, raw)
		rows[i].RawValue = &raw
	}
}
