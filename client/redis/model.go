package redis

type ServerConfig struct {
  Server   string
  Port     int
  Password string
  DB       int
}

type Info struct {
  BuildDate        string
  UsedMemory       string
  UsedMemoryHuman  string
  MaxMemory        string
  MaxMemoryHuman   string
  TotalMemory      string
  TotalMemoryHuman string
  Role             string
  Db0              string
}
