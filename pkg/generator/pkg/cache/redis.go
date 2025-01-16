package cache

import (
	"context"
	"strings"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
)

var Nil = redis.Nil

type Cache struct {
	single  *redis.Client
	cluster *redis.ClusterClient

	rdb  API
	cfg  RedisConfig
	sync *redsync.Redsync
}

func New(cfg RedisConfig) (cache *Cache) {

	cache = &Cache{cfg: cfg}
	servers := strings.Split(cfg.Address, ",")
	if len(servers) == 1 {
		cache.single = redis.NewClient(&redis.Options{
			Addr:     servers[0],
			DB:       cfg.Database,
			PoolSize: cfg.PoolSize,
			Username: cfg.Username,
			Password: cfg.Password,
		})
		cache.rdb = cache.single
		cache.sync = redsync.New(goredis.NewPool(cache.single))
	} else if len(servers) > 1 {
		cache.cluster = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    servers,
			PoolSize: cfg.PoolSize,
			Username: cfg.Username,
			Password: cfg.Password,
		})
		cache.rdb = cache.cluster
		cache.sync = redsync.New(goredis.NewPool(cache.cluster))
	}
	return
}

func (rc *Cache) R() API {
	return rc.rdb
}

type API interface {
	PTTL(ctx context.Context, key string) *redis.DurationCmd

	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
}
