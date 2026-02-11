package aggregator

import (
    "context"
    "testing"
    "time"

    "github.com/bashkirian/event-aggregator/internal/storage"
    "github.com/bashkirian/event-aggregator/pkg/models"
)

func TestAggregator_ProcessEvent(t *testing.T) {
    store := storage.NewInMemoryStorage()
    agg := New(store, 10)
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    agg.Start(ctx)
    
    event := models.Event{
        ID:        "test-1",
        Type:      "click",
        UserID:    "user-1",
        Value:     100.0,
        Timestamp: time.Now(),
    }
    
    err := agg.ProcessEvent(event)
    if err != nil {
        t.Fatalf("Failed to process event: %v", err)
    }
    
    // Даем время на обработку
    time.Sleep(100 * time.Millisecond)
    
    data := agg.GetAggregatedData("user-1", "click", time.Time{}, time.Time{})
    if data == nil {
        t.Fatal("Expected aggregated data, got nil")
    }
    
    if data.Count != 1 {
        t.Errorf("Expected count 1, got %d", data.Count)
    }
}

func TestAggregator_MultipleEvents(t *testing.T) {
    store := storage.NewInMemoryStorage()
    agg := New(store, 100)
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    agg.Start(ctx)
    
    now := time.Now()
    events := []models.Event{
        {ID: "1", Type: "click", UserID: "user-1", Value: 100, Timestamp: now},
        {ID: "2", Type: "click", UserID: "user-1", Value: 200, Timestamp: now},
        {ID: "3", Type: "click", UserID: "user-1", Value: 150, Timestamp: now},
    }
    
    for _, e := range events {
        if err := agg.ProcessEvent(e); err != nil {
            t.Fatalf("Failed to process event: %v", err)
        }
    }
    
    // Даем время на обработку
    time.Sleep(100 * time.Millisecond)
    
    data := agg.GetAggregatedData("user-1", "click", time.Time{}, time.Time{})
    if data == nil {
        t.Fatal("Expected aggregated data, got nil")
    }
    
    if data.Count != 3 {
        t.Errorf("Expected count 3, got %d", data.Count)
    }
    
    if data.AvgValue != 150 {
        t.Errorf("Expected avg 150, got %.2f", data.AvgValue)
    }
}

func TestAggregator_GetAllAggregatedData(t *testing.T) {
    store := storage.NewInMemoryStorage()
    agg := New(store, 100)
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    agg.Start(ctx)
    
    now := time.Now()
    events := []models.Event{
        {ID: "1", Type: "click", UserID: "user-1", Value: 100, Timestamp: now},
        {ID: "2", Type: "view", UserID: "user-1", Value: 50, Timestamp: now},
        {ID: "3", Type: "click", UserID: "user-2", Value: 200, Timestamp: now},
    }
    
    for _, e := range events {
        if err := agg.ProcessEvent(e); err != nil {
            t.Fatalf("Failed to process event: %v", err)
        }
    }
    
    time.Sleep(100 * time.Millisecond)
    
    results := agg.GetAllAggregatedData()
    if len(results) != 3 {
        t.Errorf("Expected 3 aggregations, got %d", len(results))
    }
}
