package main

import (
  "fmt"
  "time"
)

type Message struct {
  name string
}

func main() {
  ch := make(chan *Message)

  go func() {
    msg := <-ch
    fmt.Printf("v1 %s \n", msg.name)

    msg.name = "world"
  }()

  msg := Message{
    name: "hello",
  }

  ch <- &msg

  for {
    fmt.Printf("v2 %s \n", msg.name)
    time.Sleep(1 * time.Second)
  }
}
