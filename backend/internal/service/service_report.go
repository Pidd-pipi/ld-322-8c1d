package service

import (
	"fmt"
	"github.com/cygreenenv/greenhouse-panel/internal/model"
	"github.com/cygreenenv/greenhouse-panel/internal/repository"
	"time"
)

type ReportService struct {
	sensorRepo *repository.SensorRepository
	alertRepo  *repository.AlertRepository
}
type Report struct {
	GreenhouseID uint              `json:"greenhouseId"`
	Range        string            `json:"range"`
	GeneratedAt  time.Time         `json:"generatedAt"`
	Alerts       int64             `json:"alerts"`
	Metrics      map[string]Metric `json:"metrics"`
}
type Metric struct {
	Average float64 `json:"average"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Unit    string  `json:"unit"`
}

func NewReportService(s *repository.SensorRepository, a *repository.AlertRepository) *ReportService {
	return &ReportService{s, a}
}
func (s *ReportService) Generate(id uint, rangeName string, start, end time.Time) (*Report, error) {
	rows, err := s.sensorRepo.History(id, nil, start, end)
	if err != nil {
		return nil, fmt.Errorf("report history: %w", err)
	}
	report := &Report{GreenhouseID: id, Range: rangeName, GeneratedAt: time.Now(), Metrics: map[string]Metric{}}
	sums := map[string]float64{}
	counts := map[string]int{}
	for _, r := range rows {
		value := CalibratedValue(r.Value, r.Sensor.CalibrationOffset)
		m, ok := report.Metrics[r.Sensor.Type]
		if !ok {
			m = Metric{Min: value, Max: value, Unit: r.Sensor.Unit}
		}
		if value < m.Min {
			m.Min = value
		}
		if value > m.Max {
			m.Max = value
		}
		sums[r.Sensor.Type] += value
		counts[r.Sensor.Type]++
		m.Average = sums[r.Sensor.Type] / float64(counts[r.Sensor.Type])
		report.Metrics[r.Sensor.Type] = m
	}
	report.Alerts, err = s.alertRepo.CountBetween(id, start, end)
	if err != nil {
		return nil, err
	}
	return report, nil
}

var _ = model.Alert{}
