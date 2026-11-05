package worker

import (
	"context"
	"log"
	"time"

	"github.com/Teixeiraass/ground_guard_be/internal/service"
)

type IrrigationScheduleWorker struct {
	irrigationService service.IrrigationService
}

func NewIrrigationScheduleWorker(irrigationService service.IrrigationService) *IrrigationScheduleWorker {
	return &IrrigationScheduleWorker{
		irrigationService: irrigationService,
	}
}

func (w *IrrigationScheduleWorker) Start() {
	ticker := time.NewTicker(20 * time.Second)

	go func() {
		for range ticker.C {
			w.process()
		}
	}()
}

func (w *IrrigationScheduleWorker) process() {
	ctx := context.Background()

	if err := w.irrigationService.ProcessUpcomingIrrigationNotifications(ctx); err != nil {
		log.Printf("schedule worker (upcoming notifications): %v", err)
	}
	if err := w.irrigationService.ProcessDueIrrigationSchedules(ctx); err != nil {
		log.Printf("schedule worker (due): %v", err)
	}
	if err := w.irrigationService.ProcessScheduledIrrigationCompletion(ctx); err != nil {
		log.Printf("schedule worker (completion): %v", err)
	}
}
