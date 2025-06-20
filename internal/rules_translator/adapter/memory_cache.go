package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"firestore-clone/internal/rules_translator/domain"
)

// MemoryCache implementa RulesCache optimizado en memoria para máxima velocidad
type MemoryCache struct {
	cache    map[string]*CacheEntry
	mutex    sync.RWMutex
	stats    *domain.CacheStats
	config   *CacheConfig
	janitor  *time.Timer
	stopChan chan struct{}
}

type CacheEntry struct {
	Data        *domain.TranslationResult `json:"data"`
	ExpiresAt   time.Time                 `json:"expires_at"`
	AccessCount int64                     `json:"access_count"`
	LastAccess  time.Time                 `json:"last_access"`
	Size        int64                     `json:"size"`
}

type CacheConfig struct {
	MaxSize         int           `json:"max_size"`
	DefaultTTL      time.Duration `json:"default_ttl"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
	MaxMemoryMB     int64         `json:"max_memory_mb"`
	EnableMetrics   bool          `json:"enable_metrics"`
}

// NewMemoryCache crea una nueva instancia de caché en memoria optimizada
func NewMemoryCache(config *CacheConfig) *MemoryCache {
	if config == nil {
		config = DefaultCacheConfig()
	}

	cache := &MemoryCache{
		cache:    make(map[string]*CacheEntry, config.MaxSize),
		config:   config,
		stopChan: make(chan struct{}),
		stats: &domain.CacheStats{
			TotalRequests: 0,
			CacheSize:     0,
			MemoryUsage:   0,
		},
	}

	// Iniciar limpieza automática
	cache.startJanitor()

	return cache
}

// DefaultCacheConfig configuración optimizada por defecto
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		MaxSize:         1000,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Minute * 5,
		MaxMemoryMB:     100,
		EnableMetrics:   true,
	}
}

// Get obtiene reglas desde caché con métricas optimizadas
func (c *MemoryCache) Get(ctx context.Context, key *domain.CacheKey) (*domain.TranslationResult, error) {
	startTime := time.Now()
	defer func() {
		if c.config.EnableMetrics {
			c.updateLatencyMetrics(time.Since(startTime))
		}
	}()

	cacheKey := c.generateKey(key)

	c.mutex.RLock()
	entry, exists := c.cache[cacheKey]
	c.mutex.RUnlock()

	c.updateRequestMetrics()

	if !exists {
		c.updateMissMetrics()
		return nil, fmt.Errorf("cache miss")
	}

	// Verificar expiración
	if time.Now().After(entry.ExpiresAt) {
		c.mutex.Lock()
		delete(c.cache, cacheKey)
		c.mutex.Unlock()
		c.updateMissMetrics()
		return nil, fmt.Errorf("cache expired")
	}

	// Actualizar estadísticas de acceso
	c.mutex.Lock()
	entry.AccessCount++
	entry.LastAccess = time.Now()
	c.mutex.Unlock()

	c.updateHitMetrics()
	return entry.Data, nil
}

// Set guarda reglas en caché con TTL configurable y optimización de memoria
func (c *MemoryCache) Set(ctx context.Context, key *domain.CacheKey, result *domain.TranslationResult, ttl int) error {
	cacheKey := c.generateKey(key)

	// Calcular tamaño del objeto para control de memoria
	dataSize := c.calculateSize(result)

	entry := &CacheEntry{
		Data:        result,
		ExpiresAt:   time.Now().Add(time.Duration(ttl) * time.Second),
		AccessCount: 0,
		LastAccess:  time.Now(),
		Size:        dataSize,
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Verificar límites de memoria y tamaño
	if len(c.cache) >= c.config.MaxSize {
		c.evictLRU()
	}

	c.cache[cacheKey] = entry
	c.updateCacheSize()

	return nil
}

// Invalidate invalida reglas específicas
func (c *MemoryCache) Invalidate(ctx context.Context, key *domain.CacheKey) error {
	cacheKey := c.generateKey(key)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, exists := c.cache[cacheKey]; exists {
		delete(c.cache, cacheKey)
		c.updateCacheSize()
		return nil
	}

	return fmt.Errorf("key not found")
}

// InvalidateAll limpia toda la caché
func (c *MemoryCache) InvalidateAll(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache = make(map[string]*CacheEntry, c.config.MaxSize)
	c.updateCacheSize()

	return nil
}

// GetStats retorna estadísticas de caché
func (c *MemoryCache) GetStats() *domain.CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Crear copia para evitar race conditions
	stats := *c.stats
	stats.CacheSize = int64(len(c.cache))

	return &stats
}

// Preload precarga reglas frecuentemente usadas
func (c *MemoryCache) Preload(ctx context.Context, keys []*domain.CacheKey) error {
	// Para caché en memoria, no hay preload específico
	// Este método se implementaría en cachés distribuidos como Redis
	return nil
}

// Helper methods para optimización

func (c *MemoryCache) generateKey(key *domain.CacheKey) string {
	// Generar clave rápida y única
	return fmt.Sprintf("%s:%s:%s:%s", key.ProjectID, key.DatabaseID, key.Version, key.Hash)
}

func (c *MemoryCache) calculateSize(result *domain.TranslationResult) int64 {
	// Estimación rápida del tamaño en bytes
	data, _ := json.Marshal(result)
	return int64(len(data))
}

func (c *MemoryCache) evictLRU() {
	// Evicción LRU (Least Recently Used)
	var oldestKey string
	var oldestTime time.Time = time.Now()

	for key, entry := range c.cache {
		if entry.LastAccess.Before(oldestTime) {
			oldestTime = entry.LastAccess
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
		if c.config.EnableMetrics {
			c.stats.EvictionCount++
		}
	}
}

func (c *MemoryCache) startJanitor() {
	c.janitor = time.NewTimer(c.config.CleanupInterval)

	go func() {
		for {
			select {
			case <-c.janitor.C:
				c.cleanup()
				c.janitor.Reset(c.config.CleanupInterval)
			case <-c.stopChan:
				c.janitor.Stop()
				return
			}
		}
	}()
}

func (c *MemoryCache) cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for key, entry := range c.cache {
		if now.After(entry.ExpiresAt) {
			delete(c.cache, key)
		}
	}

	c.updateCacheSize()
}

func (c *MemoryCache) updateRequestMetrics() {
	if !c.config.EnableMetrics {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.stats.TotalRequests++
}

func (c *MemoryCache) updateHitMetrics() {
	if !c.config.EnableMetrics {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	hitRate := 1.0 / float64(c.stats.TotalRequests)
	if c.stats.HitRate == 0 {
		c.stats.HitRate = hitRate
	} else {
		c.stats.HitRate = c.stats.HitRate*0.9 + hitRate*0.1
	}

	c.stats.MissRate = 1.0 - c.stats.HitRate
}

func (c *MemoryCache) updateMissMetrics() {
	if !c.config.EnableMetrics {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	missRate := 1.0 / float64(c.stats.TotalRequests)
	if c.stats.MissRate == 0 {
		c.stats.MissRate = missRate
	} else {
		c.stats.MissRate = c.stats.MissRate*0.9 + missRate*0.1
	}

	c.stats.HitRate = 1.0 - c.stats.MissRate
}

func (c *MemoryCache) updateLatencyMetrics(latency time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.stats.AverageLatency == 0 {
		c.stats.AverageLatency = latency
	} else {
		c.stats.AverageLatency = time.Duration(
			float64(c.stats.AverageLatency)*0.9 + float64(latency)*0.1,
		)
	}

	c.stats.LastAccess = time.Now()
}

func (c *MemoryCache) updateCacheSize() {
	var totalSize int64
	for _, entry := range c.cache {
		totalSize += entry.Size
	}
	c.stats.MemoryUsage = totalSize
}

// Close cierra el caché y limpia recursos
func (c *MemoryCache) Close() {
	close(c.stopChan)
	if c.janitor != nil {
		c.janitor.Stop()
	}
}
