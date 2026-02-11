package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
	"context"

	"github.com/bashkirian/event-aggregator/pkg/server" // Импортируем main для запуска сервера
	"github.com/bashkirian/event-aggregator/pkg/models"
)

// Тестовый сервер и клиент
var serverURL = "http://localhost:8080"

func TestEventAggregationFlow(t *testing.T) {
	// 1. Запускаем сервер в фоне
	srv := server.NewServer("8080")
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server failed: %v", err)
		}
	}()

	// 2. Отправляем событие через API
	event := models.Event{
		Type:   "purchase",
		UserID: "user-123",
		Value:  299.99,
	}
	eventJSON, _ := json.Marshal(event)
	req, _ := http.NewRequest("POST", serverURL+"/events", bytes.NewBuffer(eventJSON))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected 201 Created, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 3. Ждём немного, чтобы агрегатор обработал событие
	time.Sleep(500 * time.Millisecond)

	// 4. Получаем агрегированные данные
	aggReq, _ := http.NewRequest("GET", serverURL+"/aggregated?user_id=user-123&type=purchase", nil)
	aggResp, err := client.Do(aggReq)
	if err != nil {
		t.Fatalf("Failed to get aggregated data: %v", err)
	}
	if aggResp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", aggResp.StatusCode)
	}

	var aggData models.AggregatedData
	if err := json.NewDecoder(aggResp.Body).Decode(&aggData); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	aggResp.Body.Close()

	// 5. Проверяем результаты
	if aggData.Count != 1 {
		t.Errorf("Expected count 1, got %d", aggData.Count)
	}
	if aggData.TotalValue != 299.99 {
		t.Errorf("Expected total 299.99, got %.2f", aggData.TotalValue)
	}
	if aggData.UserID != "user-123" {
		t.Errorf("Expected user_id 'user-123', got '%s'", aggData.UserID)
	}
	if aggData.EventType != "purchase" {
		t.Errorf("Expected type 'purchase', got '%s'", aggData.EventType)
	}

	// Останавливаем сервер после теста
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestHealthCheck(t *testing.T) {
	srv := server.NewServer("8080")
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server failed: %v", err)
		}
	}()
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(serverURL + "/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Health check returned %d, expected 200", resp.StatusCode)
	}

	var health map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}
	resp.Body.Close()

	if health["status"] != "ok" {
		t.Errorf("Health status is '%s', expected 'ok'", health["status"])
	}

	// Останавливаем сервер после теста
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}
