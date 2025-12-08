/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package redis

import (
  "context"
  "fmt"
  "github.com/redis/go-redis/v9"
  "strings"
  "time"
)

type Client struct {
  client         *redis.Client
  Config         *ServerConfig
  ctx            context.Context
  producedCount  int64 // AI Modified: 记录生产的消息总数
  consumedCount  int64 // AI Modified: 记录消费的消息总数
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
  // AI Modified: 打印消息计数
  fmt.Printf("Redis Client Stats - Produced: %d, Consumed: %d\n", r.producedCount, r.consumedCount)
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

// Publish 发布消息到指定频道
func (r *Client) Publish(channel string, message string) error {
  _, err := r.client.Publish(r.ctx, channel, message).Result()
  if err != nil {
    return err
  }
  r.producedCount++
  return nil
}

// Subscribe 订阅指定频道的消息
func (r *Client) Subscribe(channel string, handler func(string, string) bool, timeout int) error {
  pubsub := r.client.Subscribe(r.ctx, channel)
  defer pubsub.Close()

  // 设置超时
  var timer <-chan time.Time
  if timeout > 0 {
    timer = time.NewTimer(time.Duration(timeout) * time.Second).C
  }

  for {
    select {
    case msg := <-pubsub.Channel():
      r.consumedCount++
      if !handler(msg.Channel, msg.Payload) {
        return nil
      }
    case <-timer:
      return nil
    case <-r.ctx.Done():
      return nil
    }
  }
}

func parseInfo(info string) map[string]string {
  result := make(map[string]string)
  lines := strings.Split(info, "\n")
  for _, line := range lines {
    if strings.HasPrefix(line, "#") || !strings.Contains(line, ":") {
      continue
    }

    params := strings.Split(line, ":")
    if len(params) != 2 {
      continue
    }
    key := strings.TrimSpace(params[0])
    value := strings.TrimSpace(params[1])
    result[key] = value
  }
  return result
}
