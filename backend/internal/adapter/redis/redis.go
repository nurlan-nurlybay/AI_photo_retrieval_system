package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
}

func NewClient(ctx context.Context, cfg *config.Config) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: 50,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &Client{rdb: rdb}, nil
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

func (c *Client) Set(ctx context.Context, key string, value string, ttlSeconds int) error {
	return c.rdb.Set(ctx, key, value, time.Duration(ttlSeconds)*time.Second).Err()
}

func (c *Client) Delete(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, key).Err()
}

func (c *Client) Enqueue(ctx context.Context, key string, payload []byte) error {
	return c.rdb.RPush(ctx, key, payload).Err()
}

func (c *Client) Publish(ctx context.Context, channel string, message interface{}) error {
	return c.rdb.Publish(ctx, channel, message).Err()
}

func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.rdb.Subscribe(ctx, channels...)
}

// func (c *Client) DequeueBlock(ctx context.Context, key string, timeoutSeconds int) (string, []byte, error) {
// 	// BRPOP returns list [key, value]
// 	res, err := c.rdb.BRPop(ctx, 0*time.Second, key).Result()
// 	if err != nil {
// 		return "", nil, err
// 	}
// 	// res[0] is key, res[1] is value
// 	return res[0], []byte(res[1]), nil
// }

// TryDequeue is a non-blocking pop. It checks each key left-to-right and returns
// the first item found. Returns ("", nil, nil) when all queues are empty.
func (c *Client) TryDequeue(ctx context.Context, keys ...string) (string, []byte, error) {
	for _, key := range keys {
		val, err := c.rdb.RPop(ctx, key).Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return "", nil, fmt.Errorf("redis RPop failed: %w", err)
		}
		return key, []byte(val), nil
	}
	return "", nil, nil
}

func (c *Client) DequeueBlock(ctx context.Context, timeoutSeconds int, keys ...string) (string, []byte, error) {
	if len(keys) == 0 {
		return "", nil, fmt.Errorf("no keys provided to DequeueBlock")
	}
	
	for {
		// BRPop with a real timeout, accepts multiple keys. 
		// Redis checks the keys in the order given, natively prioritizing the first key!
		res, err := c.rdb.BRPop(ctx, time.Duration(timeoutSeconds)*time.Second, keys...).Result()
		if err != nil {
			if err == redis.Nil {
				// no job yet, just loop and retry
				continue
			}
			return "", nil, fmt.Errorf("redis BRPop failed: %w", err)
		}

		if len(res) != 2 {
			// invalid response from Redis, just continue
			continue
		}

		// res[0] is the queue key that had the element, res[1] is the value
		return res[0], []byte(res[1]), nil
	}
}
