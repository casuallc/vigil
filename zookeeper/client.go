package zookeeper

import (
  "errors"
  "fmt"
  "github.com/samuel/go-zookeeper/zk"
  "time"
)

type Client struct {
  Config *ServerConfig
  Client *zk.Conn
}

func (r *Client) Connect() error {
  if r.Config.Timeout == 0 {
    r.Config.Timeout = 30
  }

  zkAddrs := []string{fmt.Sprintf("%s:%d", r.Config.Server, r.Config.Port)}
  conn, _, err := zk.Connect(zkAddrs, time.Duration(r.Config.Timeout)*time.Second)
  if err != nil {
    return err
  }
  r.Client = conn
  defer r.Client.Close()

  fmt.Printf("✅ Connect %s:%d \n", r.Config.Server, r.Config.Port)
  return nil
}

func (r *Client) Disconnect() {
  r.Client.Close()
}

func (r *Client) Create(path string, data []byte) error {
  _, err := r.Client.Create(path, data, 0, zk.WorldACL(zk.PermAll))
  if err != nil {
    if !errors.Is(err, zk.ErrNodeExists) {
      return err
    }
  }

  return nil
}

func (r *Client) Delete(path string) error {
  err := r.Client.Delete(path, -1)
  if err != nil {
    if !errors.Is(err, zk.ErrNoNode) {
      return err
    }
  }

  fmt.Printf("✅ Delete node: %s\n", path)
  return nil
}

func (r *Client) Exists(path string) (bool, error) {
  exists, _, err := r.Client.Exists(path)
  if err != nil {
    return exists, err
  }

  return exists, nil
}

func (r *Client) Get(path string) ([]byte, error) {
  value, _, err := r.Client.Get(path)
  if err != nil {
    return nil, err
  }

  return value, nil
}

func (r *Client) Set(path string, data []byte) error {
  _, err := r.Client.Set(path, data, -1)
  if err != nil {
    return err
  }

  return nil
}
