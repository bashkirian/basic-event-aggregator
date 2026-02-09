package models

import "time"

// Event представляет входящее событие
type Event struct {
    ID        string    `json:"id"`
    Type      string    `json:"type"`
    UserID    string    `json:"user_id"`
    Value     float64   `json:"value"`
    Timestamp time.Time `json:"timestamp"`
}

// AggregatedData результат агрегации
type AggregatedData struct {
    UserID     string    `json:"user_id"`
    EventType  string    `json:"event_type"`
    Count      int64     `json:"count"`
    TotalValue float64   `json:"total_value"`
    AvgValue   float64   `json:"avg_value"`
    MinValue   float64   `json:"min_value"`
    MaxValue   float64   `json:"max_value"`
    StartTime  time.Time `json:"start_time"`
    EndTime    time.Time `json:"end_time"`
}
