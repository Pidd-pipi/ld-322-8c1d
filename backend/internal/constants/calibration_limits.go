package constants

// CalibrationLimits 定义各类传感器允许的校准偏移范围（含上下限），
// 超出范围的偏移会被拒绝，避免误填导致监测数据失真。
var CalibrationLimits = map[string]ThresholdRange{
	SensorTemperature: {Min: -10, Max: 10},
	SensorHumidity:    {Min: -20, Max: 20},
	SensorLight:       {Min: -5000, Max: 5000},
	SensorCO2:         {Min: -500, Max: 500},
	SensorSoil:        {Min: -20, Max: 20},
}
