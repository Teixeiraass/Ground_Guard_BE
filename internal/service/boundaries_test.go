package service_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	mockdb "github.com/Teixeiraass/ground_guard_be/db/mock"
	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/dto"
	"github.com/Teixeiraass/ground_guard_be/internal/service"
	"github.com/Teixeiraass/ground_guard_be/websocket"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

// Exact-threshold cases: these pin down every <, <=, >, >= in the rules.

func TestIdealRangeThresholdBoundaries(t *testing.T) {
	cases := []struct {
		threshold int32
		wantMin   int32
	}{
		{0, 35},  // not configured
		{1, 1},   // smallest valid threshold is used
		{74, 74}, // just below max-5
		{75, 35}, // max-5 itself would leave a band of 5 points: ignored
		{-3, 35}, // garbage
	}
	for _, c := range cases {
		min, max := service.IdealMoistureRange(&db.IrrigationPreference{MoistureThreshold: c.threshold})
		require.Equal(t, c.wantMin, min, "threshold %d", c.threshold)
		require.Equal(t, int32(80), max)
	}
	min, max := service.IdealMoistureRange(nil)
	require.Equal(t, int32(35), min)
	require.Equal(t, int32(80), max)
}

func TestMoistureExactlyOnTheIdealEdgesIsOK(t *testing.T) {
	for _, m := range []int32{35, 80} {
		rsp := service.ComputeSoilHealth([]service.SensorHealthInput{
			{Device: liveDevice(m), History: history(m, m, m)},
		}, soilNow)
		require.Equal(t, dto.SensorStateOK, rsp.Sensors[0].State, "moisture %d", m)
		require.Equal(t, 100, *rsp.Sensors[0].TimeInRange24h, "readings at %d are in range", m)
	}
	for m, state := range map[int32]string{34: dto.SensorStateDry, 81: dto.SensorStateWet} {
		rsp := service.ComputeSoilHealth([]service.SensorHealthInput{{Device: liveDevice(m)}}, soilNow)
		require.Equal(t, state, rsp.Sensors[0].State, "moisture %d", m)
		// One point outside: 70 - 3*1.
		require.Equal(t, 67, *rsp.Sensors[0].Score)
	}
}

func TestScoreFloorsAtZeroWhenFarOutOfRange(t *testing.T) {
	rsp := service.ComputeSoilHealth([]service.SensorHealthInput{{Device: liveDevice(0)}}, soilNow)
	require.Equal(t, 0, *rsp.Sensors[0].Score, "70 - 3*35 would be negative")

	rsp = service.ComputeSoilHealth([]service.SensorHealthInput{{Device: liveDevice(100)}}, soilNow)
	require.Equal(t, 10, *rsp.Sensors[0].Score, "70 - 3*20")
}

func TestHistoryIsUsedFromExactlyThreeReadings(t *testing.T) {
	two := service.ComputeSoilHealth([]service.SensorHealthInput{
		{Device: liveDevice(50), History: history(10, 10)},
	}, soilNow)
	require.Equal(t, 100, *two.Sensors[0].Score, "2 readings: current state decides")

	three := service.ComputeSoilHealth([]service.SensorHealthInput{
		{Device: liveDevice(50), History: history(10, 10, 50)},
	}, soilNow)
	require.Equal(t, 33, *three.Sensors[0].Score, "3 readings: time in range decides")
}

func TestOutOfRangeScoreCeilingIsExactlySeventy(t *testing.T) {
	rsp := service.ComputeSoilHealth([]service.SensorHealthInput{
		{Device: liveDevice(30), History: history(50, 50, 50)},
	}, soilNow)
	require.Equal(t, 70, *rsp.Sensors[0].Score)

	under := service.ComputeSoilHealth([]service.SensorHealthInput{
		{Device: liveDevice(30), History: history(50, 10, 10)},
	}, soilNow)
	require.Equal(t, 33, *under.Sensors[0].Score, "a score already below the ceiling isn't raised")
}

func TestStatusThresholds(t *testing.T) {
	cases := map[int]string{
		100: dto.SoilHealthExcellent,
		85:  dto.SoilHealthExcellent,
		84:  dto.SoilHealthGood,
		65:  dto.SoilHealthGood,
		64:  dto.SoilHealthAttention,
		40:  dto.SoilHealthAttention,
		39:  dto.SoilHealthCritical,
	}
	for score, want := range cases {
		// 100 readings make time-in-range equal to the wanted score.
		values := make([]int32, 100)
		for i := range values {
			if i < score {
				values[i] = 50
			}
		}
		rsp := service.ComputeSoilHealth([]service.SensorHealthInput{
			{Device: liveDevice(50), History: history(values...)},
		}, soilNow)
		require.Equal(t, score, *rsp.Score)
		require.Equal(t, want, rsp.Status, "score %d", score)
	}
}

func TestOfflineAfterExactlyFifteenMinutes(t *testing.T) {
	justInside := liveDevice(50)
	justInside.LastSeen = sql.NullTime{Time: soilNow.Add(-service.SensorOfflineAfter), Valid: true}
	justOutside := liveDevice(50)
	justOutside.LastSeen = sql.NullTime{Time: soilNow.Add(-service.SensorOfflineAfter - time.Second), Valid: true}

	rsp := service.ComputeSoilHealth([]service.SensorHealthInput{{Device: justInside}, {Device: justOutside}}, soilNow)
	require.Equal(t, dto.SensorStateOK, rsp.Sensors[0].State)
	require.Equal(t, dto.SensorStateOffline, rsp.Sensors[1].State)
	require.Equal(t, 1, rsp.Summary.OK)
	require.Equal(t, 1, rsp.Summary.Offline)
}

func TestAverageMoistureRoundsAndCountsOnlyLiveSensors(t *testing.T) {
	rsp := service.ComputeSoilHealth([]service.SensorHealthInput{
		{Device: liveDevice(50)},
		{Device: liveDevice(51)},
		{Device: liveDevice(20)}, // dry still counts towards the water level
	}, soilNow)
	require.Equal(t, 40, *rsp.AvgMoisture, "(50+51+20)/3 = 40.33")
	require.Equal(t, 3, rsp.Summary.Total)
	require.Equal(t, 1, rsp.Summary.Dry)
}

func TestGetSoilHealthPropagatesUnexpectedPreferenceErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ListDevicesByUserID(gomock.Any(), gomock.Any()).Return([]db.Device{{ID: 1}}, nil)
	store.EXPECT().GetIrrigationPreferenceByDevice(gomock.Any(), int64(1)).Return(db.IrrigationPreference{}, errors.New("db down"))

	_, err := service.NewSoilHealthService(store).GetSoilHealth(context.Background(), 1)
	require.Error(t, err)
}

func TestSupportMessageLengthBoundary(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().CreateSupportMessage(gomock.Any(), gomock.Any()).Times(1).Return(db.SupportMessage{}, nil)

	svc, _ := newTestSupportService(store)
	_, err := svc.SendUserMessage(context.Background(), 1, strings.Repeat("é", service.SupportMessageMaxLength))
	require.NoError(t, err, "exactly the maximum is accepted")
	_, err = svc.SendUserMessage(context.Background(), 1, strings.Repeat("é", service.SupportMessageMaxLength+1))
	require.ErrorIs(t, err, service.ErrSupportMessageTooLong)
}

func TestSupportReplyPreviewBoundary(t *testing.T) {
	for _, n := range []int{120, 121} {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		body := strings.Repeat("a", n)
		store.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(db.User{ID: 7}, nil)
		store.EXPECT().CreateSupportMessage(gomock.Any(), gomock.Any()).Return(db.SupportMessage{Body: body}, nil)

		svc, notifier := newTestSupportService(store)
		_, err := svc.ReplyToUser(context.Background(), 1, [16]byte{1}, body)
		require.NoError(t, err)
		if n == 120 {
			require.Equal(t, body, notifier.body, "120 chars fit untouched")
		} else {
			require.Equal(t, strings.Repeat("a", 120)+"…", notifier.body)
		}
	}
}

type failingNotifier struct{ fakeNotifier }

func (f *failingNotifier) CreateAndDeliver(context.Context, int64, string, string, string) error {
	return errors.New("push down")
}

func TestReplyStillSucceedsWhenNotificationFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(db.User{ID: 7}, nil)
	store.EXPECT().CreateSupportMessage(gomock.Any(), gomock.Any()).Return(db.SupportMessage{Body: "oi"}, nil)

	svc := service.NewSupportService(store, websocket.NewHub(), &failingNotifier{})
	_, err := svc.ReplyToUser(context.Background(), 1, [16]byte{1}, "oi")
	require.NoError(t, err, "the reply is saved even if the push fails")
}
