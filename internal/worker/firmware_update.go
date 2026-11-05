package worker

import (
	"context"
	"log"
	"time"
)

type firmwareMaintainer interface {
	RunMaintenance(ctx context.Context, now time.Time) error
}

// FirmwareUpdateWorker times out stuck OTA updates and, inside the
// overnight window, starts updates for controllers with auto-update on.
type FirmwareUpdateWorker struct {
	firmware firmwareMaintainer
}

func NewFirmwareUpdateWorker(firmware firmwareMaintainer) *FirmwareUpdateWorker {
	return &FirmwareUpdateWorker{firmware: firmware}
}

func (w *FirmwareUpdateWorker) Start() {
	ticker := time.NewTicker(5 * time.Minute)

	go func() {
		for range ticker.C {
			if err := w.firmware.RunMaintenance(context.Background(), time.Now()); err != nil {
				log.Printf("firmware update worker: %v", err)
			}
		}
	}()
}
