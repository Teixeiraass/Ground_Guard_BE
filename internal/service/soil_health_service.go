package service

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
)

const (
	DefaultIdealMoistureMin int32 = 35
	DefaultIdealMoistureMax int32 = 80

	// A sensor that hasn't reported for this long is treated as offline:
	// its last reading no longer describes the soil.
	SensorOfflineAfter = 15 * time.Minute

	soilHealthWindow       = 24 * time.Hour
	minReadingsForHistory  = 3
	outOfRangeScoreCeiling = 70
	maxDevicesPerUser      = 500
)

// SensorHealthInput is everything needed to grade one sensor.
type SensorHealthInput struct {
	Device     db.Device
	Preference *db.IrrigationPreference
	History    []db.DeviceSensorHistory
}

type SoilHealthService interface {
	GetSoilHealth(ctx context.Context, userID int64) (dto.SoilHealthResponse, error)
}

type soilHealthService struct {
	store db.Store
}

func NewSoilHealthService(store db.Store) SoilHealthService {
	return &soilHealthService{store: store}
}

func (s *soilHealthService) GetSoilHealth(ctx context.Context, userID int64) (dto.SoilHealthResponse, error) {
	now := time.Now()

	devices, err := s.store.ListDevicesByUserID(ctx, db.ListDevicesByUserIDParams{
		UserID: sql.NullInt64{Int64: userID, Valid: true},
		Limit:  maxDevicesPerUser,
		Offset: 0,
	})
	if err != nil {
		return dto.SoilHealthResponse{}, err
	}

	inputs := make([]SensorHealthInput, 0, len(devices))
	for _, device := range devices {
		input := SensorHealthInput{Device: device}

		pref, err := s.store.GetIrrigationPreferenceByDevice(ctx, device.ID)
		if err == nil {
			input.Preference = &pref
		} else if !errors.Is(err, sql.ErrNoRows) {
			return dto.SoilHealthResponse{}, err
		}

		history, err := s.store.ListDeviceSensorHistory(ctx, db.ListDeviceSensorHistoryParams{
			DeviceID:  device.ID,
			CreatedAt: now.Add(-soilHealthWindow),
		})
		if err != nil {
			return dto.SoilHealthResponse{}, err
		}
		input.History = history

		inputs = append(inputs, input)
	}

	return ComputeSoilHealth(inputs, now), nil
}

// IdealMoistureRange uses the auto-irrigation threshold as the lower bound
// when the user configured one, since that's the level they consider "dry".
func IdealMoistureRange(pref *db.IrrigationPreference) (int32, int32) {
	min, max := DefaultIdealMoistureMin, DefaultIdealMoistureMax
	if pref != nil && pref.MoistureThreshold > 0 && pref.MoistureThreshold < max-5 {
		min = pref.MoistureThreshold
	}
	return min, max
}

func ComputeSoilHealth(inputs []SensorHealthInput, now time.Time) dto.SoilHealthResponse {
	resp := dto.SoilHealthResponse{
		Sensors:     make([]dto.SensorHealthResponse, 0, len(inputs)),
		GeneratedAt: now,
	}
	resp.Summary.Total = len(inputs)

	var scoreSum, scored, moistureSum, moistureCount int

	for _, in := range inputs {
		sensor := gradeSensor(in, now)
		resp.Sensors = append(resp.Sensors, sensor)

		switch sensor.State {
		case dto.SensorStateOK:
			resp.Summary.OK++
		case dto.SensorStateDry:
			resp.Summary.Dry++
		case dto.SensorStateWet:
			resp.Summary.Wet++
		case dto.SensorStateOffline:
			resp.Summary.Offline++
		case dto.SensorStateFault:
			resp.Summary.Fault++
		case dto.SensorStateNoData:
			resp.Summary.NoData++
		}

		if sensor.Score != nil {
			scoreSum += *sensor.Score
			scored++
		}
		if isLiveState(sensor.State) && sensor.SoilMoisture != nil {
			moistureSum += int(*sensor.SoilMoisture)
			moistureCount++
		}
	}

	if moistureCount > 0 {
		avg := roundDiv(moistureSum, moistureCount)
		resp.AvgMoisture = &avg
	}

	switch {
	case len(inputs) == 0:
		resp.Status = dto.SoilHealthNoDevices
	case scored == 0:
		resp.Status = dto.SoilHealthNoData
	default:
		score := roundDiv(scoreSum, scored)
		resp.Score = &score
		resp.Status = statusForScore(score)
	}

	return resp
}

func gradeSensor(in SensorHealthInput, now time.Time) dto.SensorHealthResponse {
	device := in.Device
	min, max := IdealMoistureRange(in.Preference)

	sensor := dto.SensorHealthResponse{
		Uuid:        device.Uuid,
		Name:        device.Name,
		IdealMin:    min,
		IdealMax:    max,
		Readings24h: len(in.History),
	}
	if device.LastSeen.Valid {
		lastSeen := device.LastSeen.Time
		sensor.LastSeen = &lastSeen
	}
	if device.SoilMoisture.Valid {
		m := device.SoilMoisture.Int32
		sensor.SoilMoisture = &m
	}

	if len(in.History) > 0 {
		inRange := 0
		for _, h := range in.History {
			if h.SoilMoisture >= min && h.SoilMoisture <= max {
				inRange++
			}
		}
		pct := roundDiv(inRange*100, len(in.History))
		sensor.TimeInRange24h = &pct
	}

	switch {
	case device.SoilSensorOk.Valid && !device.SoilSensorOk.Bool:
		sensor.State = dto.SensorStateFault
		return sensor
	case !device.LastSeen.Valid || now.Sub(device.LastSeen.Time) > SensorOfflineAfter:
		sensor.State = dto.SensorStateOffline
		return sensor
	case !device.SoilMoisture.Valid:
		sensor.State = dto.SensorStateNoData
		return sensor
	}

	current := device.SoilMoisture.Int32
	var distance int32
	switch {
	case current < min:
		sensor.State = dto.SensorStateDry
		distance = min - current
	case current > max:
		sensor.State = dto.SensorStateWet
		distance = current - max
	default:
		sensor.State = dto.SensorStateOK
	}

	var score int
	if len(in.History) >= minReadingsForHistory {
		score = *sensor.TimeInRange24h
	} else if sensor.State == dto.SensorStateOK {
		score = 100
	} else {
		score = outOfRangeScoreCeiling - int(3*distance)
	}
	if sensor.State != dto.SensorStateOK && score > outOfRangeScoreCeiling {
		score = outOfRangeScoreCeiling
	}
	if score < 0 {
		score = 0
	}
	sensor.Score = &score

	return sensor
}

func isLiveState(state string) bool {
	return state == dto.SensorStateOK || state == dto.SensorStateDry || state == dto.SensorStateWet
}

func statusForScore(score int) string {
	switch {
	case score >= 85:
		return dto.SoilHealthExcellent
	case score >= 65:
		return dto.SoilHealthGood
	case score >= 40:
		return dto.SoilHealthAttention
	default:
		return dto.SoilHealthCritical
	}
}

func roundDiv(a, b int) int {
	return int(math.Round(float64(a) / float64(b)))
}
