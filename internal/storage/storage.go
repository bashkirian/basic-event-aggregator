package storage

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "sync"
    "time"

    "github.com/bashkirian/event-aggregator/pkg/models"
    "github.com/go-redis/redis/v8"
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

// RedisStorage реализация хранилища на основе Redis
type RedisStorage struct {
    client *redis.Client
    ctx    context.Context
}

// NewRedisStorage создает новое Redis хранилище
func NewRedisStorage(addr, password string, db int) *RedisStorage {
    rdb := redis.NewClient(&redis.Options{
        Addr:     addr,
        Password: password,
        DB:       db,
    })

    return &RedisStorage{
        client: rdb,
        ctx:    context.Background(),
    }
}

// AddEvent добавляет событие в Redis
func (r *RedisStorage) AddEvent(event models.Event) {
    eventJSON, err := json.Marshal(event)
    if err != nil {
        return
    }

    // Используем ключ в формате events:{userID}:{eventType}
    key := fmt.Sprintf("events:%s:%s", event.UserID, event.Type)
    
    // Добавляем событие в список (LPUSH для добавления в начало)
    r.client.LPush(r.ctx, key, eventJSON)
    
    // Устанавливаем TTL на 24 часа для предотвращения переполнения
    r.client.Expire(r.ctx, key, 24*time.Hour)
}

// GetAggregated получает агрегированные данные из Redis
func (r *RedisStorage) GetAggregated(userID, eventType string, from, to time.Time) *models.AggregatedData {
    var keys []string
    
    if userID != "" && eventType != "" {
        // Конкретный пользователь и тип события
        keys = []string{fmt.Sprintf("events:%s:%s", userID, eventType)}
    } else if userID != "" {
        // Все типы событий для пользователя
        pattern := fmt.Sprintf("events:%s:*", userID)
        keys = r.getKeysByPattern(pattern)
    } else if eventType != "" {
        // Все пользователи для конкретного типа события
        pattern := fmt.Sprintf("events:*:%s", eventType)
        keys = r.getKeysByPattern(pattern)
    } else {
        // Все события
        pattern := "events:*"
        keys = r.getKeysByPattern(pattern)
    }

    var allEvents []models.Event
    
    for _, key := range keys {
        events, err := r.getEventsFromKey(key, from, to)
        if err != nil {
            continue
        }
        allEvents = append(allEvents, events...)
    }

    if len(allEvents) == 0 {
        return nil
    }

    // Фильтруем по userID и eventType если нужно
    var filtered []models.Event
    for _, e := range allEvents {
        if (userID == "" || e.UserID == userID) &&
            (eventType == "" || e.Type == eventType) {
            filtered = append(filtered, e)
        }
    }

    if len(filtered) == 0 {
        return nil
    }

    return aggregate(filtered, userID, eventType)
}

// GetAllAggregated получает все агрегированные данные из Redis
func (r *RedisStorage) GetAllAggregated() []models.AggregatedData {
    pattern := "events:*"
    keys := r.getKeysByPattern(pattern)
    
    result := make([]models.AggregatedData, 0)
    
    for _, key := range keys {
        events, err := r.getEventsFromKey(key, time.Time{}, time.Time{})
        if err != nil || len(events) == 0 {
            continue
        }
        
        // Извлекаем userID и eventType из ключа
        parts := strings.Split(key, ":")
        if len(parts) != 3 {
            continue
        }
        
        userID := parts[1]
        eventType := parts[2]
        
        agg := aggregate(events, userID, eventType)
        if agg != nil {
            result = append(result, *agg)
        }
    }
    
    return result
}

// getKeysByPattern получает ключи по шаблону
func (r *RedisStorage) getKeysByPattern(pattern string) []string {
    iter := r.client.Scan(r.ctx, 0, pattern, 0).Iterator()
    var keys []string
    
    for iter.Next(r.ctx) {
        keys = append(keys, iter.Val())
    }
    
    return keys
}

// getEventsFromKey получает события из ключа с фильтрацией по времени
func (r *RedisStorage) getEventsFromKey(key string, from, to time.Time) ([]models.Event, error) {
    // Получаем все события из списка
    eventStrs, err := r.client.LRange(r.ctx, key, 0, -1).Result()
    if err != nil {
        return nil, err
    }
    
    var events []models.Event
    
    for _, eventStr := range eventStrs {
        var event models.Event
        if err := json.Unmarshal([]byte(eventStr), &event); err != nil {
            continue
        }
        
        // Фильтруем по времени
        if (!from.IsZero() && event.Timestamp.Before(from)) ||
            (!to.IsZero() && event.Timestamp.After(to)) {
            continue
        }
        
        events = append(events, event)
    }
    
    return events, nil
}

// Close закрывает соединение с Redis
func (r *RedisStorage) Close() error {
    return r.client.Close()
}
