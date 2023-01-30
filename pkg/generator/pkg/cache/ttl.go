package cache

import (
	"context"
	"encoding/json"
	"time"
)

const (
	NeverExpire = time.Duration(0)
)

type cacheItemIn struct {
	E time.Time       `json:"e"`
	P json.RawMessage `json:"p"`
}

type cacheItemOut struct {
	E time.Time   `json:"e"`
	P interface{} `json:"p"`
}

func (rc *Cache) SetTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) (err error) {

	var bytes []byte
	if bytes, err = json.Marshal(cacheItemOut{E: time.Now(), P: value}); err != nil {
		return
	}
	return rc.rdb.Set(ctx, key, string(bytes), ttl).Err()
}

func (rc *Cache) GetTTL(ctx context.Context, key string, value interface{}) (createdAt time.Time, ttl time.Duration, err error) {

	cacheItem := rc.rdb.Get(ctx, key)
	if err = cacheItem.Err(); err != nil {
		return
	}
	if durationRet := rc.rdb.PTTL(ctx, key); durationRet.Err() == nil {
		if ttl, err = durationRet.Result(); err != nil {
			return
		}
	}
	var item cacheItemIn
	bytes, _ := cacheItem.Bytes()
	if err = json.Unmarshal(bytes, &item); err != nil {
		return
	}
	createdAt = item.E
	err = json.Unmarshal(item.P, &value)
	return
}
