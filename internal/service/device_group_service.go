package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/google/uuid"
)

type DeviceGroupService interface {
	CreateGroup(ctx context.Context, userID int64, name string) (*db.DeviceGroup, error)
	ListGroups(ctx context.Context, userID int64) ([]dto.DeviceGroupResponse, error)
	GetGroup(ctx context.Context, userID int64, groupUUID uuid.UUID) (*dto.DeviceGroupDetailResponse, error)
	RenameGroup(ctx context.Context, userID int64, groupUUID uuid.UUID, name string) (*db.DeviceGroup, error)
	DeleteGroup(ctx context.Context, userID int64, groupUUID uuid.UUID) error
	TriggerGroupIrrigation(ctx context.Context, userID int64, groupUUID uuid.UUID, action string) ([]dto.GroupIrrigationCommandResult, error)
	CreateGroupSchedule(ctx context.Context, userID int64, groupUUID uuid.UUID, req dto.CreateGroupScheduleRequest) ([]db.IrrigationSchedule, error)
	GetGroupStatistics(ctx context.Context, userID int64, groupUUID uuid.UUID, date time.Time, period string) (*dto.DeviceGroupStatisticsResponse, error)
}

type deviceGroupService struct {
	store             db.Store
	irrigationService IrrigationService
}

func NewDeviceGroupService(store db.Store, irrigationService IrrigationService) DeviceGroupService {
	return &deviceGroupService{store: store, irrigationService: irrigationService}
}

func (s *deviceGroupService) CreateGroup(ctx context.Context, userID int64, name string) (*db.DeviceGroup, error) {
	group, err := s.store.CreateDeviceGroup(ctx, db.CreateDeviceGroupParams{UserID: userID, Name: name})
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *deviceGroupService) ListGroups(ctx context.Context, userID int64) ([]dto.DeviceGroupResponse, error) {
	groups, err := s.store.ListDeviceGroupsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	response := make([]dto.DeviceGroupResponse, 0, len(groups))
	for _, group := range groups {
		devices, err := s.store.ListDevicesByGroupID(ctx, sql.NullInt64{Int64: group.ID, Valid: true})
		if err != nil {
			return nil, err
		}
		response = append(response, dto.NewDeviceGroupResponse(group, devices))
	}
	return response, nil
}

func (s *deviceGroupService) GetGroup(ctx context.Context, userID int64, groupUUID uuid.UUID) (*dto.DeviceGroupDetailResponse, error) {
	group, err := s.store.GetDeviceGroup(ctx, db.GetDeviceGroupParams{Uuid: groupUUID, UserID: userID})
	if err != nil {
		return nil, err
	}

	devices, err := s.store.ListDevicesByGroupID(ctx, sql.NullInt64{Int64: group.ID, Valid: true})
	if err != nil {
		return nil, err
	}

	response := dto.NewDeviceGroupDetailResponse(group, devices)
	return &response, nil
}

func (s *deviceGroupService) RenameGroup(ctx context.Context, userID int64, groupUUID uuid.UUID, name string) (*db.DeviceGroup, error) {
	group, err := s.store.UpdateDeviceGroupName(ctx, db.UpdateDeviceGroupNameParams{
		Uuid:   groupUUID,
		UserID: userID,
		Name:   name,
	})
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *deviceGroupService) DeleteGroup(ctx context.Context, userID int64, groupUUID uuid.UUID) error {
	_, err := s.store.DeleteDeviceGroup(ctx, db.DeleteDeviceGroupParams{Uuid: groupUUID, UserID: userID})
	return err
}

// TriggerGroupIrrigation fires the same manual irrigation command on every
// device in the group, reusing IrrigationService.CreateIrrigationCommand
// (same pending/active guards as a single-device command). One device
// failing (e.g. already irrigating) never aborts the others — each result
// is reported individually.
func (s *deviceGroupService) TriggerGroupIrrigation(ctx context.Context, userID int64, groupUUID uuid.UUID, action string) ([]dto.GroupIrrigationCommandResult, error) {
	group, err := s.store.GetDeviceGroup(ctx, db.GetDeviceGroupParams{Uuid: groupUUID, UserID: userID})
	if err != nil {
		return nil, err
	}

	devices, err := s.store.ListDevicesByGroupID(ctx, sql.NullInt64{Int64: group.ID, Valid: true})
	if err != nil {
		return nil, err
	}

	results := make([]dto.GroupIrrigationCommandResult, 0, len(devices))
	for _, device := range devices {
		_, err := s.irrigationService.CreateIrrigationCommand(ctx, dto.CreateIrrigationCommandRequest{
			DeviceID: device.Uuid.String(),
			Action:   action,
		}, userID)

		result := dto.GroupIrrigationCommandResult{DeviceUuid: device.Uuid, Success: err == nil}
		if err != nil {
			result.Error = err.Error()
		}
		results = append(results, result)
	}
	return results, nil
}

// CreateGroupSchedule creates one irrigation_schedule per device in the
// group (via IrrigationService.CreateIrrigationSchedule — same validation
// as a single-device schedule), then tags each with the group's id so the
// UI can show/manage them together.
func (s *deviceGroupService) CreateGroupSchedule(ctx context.Context, userID int64, groupUUID uuid.UUID, req dto.CreateGroupScheduleRequest) ([]db.IrrigationSchedule, error) {
	group, err := s.store.GetDeviceGroup(ctx, db.GetDeviceGroupParams{Uuid: groupUUID, UserID: userID})
	if err != nil {
		return nil, err
	}

	devices, err := s.store.ListDevicesByGroupID(ctx, sql.NullInt64{Int64: group.ID, Valid: true})
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, errors.New("group has no devices")
	}

	schedules := make([]db.IrrigationSchedule, 0, len(devices))
	for _, device := range devices {
		scheduleReq := req.ToScheduleRequest(device.Uuid.String())

		schedule, err := s.irrigationService.CreateIrrigationSchedule(ctx, scheduleReq, userID)
		if err != nil {
			return nil, err
		}

		if err := s.store.SetIrrigationScheduleGroup(ctx, db.SetIrrigationScheduleGroupParams{
			ID:      schedule.ID,
			GroupID: sql.NullInt64{Int64: group.ID, Valid: true},
		}); err != nil {
			return nil, err
		}
		schedule.GroupID = sql.NullInt64{Int64: group.ID, Valid: true}

		schedules = append(schedules, *schedule)
	}
	return schedules, nil
}

// GetGroupStatistics aggregates the same water-consumption/irrigation-history
// data the per-device statistics endpoint computes, but scoped to every
// device in the group, plus a live soil-moisture summary across members.
func (s *deviceGroupService) GetGroupStatistics(ctx context.Context, userID int64, groupUUID uuid.UUID, date time.Time, period string) (*dto.DeviceGroupStatisticsResponse, error) {
	group, err := s.store.GetDeviceGroup(ctx, db.GetDeviceGroupParams{Uuid: groupUUID, UserID: userID})
	if err != nil {
		return nil, err
	}

	devices, err := s.store.ListDevicesByGroupID(ctx, sql.NullInt64{Int64: group.ID, Valid: true})
	if err != nil {
		return nil, err
	}

	if date.IsZero() {
		date = time.Now().In(time.Local)
	}

	if len(devices) == 0 {
		// Grupo vazio: nada a agregar. Resolve aqui em vez de deixar
		// deviceIDs vazio virar "sem filtro" (que significaria "todos os
		// devices do usuário") lá na camada de consumo/histórico.
		consumption, err := s.irrigationService.GetWaterConsumptionForDevices(ctx, userID, period, []int64{-1})
		if err != nil {
			return nil, err
		}
		return &dto.DeviceGroupStatisticsResponse{
			GroupUuid:         groupUUID,
			Date:              date.Format("2006-01-02"),
			Period:            consumption.Period,
			WaterConsumption:  consumption,
			IrrigationHistory: []dto.IrrigationHistoryResponse{},
		}, nil
	}

	onlineCount, irrigatingCount := 0, 0
	deviceIDs := make([]int64, len(devices))
	var moistureSum int64
	var moistureReadings int
	var minMoisture, maxMoisture *int32

	for i, d := range devices {
		deviceIDs[i] = d.ID
		if d.IsOnline {
			onlineCount++
		}
		if d.IsIrrigating {
			irrigatingCount++
		}
		if d.SoilMoisture.Valid {
			value := d.SoilMoisture.Int32
			moistureSum += int64(value)
			moistureReadings++
			if minMoisture == nil || value < *minMoisture {
				minMoisture = &value
			}
			if maxMoisture == nil || value > *maxMoisture {
				maxMoisture = &value
			}
		}
	}

	var avgMoisture *float64
	if moistureReadings > 0 {
		avg := float64(moistureSum) / float64(moistureReadings)
		avgMoisture = &avg
	}

	history, err := s.irrigationService.ListIrrigationHistoryForDevices(ctx, userID, date, deviceIDs)
	if err != nil {
		return nil, err
	}

	consumption, err := s.irrigationService.GetWaterConsumptionForDevices(ctx, userID, period, deviceIDs)
	if err != nil {
		return nil, err
	}

	historyResponses := make([]dto.IrrigationHistoryResponse, 0, len(history))
	for _, action := range history {
		historyResponses = append(historyResponses, dto.NewIrrigationHistoryResponse(action))
	}

	return &dto.DeviceGroupStatisticsResponse{
		GroupUuid:         groupUUID,
		Date:              date.Format("2006-01-02"),
		Period:            consumption.Period,
		DeviceCount:       len(devices),
		OnlineCount:       onlineCount,
		IrrigatingCount:   irrigatingCount,
		AvgSoilMoisture:   avgMoisture,
		MinSoilMoisture:   minMoisture,
		MaxSoilMoisture:   maxMoisture,
		WaterConsumption:  consumption,
		IrrigationHistory: historyResponses,
	}, nil
}
