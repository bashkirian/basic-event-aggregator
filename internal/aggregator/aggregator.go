package aggregator

import (
    "context"
    "log"
    "time"
	"fmt"
	
    "github.com/bashkirian/event-aggregator/internal/storage"
    "github.com/bashkirian/event-aggregator/pkg/models"
)

type Aggregator struct {
    storage    storage.Storage
    eventChan  chan models.Event
    bufferSize int
}

func New(storage storage.Storage, bufferSize int) *Aggregator {
    return &Aggregator{
        storage:    storage,
        eventChan:  make(chan models.Event, bufferSize),
        bufferSize: bufferSize,
    }
}

// Start запускает обработку событий
func (a *Aggregator) Start(ctx context.Context) {
    go func() {
        for {
            select {
            case event := <-a.eventChan:
                a.processEvent(event)
            case <-ctx.Done():
                log.Println("Aggregator stopping...")
                return
            }
        }
    }()
}

// ProcessEvent добавляет событие в очередь
func (a *Aggregator) ProcessEvent(event models.Event) error {
    select {
    case a.eventChan <- event:
        return nil
    case <-time.After(5 * time.Second):
        return fmt.Errorf("timeout adding event to queue")
    }
}

func (a *Aggregator) processEvent(event models.Event) {
    a.storage.AddEvent(event)
    log.Printf("Processed event: %s, user: %s, type: %s, value: %.2f",
        event.ID, event.UserID, event.Type, event.Value)
}

// GetAggregatedData возвращает агрегированные данные
func (a *Aggregator) GetAggregatedData(userID, eventType string, from, to time.Time) *models.AggregatedData {
    return a.storage.GetAggregated(userID, eventType, from, to)
}

// GetAllAggregatedData возвращает все агрегации
func (a *Aggregator) GetAllAggregatedData() []models.AggregatedData {
    return a.storage.GetAllAggregated()
}
