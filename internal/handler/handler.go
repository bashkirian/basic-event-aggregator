package handler

import (
    "encoding/json"
    "net/http"
    "time"

    "github.com/google/uuid"
    "github.com/bashkirian/event-aggregator/internal/aggregator"
    "github.com/bashkirian/event-aggregator/pkg/models"
)

type Handler struct {
    aggregator *aggregator.Aggregator
}

func New(agg *aggregator.Aggregator) *Handler {
    return &Handler{aggregator: agg}
}

// POST /events - отправить событие
func (h *Handler) HandlePostEvent(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var event models.Event
    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Валидация
    if event.UserID == "" || event.Type == "" {
        http.Error(w, "user_id and type are required", http.StatusBadRequest)
        return
    }

    // Генерируем ID и timestamp если не указаны
    if event.ID == "" {
        event.ID = uuid.New().String()
    }
    if event.Timestamp.IsZero() {
        event.Timestamp = time.Now()
    }

    if err := h.aggregator.ProcessEvent(event); err != nil {
        http.Error(w, "Failed to process event", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{
        "id":     event.ID,
        "status": "accepted",
    })
}

// GET /aggregated - получить агрегированные данные
func (h *Handler) HandleGetAggregated(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    query := r.URL.Query()
    userID := query.Get("user_id")
    eventType := query.Get("type")

    var from, to time.Time
    if fromStr := query.Get("from"); fromStr != "" {
        from, _ = time.Parse(time.RFC3339, fromStr)
    }
    if toStr := query.Get("to"); toStr != "" {
        to, _ = time.Parse(time.RFC3339, toStr)
    }

    data := h.aggregator.GetAggregatedData(userID, eventType, from, to)
    
    w.Header().Set("Content-Type", "application/json")
    if data == nil {
        json.NewEncoder(w).Encode(map[string]string{"message": "no data found"})
        return
    }
    json.NewEncoder(w).Encode(data)
}

// GET /aggregated/all - получить все агрегации
func (h *Handler) HandleGetAllAggregated(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    data := h.aggregator.GetAllAggregatedData()
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(data)
}

// GET /health - healthcheck
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
