package redis

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
)

func NewClient(addr, password string, db int) (*goredis.Client, error) {
	client := goredis.NewClient(&goredis.Options{Addr: addr, Password: password, DB: db})
	if err := client.Ping(context.Background()).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}
