package storage

import (
    "testing"
    "time"

    "github.com/bashkirian/event-aggregator/pkg/models"
    "github.com/go-redis/redis/v8"
)

func TestInMemoryStorage_AddEvent(t *testing.T) {
    s := NewInMemoryStorage()
    
    event := models.Event{
        ID:        "test-1",
        Type:      "click",
        UserID:    "user-1",
        Value:     100.0,
        Timestamp: time.Now(),
    }
    
    s.AddEvent(event)
    
    if len(s.events) != 1 {
        t.Errorf("Expected 1 event, got %d", len(s.events))
    }
}

func TestInMemoryStorage_GetAggregated(t *testing.T) {
    s := NewInMemoryStorage()
    now := time.Now()
    
    events := []models.Event{
        {ID: "1", Type: "click", UserID: "user-1", Value: 100, Timestamp: now},
        {ID: "2", Type: "click", UserID: "user-1", Value: 200, Timestamp: now.Add(1 * time.Minute)},
        {ID: "3", Type: "click", UserID: "user-1", Value: 150, Timestamp: now.Add(2 * time.Minute)},
    }
    
    for _, e := range events {
        s.AddEvent(e)
    }
    
    agg := s.GetAggregated("user-1", "click", time.Time{}, time.Time{})
    
    if agg == nil {
        t.Fatal("Expected aggregated data, got nil")
    }
    
    if agg.Count != 3 {
        t.Errorf("Expected count 3, got %d", agg.Count)
    }
    
    if agg.TotalValue != 450 {
        t.Errorf("Expected total 450, got %.2f", agg.TotalValue)
    }
    
    if agg.AvgValue != 150 {
        t.Errorf("Expected avg 150, got %.2f", agg.AvgValue)
    }
    
    if agg.MinValue != 100 {
        t.Errorf("Expected min 100, got %.2f", agg.MinValue)
    }
    
    if agg.MaxValue != 200 {
        t.Errorf("Expected max 200, got %.2f", agg.MaxValue)
    }
}

func TestInMemoryStorage_GetAggregated_WithFilters(t *testing.T) {
    s := NewInMemoryStorage()
    now := time.Now()
    
    events := []models.Event{
        {ID: "1", Type: "click", UserID: "user-1", Value: 100, Timestamp: now},
        {ID: "2", Type: "view", UserID: "user-1", Value: 200, Timestamp: now},
        {ID: "3", Type: "click", UserID: "user-2", Value: 150, Timestamp: now},
    }
    
    for _, e := range events {
        s.AddEvent(e)
    }
    
    // Фильтр по user и type
    agg := s.GetAggregated("user-1", "click", time.Time{}, time.Time{})
    
    if agg == nil {
        t.Fatal("Expected aggregated data, got nil")
    }
    
    if agg.Count != 1 {
        t.Errorf("Expected count 1, got %d", agg.Count)
    }
    
    if agg.TotalValue != 100 {
        t.Errorf("Expected total 100, got %.2f", agg.TotalValue)
    }
}

func TestInMemoryStorage_GetAllAggregated(t *testing.T) {
    s := NewInMemoryStorage()
    now := time.Now()
    
    events := []models.Event{
        {ID: "1", Type: "click", UserID: "user-1", Value: 100, Timestamp: now},
        {ID: "2", Type: "click", UserID: "user-1", Value: 200, Timestamp: now},
        {ID: "3", Type: "view", UserID: "user-1", Value: 50, Timestamp: now},
        {ID: "4", Type: "click", UserID: "user-2", Value: 300, Timestamp: now},
    }
    
    for _, e := range events {
        s.AddEvent(e)
    }
    
    results := s.GetAllAggregated()
    
    // Ожидаем 3 группы: user-1:click, user-1:view, user-2:click
    if len(results) != 3 {
        t.Errorf("Expected 3 aggregations, got %d", len(results))
    }
}

func TestInMemoryStorage_Concurrency(t *testing.T) {
    s := NewInMemoryStorage()
    now := time.Now()
    
    // Запускаем 100 горутин, каждая добавляет 10 событий
    done := make(chan bool, 100)
    for i := 0; i < 100; i++ {
        go func(id int) {
            for j := 0; j < 10; j++ {
                s.AddEvent(models.Event{
                    ID:        string(rune(id*10 + j)),
                    Type:      "click",
                    UserID:    "user-1",
                    Value:     float64(j),
                    Timestamp: now,
                })
            }
            done <- true
        }(i)
    }
    
    // Ждем завершения всех горутин
    for i := 0; i < 100; i++ {
        <-done
    }
    
    // Проверяем что все 1000 событий добавлены
    if len(s.events) != 1000 {
        t.Errorf("Expected 1000 events, got %d", len(s.events))
    }
}

// setupRedisStorage создает тестовое Redis хранилище
func setupRedisStorage(t *testing.T) *RedisStorage {
    // Используем тестовую базу данных Redis (DB 15)
    r := NewRedisStorage("localhost:6379", "", 15)
    
    // Очищаем тестовую базу данных
    r.client.FlushDB(r.ctx)
    
    return r
}

func TestRedisStorage_AddEvent(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping Redis test in short mode")
    }
    
    r := setupRedisStorage(t)
    defer r.Close()
    
    event := models.Event{
        ID:        "test-1",
        Type:      "click",
        UserID:    "user-1",
        Value:     100.0,
        Timestamp: time.Now(),
    }
    
    r.AddEvent(event)
    
    // Проверяем, что событие было добавлено
    key := "events:user-1:click"
    count, err := r.client.LLen(r.ctx, key).Result()
    if err != nil {
        t.Fatalf("Failed to get list length: %v", err)
    }
    
    if count != 1 {
        t.Errorf("Expected 1 event in Redis, got %d", count)
    }
}

func TestRedisStorage_GetAggregated(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping Redis test in short mode")
    }
    
    r := setupRedisStorage(t)
    defer r.Close()
    
    now := time.Now()
    
    events := []models.Event{
        {ID: "1", Type: "click", UserID: "user-1", Value: 100, Timestamp: now},
        {ID: "2", Type: "click", UserID: "user-1", Value: 200, Timestamp: now.Add(1 * time.Minute)},
        {ID: "3", Type: "click", UserID: "user-1", Value: 150, Timestamp: now.Add(2 * time.Minute)},
    }
    
    for _, e := range events {
        r.AddEvent(e)
    }
    
    agg := r.GetAggregated("user-1", "click", time.Time{}, time.Time{})
    
    if agg == nil {
        t.Fatal("Expected aggregated data, got nil")
    }
    
    if agg.Count != 3 {
        t.Errorf("Expected count 3, got %d", agg.Count)
    }
    
    if agg.TotalValue != 450 {
        t.Errorf("Expected total 450, got %.2f", agg.TotalValue)
    }
    
    if agg.AvgValue != 150 {
        t.Errorf("Expected avg 150, got %.2f", agg.AvgValue)
    }
    
    if agg.MinValue != 100 {
        t.Errorf("Expected min 100, got %.2f", agg.MinValue)
    }
    
    if agg.MaxValue != 200 {
        t.Errorf("Expected max 200, got %.2f", agg.MaxValue)
    }
}

func TestRedisStorage_GetAggregated_WithFilters(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping Redis test in short mode")
    }
    
    r := setupRedisStorage(t)
    defer r.Close()
    
    now := time.Now()
    
    events := []models.Event{
        {ID: "1", Type: "click", UserID: "user-1", Value: 100, Timestamp: now},
        {ID: "2", Type: "view", UserID: "user-1", Value: 200, Timestamp: now},
        {ID: "3", Type: "click", UserID: "user-2", Value: 150, Timestamp: now},
    }
    
    for _, e := range events {
        r.AddEvent(e)
    }
    
    // Фильтр по user и type
    agg := r.GetAggregated("user-1", "click", time.Time{}, time.Time{})
    
    if agg == nil {
        t.Fatal("Expected aggregated data, got nil")
    }
    
    if agg.Count != 1 {
        t.Errorf("Expected count 1, got %d", agg.Count)
    }
    
    if agg.TotalValue != 100 {
        t.Errorf("Expected total 100, got %.2f", agg.TotalValue)
    }
}

func TestRedisStorage_GetAllAggregated(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping Redis test in short mode")
    }
    
    r := setupRedisStorage(t)
    defer r.Close()
    
    now := time.Now()
    
    events := []models.Event{
        {ID: "1", Type: "click", UserID: "user-1", Value: 100, Timestamp: now},
        {ID: "2", Type: "click", UserID: "user-1", Value: 200, Timestamp: now},
        {ID: "3", Type: "view", UserID: "user-1", Value: 50, Timestamp: now},
        {ID: "4", Type: "click", UserID: "user-2", Value: 300, Timestamp: now},
    }
    
    for _, e := range events {
        r.AddEvent(e)
    }
    
    results := r.GetAllAggregated()
    
    // Ожидаем 3 группы: user-1:click, user-1:view, user-2:click
    if len(results) != 3 {
        t.Errorf("Expected 3 aggregations, got %d", len(results))
    }
}

func TestRedisStorage_Concurrency(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping Redis test in short mode")
    }
    
    r := setupRedisStorage(t)
    defer r.Close()
    
    now := time.Now()
    
    // Запускаем 100 горутин, каждая добавляет 10 событий
    done := make(chan bool, 100)
    for i := 0; i < 100; i++ {
        go func(id int) {
            for j := 0; j < 10; j++ {
                r.AddEvent(models.Event{
                    ID:        string(rune(id*10 + j)),
                    Type:      "click",
                    UserID:    "user-1",
                    Value:     float64(j),
                    Timestamp: now,
                })
            }
            done <- true
        }(i)
    }
    
    // Ждем завершения всех горутин
    for i := 0; i < 100; i++ {
        <-done
    }
    
    // Проверяем что все 1000 событий добавлены
    agg := r.GetAggregated("user-1", "click", time.Time{}, time.Time{})
    if agg == nil || agg.Count != 1000 {
        t.Errorf("Expected 1000 events, got %d", agg.Count)
    }
}
