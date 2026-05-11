package util

import (
	"database/sql"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

func ToUUID(value uuid.NullUUID) uuid.UUID {
	return value.UUID
}

func ToNullString(value string) sql.NullString {
	return sql.NullString{
		String: value,
		Valid:  value != "",
	}
}

func ToNullTime(value time.Time) sql.NullTime {
	return sql.NullTime{
		Time:  value,
		Valid: !value.IsZero(),
	}
}

func ToInet(value string) pqtype.Inet {
	ip := net.ParseIP(value)

	return pqtype.Inet{
		IPNet: net.IPNet{
			IP:   ip,
			Mask: net.CIDRMask(32, 32),
		},
		Valid: ip != nil,
	}
}