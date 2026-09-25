package model

import "time"

type Sensor struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	GreenhouseID      uint      `gorm:"index" json:"greenhouseId"`
	Name              string    `gorm:"type:varchar(100)" json:"name"`
	Type              string    `gorm:"type:varchar(50);index" json:"type"`
	Unit              string    `gorm:"type:varchar(32)" json:"unit"`
	Status            string    `gorm:"type:varchar(32)" json:"status"`
	CalibrationOffset float64   `gorm:"not null;default:0" json:"calibrationOffset"`
	CreatedAt         time.Time `json:"createdAt"`
	Threshold         Threshold `json:"threshold,omitempty"`
}
type SensorReading struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SensorID   uint      `gorm:"index" json:"sensorId"`
	Value      float64   `json:"value"`
	RecordedAt time.Time `gorm:"index" json:"recordedAt"`
	// RawValue 是校准前的原始读数，仅在应用了校准偏移的响应中填充，不落库。
	RawValue *float64 `gorm:"-" json:"rawValue,omitempty"`
	Sensor   Sensor   `json:"sensor,omitempty"`
}
type Threshold struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SensorID  uint      `gorm:"uniqueIndex" json:"sensorId"`
	MinValue  float64   `json:"minValue"`
	MaxValue  float64   `json:"maxValue"`
	UpdatedAt time.Time `json:"updatedAt"`
}
