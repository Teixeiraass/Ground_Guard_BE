package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/mqtt/client"
	"github.com/Teixeiraass/ground_guard_be/mqtt/publisher"
	"github.com/google/uuid"
)

type IrrigationService interface {
	CreateIrrigationCommand(ctx context.Context, req dto.CreateIrrigationCommandRequest, userID int64) (*db.IrrigationCommand, error)
	GetIrrigationCommand(ctx context.Context, commandUUID uuid.UUID) (*db.IrrigationCommand, error)
	ListIrrigationHistory(ctx context.Context, userID int64, date time.Time, deviceUUID *uuid.UUID) ([]db.IrrigationAction, error)
	GetIrrigationHistory(ctx context.Context, actionUUID uuid.UUID) (*db.IrrigationAction, error)
	GetWaterConsumption(ctx context.Context, userID int64, period string, deviceUUID *uuid.UUID) (dto.WaterConsumptionResponse, error)
	CreateIrrigationPreference(ctx context.Context, req dto.CreateIrrigationPreferenceRequest) (*db.IrrigationPreference, error)
	GetIrrigationPreference(ctx context.Context, preferenceUUID uuid.UUID) (*db.IrrigationPreference, error)
	GetIrrigationPreferenceByDevice(ctx context.Context, deviceUUID uuid.UUID) (*db.IrrigationPreference, error)
	CreateIrrigationSchedule(ctx context.Context, req dto.CreateIrrigationScheduleRequest, userID int64) (*db.IrrigationSchedule, error)
	ListIrrigationSchedules(ctx context.Context, userID int64) ([]db.IrrigationSchedule, error)
	GetIrrigationSchedule(ctx context.Context, scheduleUUID uuid.UUID, userID int64) (*db.IrrigationSchedule, error)
	UpdateIrrigationSchedule(ctx context.Context, scheduleUUID uuid.UUID, req dto.UpdateIrrigationScheduleRequest, userID int64) (*db.IrrigationSchedule, error)
	DeleteIrrigationSchedule(ctx context.Context, scheduleUUID uuid.UUID, userID int64) error
}

type irrigationService struct {
	store      db.Store
	mqttClient client.Client
}

func NewIrrigationService(store db.Store, mqttClient client.Client) IrrigationService {
	return &irrigationService{
		store:      store,
		mqttClient: mqttClient,
	}
}

func (s *irrigationService) CreateIrrigationCommand(ctx context.Context, req dto.CreateIrrigationCommandRequest, userID int64) (*db.IrrigationCommand, error) {
	deviceUUID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		return nil, err
	}

	device, err := s.store.GetDevice(ctx, deviceUUID)
	if err != nil {
		return nil, err
	}

	if !device.UserID.Valid || device.UserID.Int64 != userID {
		return nil, errors.New("device doesn't belong to authenticated user")
	}

	if !strings.EqualFold(strings.TrimSpace(device.Status), "ativo") {
		return nil, errors.New("device is inactive, cannot send command")
	}

	pending, err := s.store.ExistsPendingIrrigationCommand(ctx, device.ID)
	if err != nil {
		return nil, err
	}

	if pending {
		return nil, errors.New("device already has a pending command")
	}

	active, err := s.store.ExistsActiveIrrigationAction(ctx, device.ID)
	if err != nil {
		return nil, err
	}

	switch req.Action {
	case "START":
		if active {
			return nil, errors.New("device is already irrigating")
		}
	case "STOP":
		if !active {
			return nil, errors.New("device is not irrigating")
		}
	}

	var durationSeconds sql.NullInt32
	if req.Duration != nil {
		durationSeconds = sql.NullInt32{Int32: *req.Duration, Valid: true}
	}

	command, err := s.store.CreateIrrigationCommand(ctx, db.CreateIrrigationCommandParams{
		DeviceID:        device.ID,
		UserID:          userID,
		Action:          req.Action,
		DurationSeconds: durationSeconds,
	})
	if err != nil {
		return nil, err
	}

	payload := dto.IrrigationCommandPayload{
		CommandID: command.Uuid.String(),
		Action:    command.Action,
	}
	if durationSeconds.Valid {
		payload.DurationSeconds = &durationSeconds.Int32
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	if err := publisher.PublishCommand(s.mqttClient, device.DeviceUid, data); err != nil {
		_, updateErr := s.store.UpdateIrrigationCommandStatus(context.Background(), db.UpdateIrrigationCommandStatusParams{
			Uuid:   command.Uuid,
			Status: "FAILED",
			ErrorMessage: sql.NullString{
				String: err.Error(),
				Valid:  true,
			},
		})
		if updateErr != nil {
			return nil, updateErr
		}
		return nil, err
	}

	return &command, nil
}

func (s *irrigationService) GetIrrigationCommand(ctx context.Context, commandUUID uuid.UUID) (*db.IrrigationCommand, error) {
	command, err := s.store.GetIrrigationCommand(ctx, commandUUID)
	if err != nil {
		return nil, err
	}
	return &command, nil
}

func (s *irrigationService) ListIrrigationHistory(ctx context.Context, userID int64, date time.Time, deviceUUID *uuid.UUID) ([]db.IrrigationAction, error) {
	deviceID, err := s.resolveDeviceID(ctx, userID, deviceUUID)
	if err != nil {
		return nil, err
	}

	if date.IsZero() {
		date = time.Now().In(time.Local)
	}
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	actions, err := s.loadAllIrrigationActions(ctx, userID)
	if err != nil {
		return nil, err
	}

	filtered := make([]db.IrrigationAction, 0, len(actions))
	for _, action := range actions {
		if deviceID != nil && action.DeviceID != *deviceID {
			continue
		}

		startedAt := action.StartedAt.In(dayStart.Location())
		if startedAt.Before(dayStart) || !startedAt.Before(dayEnd) {
			continue
		}

		filtered = append(filtered, action)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].StartedAt.Equal(filtered[j].StartedAt) {
			return filtered[i].ID > filtered[j].ID
		}
		return filtered[i].StartedAt.After(filtered[j].StartedAt)
	})

	return filtered, nil
}

func (s *irrigationService) GetIrrigationHistory(ctx context.Context, actionUUID uuid.UUID) (*db.IrrigationAction, error) {
	action, err := s.store.GetIrrigationAction(ctx, actionUUID)
	if err != nil {
		return nil, err
	}
	return &action, nil
}

func (s *irrigationService) GetWaterConsumption(ctx context.Context, userID int64, period string, deviceUUID *uuid.UUID) (dto.WaterConsumptionResponse, error) {
	normalizedPeriod := strings.ToLower(strings.TrimSpace(period))
	if normalizedPeriod == "" {
		normalizedPeriod = "week"
	}

	deviceID, err := s.resolveDeviceID(ctx, userID, deviceUUID)
	if err != nil {
		return dto.WaterConsumptionResponse{}, err
	}

	window, err := buildConsumptionWindow(time.Now().In(time.Local), normalizedPeriod)
	if err != nil {
		return dto.WaterConsumptionResponse{}, err
	}

	actions, err := s.loadAllIrrigationActions(ctx, userID)
	if err != nil {
		return dto.WaterConsumptionResponse{}, err
	}

	points := make([]dto.WaterConsumptionPoint, len(window.buckets))
	for i, bucket := range window.buckets {
		points[i] = dto.WaterConsumptionPoint{Label: bucket.label}
	}

	var currentTotal int64
	var previousTotal int64

	for _, action := range actions {
		if deviceID != nil && action.DeviceID != *deviceID {
			continue
		}

		if !action.WaterVolumeMl.Valid || !strings.EqualFold(action.Status, irrigationActionFinished) {
			continue
		}

		volume := int64(action.WaterVolumeMl.Int32)
		startedAt := action.StartedAt.In(window.currentStart.Location())

		if startedAt.Before(window.currentStart) == false && startedAt.Before(window.currentEnd) {
			currentTotal += volume
			bucketIndex := int(startedAt.Sub(window.currentStart) / window.bucketDuration)
			if bucketIndex >= 0 && bucketIndex < len(points) {
				points[bucketIndex].WaterVolumeMl += volume
			}
			continue
		}

		if !startedAt.Before(window.previousStart) && startedAt.Before(window.previousEnd) {
			previousTotal += volume
		}
	}

	difference := currentTotal - previousTotal
	changePercent := 0.0
	if previousTotal != 0 {
		changePercent = (float64(difference) / float64(previousTotal)) * 100
		changePercent = math.Round(changePercent*100) / 100
	}

	return dto.WaterConsumptionResponse{
		Period:             normalizedPeriod,
		CurrentPeriodStart: window.currentStart,
		CurrentPeriodEnd:   window.currentEnd,
		CurrentTotalMl:     currentTotal,
		PreviousTotalMl:    previousTotal,
		DifferenceMl:       difference,
		ChangePercent:      changePercent,
		Points:             points,
	}, nil
}

func (s *irrigationService) CreateIrrigationPreference(ctx context.Context, req dto.CreateIrrigationPreferenceRequest) (*db.IrrigationPreference, error) {
	deviceUUID, err := uuid.Parse(req.DeviceUUID)
	if err != nil {
		return nil, err
	}

	device, err := s.store.GetDevice(ctx, deviceUUID)
	if err != nil {
		return nil, err
	}

	var startHour sql.NullTime
	if req.StartHour != nil {
		t, err := time.Parse("15:04", *req.StartHour)
		if err != nil {
			return nil, err
		}
		startHour = sql.NullTime{Time: t, Valid: true}
	}

	var endHour sql.NullTime
	if req.EndHour != nil {
		t, err := time.Parse("15:04", *req.EndHour)
		if err != nil {
			return nil, err
		}
		endHour = sql.NullTime{Time: t, Valid: true}
	}

	arg := db.CreateIrrigationPreferencesParams{
		DeviceID:             device.ID,
		IrrigationMode:       req.IrrigationMode,
		MoistureThreshold:    req.MoistureThreshold,
		DryTimeMinutes:       req.DryTimeMinutes,
		MaxIrrigationsPerDay: req.MaxIrrigationsPerDay,
		StartHour:            startHour,
		EndHour:              endHour,
	}

	preference, err := s.store.CreateIrrigationPreferences(ctx, arg)
	if err != nil {
		return nil, err
	}

	return &preference, nil
}

func (s *irrigationService) GetIrrigationPreference(ctx context.Context, preferenceUUID uuid.UUID) (*db.IrrigationPreference, error) {
	preference, err := s.store.GetIrrigationPreference(ctx, preferenceUUID)
	if err != nil {
		return nil, err
	}
	return &preference, nil
}

func (s *irrigationService) GetIrrigationPreferenceByDevice(ctx context.Context, deviceUUID uuid.UUID) (*db.IrrigationPreference, error) {
	device, err := s.store.GetDevice(ctx, deviceUUID)
	if err != nil {
		return nil, err
	}

	preference, err := s.store.GetIrrigationPreferenceByDevice(ctx, device.ID)
	if err != nil {
		return nil, err
	}
	return &preference, nil
}

const irrigationActionFinished = "FINALIZADO"

type consumptionBucket struct {
	label string
}

type consumptionWindow struct {
	currentStart   time.Time
	currentEnd     time.Time
	previousStart  time.Time
	previousEnd    time.Time
	bucketDuration time.Duration
	buckets        []consumptionBucket
}

func (s *irrigationService) loadAllIrrigationActions(ctx context.Context, userID int64) ([]db.IrrigationAction, error) {
	const pageSize int32 = 100

	var allActions []db.IrrigationAction
	var offset int32

	for {
		actions, err := s.store.ListIrrigationAction(ctx, db.ListIrrigationActionParams{
			UserID: userID,
			Limit:  pageSize,
			Offset: offset,
		})
		if err != nil {
			return nil, err
		}

		allActions = append(allActions, actions...)
		if int32(len(actions)) < pageSize {
			break
		}

		offset += pageSize
	}

	return allActions, nil
}

func (s *irrigationService) resolveDeviceID(ctx context.Context, userID int64, deviceUUID *uuid.UUID) (*int64, error) {
	if deviceUUID == nil {
		return nil, nil
	}

	device, err := s.store.GetDevice(ctx, *deviceUUID)
	if err != nil {
		return nil, err
	}

	if !device.UserID.Valid || device.UserID.Int64 != userID {
		return nil, errors.New("device doesn't belong to authenticated user")
	}

	return &device.ID, nil
}

func (s *irrigationService) CreateIrrigationSchedule(ctx context.Context, req dto.CreateIrrigationScheduleRequest, userID int64) (*db.IrrigationSchedule, error) {
	deviceUUID, err := uuid.Parse(req.DeviceUUID)
	if err != nil {
		return nil, err
	}
	device, err := s.store.GetDevice(ctx, deviceUUID)
	if err != nil {
		return nil, err
	}
	if !device.UserID.Valid || device.UserID.Int64 != userID {
		return nil, errors.New("device doesn't belong to authenticated user")
	}

	startTime, err := dto.ScheduleTime(req.StartTime)
	if err != nil {
		return nil, errors.New("start_time must use HH:MM format")
	}
	if err := validateScheduleDays(req.DaysOfWeek); err != nil {
		return nil, err
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	schedule, err := s.store.CreateIrrigationSchedule(ctx, db.CreateIrrigationScheduleParams{
		DeviceID:        device.ID,
		UserID:          userID,
		Name:            dto.ScheduleName(req.Name),
		Enabled:         enabled,
		StartTime:       startTime,
		DurationSeconds: req.DurationSeconds,
		DaysOfWeek:      req.DaysOfWeek,
	})
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (s *irrigationService) ListIrrigationSchedules(ctx context.Context, userID int64) ([]db.IrrigationSchedule, error) {
	return s.store.ListIrrigationSchedules(ctx, userID)
}

func (s *irrigationService) GetIrrigationSchedule(ctx context.Context, scheduleUUID uuid.UUID, userID int64) (*db.IrrigationSchedule, error) {
	schedule, err := s.store.GetIrrigationSchedule(ctx, db.GetIrrigationScheduleParams{Uuid: scheduleUUID, UserID: userID})
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (s *irrigationService) UpdateIrrigationSchedule(ctx context.Context, scheduleUUID uuid.UUID, req dto.UpdateIrrigationScheduleRequest, userID int64) (*db.IrrigationSchedule, error) {
	startTime, err := dto.ScheduleTime(req.StartTime)
	if err != nil {
		return nil, errors.New("start_time must use HH:MM format")
	}
	if err := validateScheduleDays(req.DaysOfWeek); err != nil {
		return nil, err
	}

	schedule, err := s.store.UpdateIrrigationSchedule(ctx, db.UpdateIrrigationScheduleParams{
		Uuid:            scheduleUUID,
		UserID:          userID,
		Name:            dto.ScheduleName(req.Name),
		Enabled:         req.Enabled,
		StartTime:       startTime,
		DurationSeconds: req.DurationSeconds,
		DaysOfWeek:      req.DaysOfWeek,
	})
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (s *irrigationService) DeleteIrrigationSchedule(ctx context.Context, scheduleUUID uuid.UUID, userID int64) error {
	_, err := s.store.DeleteIrrigationSchedule(ctx, db.DeleteIrrigationScheduleParams{Uuid: scheduleUUID, UserID: userID})
	return err
}

func validateScheduleDays(value string) error {
	parts := strings.Split(value, ",")
	if len(parts) == 0 || len(parts) > 7 {
		return errors.New("days_of_week must contain between 1 and 7 unique days from 0 to 6")
	}
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) != 1 || part[0] < '0' || part[0] > '6' {
			return errors.New("days_of_week must contain days from 0 to 6 separated by commas")
		}
		if _, ok := seen[part]; ok {
			return errors.New("days_of_week cannot contain duplicate days")
		}
		seen[part] = struct{}{}
	}
	return nil
}

func buildConsumptionWindow(now time.Time, period string) (consumptionWindow, error) {
	switch period {
	case "day":
		currentStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		currentEnd := currentStart.Add(24 * time.Hour)
		return consumptionWindow{
			currentStart:   currentStart,
			currentEnd:     currentEnd,
			previousStart:  currentStart.Add(-24 * time.Hour),
			previousEnd:    currentStart,
			bucketDuration: time.Hour,
			buckets:        buildHourlyBuckets(currentStart, 24),
		}, nil
	case "week":
		currentStart := startOfISOWeek(now)
		currentEnd := currentStart.Add(7 * 24 * time.Hour)
		return consumptionWindow{
			currentStart:   currentStart,
			currentEnd:     currentEnd,
			previousStart:  currentStart.Add(-7 * 24 * time.Hour),
			previousEnd:    currentStart,
			bucketDuration: 24 * time.Hour,
			buckets:        buildDailyBuckets(currentStart, 7),
		}, nil
	case "month":
		currentStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		currentEnd := currentStart.AddDate(0, 1, 0)
		bucketCount := int(currentEnd.Sub(currentStart) / (24 * time.Hour))
		return consumptionWindow{
			currentStart:   currentStart,
			currentEnd:     currentEnd,
			previousStart:  currentStart.AddDate(0, -1, 0),
			previousEnd:    currentStart,
			bucketDuration: 24 * time.Hour,
			buckets:        buildDailyBuckets(currentStart, bucketCount),
		}, nil
	default:
		return consumptionWindow{}, fmt.Errorf("invalid period: %s", period)
	}
}

func startOfISOWeek(now time.Time) time.Time {
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := now.AddDate(0, 0, -(weekday - 1))
	return time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, now.Location())
}

func buildHourlyBuckets(start time.Time, count int) []consumptionBucket {
	buckets := make([]consumptionBucket, 0, count)
	for i := 0; i < count; i++ {
		bucketStart := start.Add(time.Duration(i) * time.Hour)
		buckets = append(buckets, consumptionBucket{label: bucketStart.Format("15h")})
	}
	return buckets
}

func buildDailyBuckets(start time.Time, count int) []consumptionBucket {
	weekdayLabels := map[time.Weekday]string{
		time.Monday:    "Seg",
		time.Tuesday:   "Ter",
		time.Wednesday: "Qua",
		time.Thursday:  "Qui",
		time.Friday:    "Sex",
		time.Saturday:  "Sab",
		time.Sunday:    "Dom",
	}

	buckets := make([]consumptionBucket, 0, count)
	for i := 0; i < count; i++ {
		bucketStart := start.AddDate(0, 0, i)
		label := bucketStart.Format("02")
		if count == 7 {
			label = weekdayLabels[bucketStart.Weekday()]
		}
		buckets = append(buckets, consumptionBucket{label: label})
	}
	return buckets
}
