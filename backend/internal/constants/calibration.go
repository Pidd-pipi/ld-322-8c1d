package constants

// MaxCalibrationOffset 限定各类传感器允许的校准偏移绝对值，
// 防止值班员误填造成数据失真；偏移回到 0 即恢复原始读数。
var MaxCalibrationOffset = map[string]float64{
	SensorTemperature: 5,
	SensorHumidity:    10,
	SensorLight:       5000,
	SensorCO2:         200,
	SensorSoil:        10,
}
