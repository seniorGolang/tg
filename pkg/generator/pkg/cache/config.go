package cache

type RedisConfig struct {
	Address  string `envconfig:"REDIS_ADDRESS" default:"127.0.0.1:6379"`
	Username string `envconfig:"REDIS_USER"`
	Password string `envconfig:"REDIS_PASSWORD"`
	Database int    `envconfig:"REDIS_DATABASE" default:"0"`
	PoolSize int    `envconfig:"REDIS_POOL_SIZE" default:"10"`
}
