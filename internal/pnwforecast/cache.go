package pnwforecast

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type cacheItem struct {
	CachedAt string          `json:"cached_at"`
	Value    json.RawMessage `json:"value"`
}

type JSONCache struct {
	Path       string
	TTLSeconds int
}

func NewJSONCache(path string, ttlSeconds int) (*JSONCache, error) {
	c := &JSONCache{Path: path, TTLSeconds: ttlSeconds}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c *JSONCache) load() (map[string]cacheItem, error) {
	payload := map[string]cacheItem{}
	raw, err := os.ReadFile(c.Path)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return payload, nil
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (c *JSONCache) save(v map[string]cacheItem) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.Path, raw, 0o644)
}

func (c *JSONCache) MakeKey(provider, endpoint string, params any) (string, error) {
	body, err := json.Marshal(map[string]any{
		"provider": provider,
		"endpoint": endpoint,
		"params":   params,
	})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

func (c *JSONCache) Get(key string) (json.RawMessage, int, bool, error) {
	payload, err := c.load()
	if err != nil {
		return nil, 0, false, err
	}
	item, ok := payload[key]
	if !ok {
		return nil, 0, false, nil
	}
	cachedAt, err := time.Parse(time.RFC3339, item.CachedAt)
	if err != nil {
		return nil, 0, false, nil
	}
	age := int(time.Since(cachedAt).Seconds())
	if age > c.TTLSeconds {
		return nil, age, false, nil
	}
	return item.Value, age, true, nil
}

func (c *JSONCache) Put(key string, value any) error {
	payload, err := c.load()
	if err != nil {
		return err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	payload[key] = cacheItem{
		CachedAt: time.Now().UTC().Format(time.RFC3339),
		Value:    raw,
	}
	return c.save(payload)
}
