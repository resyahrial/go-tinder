package infra

import (
	"log"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
)

var (
	RedisPool     *redis.Pool
	redisPoolOnce sync.Once
)

func NewRedisPool(server, password string, db int) {
	redisPoolOnce.Do(func() {
		RedisPool = &redis.Pool{
			MaxIdle:     3,
			Wait:        true,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				c, err := redis.Dial("tcp", server)
				if err != nil {
					return nil, err
				}
				if password != "" {
					if _, err := c.Do("AUTH", password); err != nil {
						c.Close()
						return nil, err
					}
				}

				if _, err := c.Do("SELECT", db); err != nil {
					c.Close()
					return nil, err
				}
				return c, err
			},
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				_, err := c.Do("PING")
				return err
			},
		}

		c, err := RedisPool.Dial()
		if err != nil {
			panic(err)
		}
		defer c.Close()

		if pong, err := redis.String(c.Do("PING")); err != nil {
			panic("Cannot ping Redis")
		} else {
			log.Printf("Redis Ping: %s\n", pong)
		}
	})
}

func TerminalRedisPool() {
	log.Println("closing cache connection")
	if err := RedisPool.Close(); err != nil {
		log.Printf("fail closing cache connection: %s\n", err)
	} else {
		log.Println("cache connection closed")
	}
}
