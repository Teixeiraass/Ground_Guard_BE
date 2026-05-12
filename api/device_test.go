package api

import (
	"database/sql"
	"testing"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/util"
)

func TestGetDeviceAPI(t *testing.T) {
	user, _ := randomUser(t)
	device := randomDevice(user.ID)
}

func randomDevice(userID int64) db.Device {
	return db.Device{
		ID: util.RandomInt(1, 1000),
		Uuid: util.RandomUuid(),
		DeviceUid: util.RandomString(10),
		Name: util.RandomString(6),
		FirmwareVersion: util.RandomString(8),
		Status: util.RandomStatus(),
		UserID: sql.NullInt64{
			Int64: userID,
			Valid: true,
		},
	}
}