package handler

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

func parseDateQuery(dateValue string) (time.Time, error) {
	dateValue = strings.TrimSpace(dateValue)
	if dateValue == "" {
		return time.Now().In(time.Local), nil
	}

	return time.ParseInLocation("2006-01-02", dateValue, time.Local)
}

func parseUUIDQuery(value string) (*uuid.UUID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	parsed, err := uuid.Parse(value)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}
