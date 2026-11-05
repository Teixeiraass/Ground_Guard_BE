package dto

import (
	"time"

	"github.com/google/uuid"
)

const (
	SoilHealthExcellent = "EXCELLENT"
	SoilHealthGood      = "GOOD"
	SoilHealthAttention = "ATTENTION"
	SoilHealthCritical  = "CRITICAL"
	SoilHealthNoData    = "NO_DATA"
	SoilHealthNoDevices = "NO_DEVICES"

	SensorStateOK      = "OK"
	SensorStateDry     = "DRY"
	SensorStateWet     = "WET"
	SensorStateOffline = "OFFLINE"
	SensorStateFault   = "FAULT"
	SensorStateNoData  = "NO_DATA"
)

type SoilHealthSummary struct {
	Total   int `json:"total"`
	OK      int `json:"ok"`
	Dry     int `json:"dry"`
	Wet     int `json:"wet"`
	Offline int `json:"offline"`
	Fault   int `json:"fault"`
	NoData  int `json:"no_data"`
}

type SensorHealthResponse struct {
	Uuid            uuid.UUID  `json:"uuid"`
	Name            string     `json:"name"`
	State           string     `json:"state"`
	SoilMoisture    *int32     `json:"soil_moisture"`
	IdealMin        int32      `json:"ideal_min"`
	IdealMax        int32      `json:"ideal_max"`
	TimeInRange24h  *int       `json:"time_in_range_24h"`
	Readings24h     int        `json:"readings_24h"`
	Score           *int       `json:"score"`
	LastSeen        *time.Time `json:"last_seen"`
}

type SoilHealthResponse struct {
	Status      string                 `json:"status"`
	Score       *int                   `json:"score"`
	AvgMoisture *int                   `json:"avg_moisture"`
	Summary     SoilHealthSummary      `json:"summary"`
	Sensors     []SensorHealthResponse `json:"sensors"`
	GeneratedAt time.Time              `json:"generated_at"`
}
