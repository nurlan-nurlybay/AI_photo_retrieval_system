package redis

// type WorkerClient struct {
// 	rdb *redis.Client
// }

// func NewWorkerClient(cfg *config.Config) *WorkerClient {
// 	rdb := redis.NewClient(&redis.Options{
// 		Addr:     cfg.Redis.Addr,
// 		Password: cfg.Redis.Password,
// 		DB:       cfg.Redis.DB,
// 	})
// 	return &WorkerClient{rdb: rdb}
// }
// func (c *WorkerClient) Enqueue(ctx context.Context, key string, payload []byte) error {
// 	return c.rdb.RPush(ctx, key, payload).Err()
// }

// func (c *WorkerClient) DequeueBlock(ctx context.Context, key string, timeoutSeconds int) (string, []byte, error) {
// 	// BRPOP returns list,key and value; we just care about value
// 	res, err := c.rdb.BRPop(ctx, time.Duration(timeoutSeconds)*time.Second, key).Result()
// 	if err != nil {
// 		return "", nil, err
// 	}
// 	// res[0] is key, res[1] is value
// 	return res[0], []byte(res[1]), nil
// }
