package redis

import (
  "context"
  "fmt"
  "github.com/redis/go-redis/v9"
  "strings"
)

type Client struct {
  client *redis.Client
  Config *ServerConfig
  ctx    context.Context
}

func (r *Client) Connect() error {
  r.ctx = context.Background()
  addr := fmt.Sprintf("%s:%d", r.Config.Server, r.Config.Port)

  client := redis.NewClient(&redis.Options{
    Addr:     addr,
    Password: r.Config.Password,
    DB:       r.Config.DB,
  })

  r.client = client
  return nil
}

func (r *Client) Disconnect() {
  err := r.client.Close()
  if err != nil {
    return
  }
}

func (r *Client) Get(key string) (string, error) {
  value, err := r.client.Get(r.ctx, key).Result()
  if err != nil {
    return "", err
  }
  return value, nil
}

func (r *Client) Set(key string, value string) error {
  _, err := r.client.Set(r.ctx, key, value, 0).Result()
  if err != nil {
    return err
  }
  return nil
}

func (r *Client) Delete(key string) error {
  _, err := r.client.Del(r.ctx, key).Result()
  if err != nil {
    return err
  }
  return nil
}

func (r *Client) Info() (*Info, error) {
  info, err := r.client.Info(r.ctx).Result()
  if err != nil {
    return nil, err
  }

  var redisInfo Info
  kvs := parseInfo(info)
  redisInfo.BuildDate = kvs["build_date"]
  redisInfo.UsedMemory = kvs["used_memory"]
  redisInfo.UsedMemoryHuman = kvs["used_memory_human"]
  redisInfo.MaxMemory = kvs["maxmemory"]
  redisInfo.MaxMemoryHuman = kvs["maxmemory_human"]
  redisInfo.TotalMemory = kvs["total_system_memory"]
  redisInfo.TotalMemoryHuman = kvs["total_system_memory_human"]
  redisInfo.Role = kvs["role"]
  redisInfo.Db0 = kvs["db0"]
  return &redisInfo, nil
}

func parseInfo(info string) map[string]string {
  result := make(map[string]string)
  lines := strings.Split(info, "\n")
  for _, line := range lines {
    if strings.HasPrefix(line, "#") || !strings.Contains(info, ":") {
      continue
    }

    params := strings.Split(info, ":")
    if len(params) != 2 {
      continue
    }
    key := strings.TrimSpace(params[0])
    value := strings.TrimSpace(params[1])
    result[key] = value
  }
  return result
}
