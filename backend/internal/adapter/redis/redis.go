package redis

import (
	"context"
	"time"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
}

func NewClient(cfg *config.Config) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	return &Client{rdb: rdb}
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

func (c *Client) DequeueBlock(ctx context.Context, key string, timeoutSeconds int) (string, []byte, error) {
	// BRPOP returns list,key and value; we just care about value
	res, err := c.rdb.BRPop(ctx, time.Duration(timeoutSeconds)*time.Second, key).Result()
	if err != nil {
		return "", nil, err
	}
	// res[0] is key, res[1] is value
	return res[0], []byte(res[1]), nil
}
