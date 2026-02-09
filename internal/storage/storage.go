package storage

import (
    "sync"
    "time"

    "github.com/bashkirian/event-aggregator/pkg/models"
)

type Storage interface {
    AddEvent(event models.Event)
    GetAggregated(userID, eventType string, from, to time.Time) *models.AggregatedData
    GetAllAggregated() []models.AggregatedData
}

type InMemoryStorage struct {
    mu     sync.RWMutex
    events []models.Event
}

func NewInMemoryStorage() *InMemoryStorage {
    return &InMemoryStorage{
        events: make([]models.Event, 0),
    }
}

func (s *InMemoryStorage) AddEvent(event models.Event) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.events = append(s.events, event)
}

func (s *InMemoryStorage) GetAggregated(userID, eventType string, from, to time.Time) *models.AggregatedData {
    s.mu.RLock()
    defer s.mu.RUnlock()

    var filtered []models.Event
    for _, e := range s.events {
        if (userID == "" || e.UserID == userID) &&
            (eventType == "" || e.Type == eventType) &&
            (from.IsZero() || e.Timestamp.After(from) || e.Timestamp.Equal(from)) &&
            (to.IsZero() || e.Timestamp.Before(to) || e.Timestamp.Equal(to)) {
            filtered = append(filtered, e)
        }
    }

    if len(filtered) == 0 {
        return nil
    }

    return aggregate(filtered, userID, eventType)
}

func (s *InMemoryStorage) GetAllAggregated() []models.AggregatedData {
    s.mu.RLock()
    defer s.mu.RUnlock()

    // Группируем по userID и eventType
    groups := make(map[string][]models.Event)
    for _, e := range s.events {
        key := e.UserID + ":" + e.Type
        groups[key] = append(groups[key], e)
    }

    result := make([]models.AggregatedData, 0, len(groups))
    for _, events := range groups {
        if len(events) > 0 {
            agg := aggregate(events, events[0].UserID, events[0].Type)
            result = append(result, *agg)
        }
    }

    return result
}

func aggregate(events []models.Event, userID, eventType string) *models.AggregatedData {
    if len(events) == 0 {
        return nil
    }

    agg := &models.AggregatedData{
        UserID:    userID,
        EventType: eventType,
        Count:     int64(len(events)),
        MinValue:  events[0].Value,
        MaxValue:  events[0].Value,
        StartTime: events[0].Timestamp,
        EndTime:   events[0].Timestamp,
    }

    for _, e := range events {
        agg.TotalValue += e.Value
        if e.Value < agg.MinValue {
            agg.MinValue = e.Value
        }
        if e.Value > agg.MaxValue {
            agg.MaxValue = e.Value
        }
        if e.Timestamp.Before(agg.StartTime) {
            agg.StartTime = e.Timestamp
        }
        if e.Timestamp.After(agg.EndTime) {
            agg.EndTime = e.Timestamp
        }
    }

    agg.AvgValue = agg.TotalValue / float64(agg.Count)
    return agg
}
