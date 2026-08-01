package worker

import (
	"context"
	"log"
	"time"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
)

type IrrigationTimeoutWorker struct {
	store db.Store
}

func NewIrrigationTimeoutWorker(store db.Store) *IrrigationTimeoutWorker {
	return &IrrigationTimeoutWorker{
		store: store,
	}
}

func (w *IrrigationTimeoutWorker) Start() {
	ticker := time.NewTicker(10 * time.Second)

	go func() {
		for range ticker.C {
			w.process()
		}
	}()
}

func (w *IrrigationTimeoutWorker) process() {
	ctx := context.Background()

	if err := w.store.FailTimedOutCommands(ctx); err != nil {
		log.Printf("timeout worker: %v", err)
	}
}
