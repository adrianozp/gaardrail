package orchestrator

import (
	"context"
	"log"
	"time"
)

type Consumer interface {
	Consume() (string, error)
	Size() (int64, error)
}

type Orchestrator struct {
	DrainRate float64
	consumer  Consumer
}

func NewOrchestrator(c Consumer) *Orchestrator {
	return &Orchestrator{
		consumer: c,
	}
}

func (o *Orchestrator) SetDrainRate(drainRate float64) error {
	o.DrainRate = drainRate
	return nil
}

func (o Orchestrator) Start(tickRate int) error {
	ctx := context.Background()
	go o.run(ctx, tickRate)
	return nil
}

func (o *Orchestrator) run(ctx context.Context, tickRate int) {
	ticker := time.NewTicker(time.Duration(tickRate) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("orchestrator: drain loop shutting down")
			return
		case <-ticker.C:
		}

		queuelen, err := o.consumer.Size()
		if err != nil {
			log.Printf("orchestrator: error while getting queue length: %s", err.Error())
			continue
		}

		if queuelen == 0 {
			continue
		}

		messageId, err := o.consumer.Consume()
		if err != nil {
			log.Printf("orchestrator: error while consuming queue: %s", err.Error())
			continue
		}

		log.Printf("orchestrator: sucessful consumed message: %s", messageId)
	}
}
