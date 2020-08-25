package lru

import (
	lru "github.com/hashicorp/golang-lru"
	"github.com/hashicorp/golang-lru/simplelru"
	log "github.com/sirupsen/logrus"
)

type CacheItem struct {
	ImageID   string
	ImageName string
}

type Cache struct {
	lru            simplelru.LRUCache
	onAdd, onEvict func(imageName string, imageID string)
}

func NewCache(cacheSize int) (*Cache, error) {
	cache := &Cache{}
	lruCache, err := lru.NewWithEvict(cacheSize, cache.lruEvict())
	if err != nil {
		return nil, err
	}
	cache.lru = lruCache
	cache.onAdd = func(imageName string, imageID string) { log.Warn("cache onAdd handler is not defined") }
	cache.onEvict = func(imageName string, imageID string) { log.Warn("cache onEvict handler is not defined") }

	log.Infof("LRU eviction, max-size: %d", cacheSize)

	return cache, nil
}

func (c *Cache) AddSilent(imageName, imageID string) {
	c.lru.Add(imageName, CacheItem{ImageName: imageName, ImageID: imageID})
}

func (c *Cache) Add(imageName, imageID string) {
	isNew := !c.lru.Contains(imageName)
	c.lru.Add(imageName, CacheItem{ImageName: imageName, ImageID: imageID})
	if isNew {
		c.onAdd(imageName, imageID)
	}
}

func (c *Cache) OnAdd(f func(imageName string, imageID string)) {
	c.onAdd = f
}

func (c *Cache) OnEvict(f func(imageName string, imageID string)) {
	c.onEvict = f
}

func (c *Cache) lruEvict() func(key interface{}, value interface{}) {
	return func(key, value interface{}) {
		item := value.(CacheItem)
		c.onEvict(item.ImageName, item.ImageID)
	}
}
