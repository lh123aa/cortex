# Go Concurrency Guide

Go's concurrency model is based on goroutines and channels.
Goroutines are lightweight threads managed by the Go runtime.
Channels allow goroutines to communicate with each other.

## Goroutines

Use `go` keyword to start a goroutine:
```go
go func() {
    fmt.Println("Hello from goroutine")
}()
```

## Channels

Channels are typed conduits for communication:
```go
ch := make(chan int)
go func() { ch <- 42 }()
value := <-ch
```
