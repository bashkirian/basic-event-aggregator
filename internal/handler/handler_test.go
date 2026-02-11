package handler

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/bashkirian/event-aggregator/internal/aggregator"
    "github.com/bashkirian/event-aggregator/internal/storage"
    "github.com/bashkirian/event-aggregator/pkg/models"
)

func setupHandler() *Handler {
    store := storage.NewInMemoryStorage()
    agg := aggregator.New(store, 100)
    ctx := context.Background()
    agg.Start(ctx)
    return New(agg)
}

func TestHandler_HandlePostEvent(t *testing.T) {
    h := setupHandler()
    
    event := models.Event{
        Type:   "click",
        UserID: "user-1",
        Value:  100.0,
    }
    
    body, _ := json.Marshal(event)
    req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body))
    w := httptest.NewRecorder()
    
    h.HandlePostEvent(w, req)
    
    if w.Code != http.StatusCreated {
        t.Errorf("Expected status 201, got %d", w.Code)
    }
    
    var response map[string]string
    json.NewDecoder(w.Body).Decode(&response)
    
    if response["status"] != "accepted" {
        t.Errorf("Expected status 'accepted', got '%s'", response["status"])
    }
    
    if response["id"] == "" {
        t.Error("Expected non-empty id")
    }
}

func TestHandler_HandlePostEvent_InvalidMethod(t *testing.T) {
    h := setupHandler()
    
    req := httptest.NewRequest(http.MethodGet, "/events", nil)
    w := httptest.NewRecorder()
    
    h.HandlePostEvent(w, req)
    
    if w.Code != http.StatusMethodNotAllowed {
        t.Errorf("Expected status 405, got %d", w.Code)
    }
}

func TestHandler_HandlePostEvent_MissingFields(t *testing.T) {
    h := setupHandler()
    
    event := models.Event{
        Value: 100.0,
        // Missing UserID and Type
    }
    
    body, _ := json.Marshal(event)
    req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body))
    w := httptest.NewRecorder()
    
    h.HandlePostEvent(w, req)
    
    if w.Code != http.StatusBadRequest {
        t.Errorf("Expected status 400, got %d", w.Code)
    }
}

func TestHandler_HandleGetAggregated(t *testing.T) {
    h := setupHandler()
    
    // Сначала добавляем события
    events := []models.Event{
        {ID: "1", Type: "click", UserID: "user-1", Value: 100, Timestamp: time.Now()},
        {ID: "2", Type: "click", UserID: "user-1", Value: 200, Timestamp: time.Now()},
    }
    
    for _, e := range events {
        h.aggregator.ProcessEvent(e)
    }
    
    time.Sleep(100 * time.Millisecond)
    
    req := httptest.NewRequest(http.MethodGet, "/aggregated?user_id=user-1&type=click", nil)
    w := httptest.NewRecorder()
    
    h.HandleGetAggregated(w, req)
    
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
    
    var agg models.AggregatedData
    json.NewDecoder(w.Body).Decode(&agg)
    
    if agg.Count != 2 {
        t.Errorf("Expected count 2, got %d", agg.Count)
    }
    
    if agg.TotalValue != 300 {
        t.Errorf("Expected total 300, got %.2f", agg.TotalValue)
    }
}

func TestHandler_HandleGetAllAggregated(t *testing.T) {
    h := setupHandler()
    
    events := []models.Event{
        {ID: "1", Type: "click", UserID: "user-1", Value: 100, Timestamp: time.Now()},
        {ID: "2", Type: "view", UserID: "user-1", Value: 50, Timestamp: time.Now()},
        {ID: "3", Type: "click", UserID: "user-2", Value: 200, Timestamp: time.Now()},
    }
    
    for _, e := range events {
        h.aggregator.ProcessEvent(e)
    }
    
    time.Sleep(100 * time.Millisecond)
    
    req := httptest.NewRequest(http.MethodGet, "/aggregated/all", nil)
    w := httptest.NewRecorder()
    
    h.HandleGetAllAggregated(w, req)
    
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
    
    var results []models.AggregatedData
    json.NewDecoder(w.Body).Decode(&results)
    
    if len(results) != 3 {
        t.Errorf("Expected 3 results, got %d", len(results))
    }
}

func TestHandler_HandleHealth(t *testing.T) {
    h := setupHandler()
    
    req := httptest.NewRequest(http.MethodGet, "/health", nil)
    w := httptest.NewRecorder()
    
    h.HandleHealth(w, req)
    
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
    
    var response map[string]string
    json.NewDecoder(w.Body).Decode(&response)
    
    if response["status"] != "ok" {
        t.Errorf("Expected status 'ok', got '%s'", response["status"])
    }
}
