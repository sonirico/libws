# libws

A flexible and extensible WebSocket client library for Go with focus on connection resilience and customizability.

![libws - WebSocket Library for Go](https://github.com/sonirico/libws/blob/master/art.png)

## Motivation

This library was created after many years of handling complex WebSocket behaviors across a wide range of providers and integrations. Each provider often presented unique challenges:

- Different reconnection strategies
- Various keep-alive mechanisms
- Inconsistent ping/pong implementations
- Varying timeout behaviors
- Custom authentication flows
- Rate limiting concerns

Rather than creating a new client implementation for each provider, `libws` unifies the configuration approach through composition. By breaking down WebSocket client functionality into composable components, you can:

1. **Mix and match** behaviors to accommodate specific provider requirements
2. **Extend the library** with custom components for specific use cases
3. **Reuse common patterns** across different integrations
4. **Test individual components** in isolation

This compositional approach means you can plug in or swap out components as needed, constructing a WebSocket client with exactly the behaviors required for your specific provider - all without modifying the core library.

## Features

- **Connection Resilience**: Automatic reconnection with configurable retry strategies
- **Flexible Logging**: Pluggable logging interface
- **Event-Driven Architecture**: Subscribe to connection events (connect, reconnect, close)
- **Keep-Alive Mechanisms**: Both active and passive keep-alive strategies
- **Extensible Design**: Compose connection handlers to build complex behaviors
- **Message Handling**: Structured message types and processing

## Installation

```go
go get github.com/sonirico/libws
```

## Usage Examples

### Basic WebSocket Client

```go
package main

import (
    "context"
    "log"
    "net/url"
    "os"
    "time"

    "github.com/fasthttp/websocket"
    "github.com/sonirico/libws"
)

func main() {
    // Create logger
    logger := libws.NewTestLogger(os.Stdout)
    
    // Set up connection parameters
    wsURL, _ := url.Parse("wss://example.com/socket")
    params := libws.OpenConnectionParams{
        URL: *wsURL,
    }
    
    // Create a params repo that will provide connection parameters
    paramsGetter := func(ctx context.Context) (libws.OpenConnectionParams, error) {
        return params, nil
    }
    paramsRepo := libws.NewOpenConnectionParamsRepo(logger, paramsGetter)
    
    // Create WebSocket dialer
    dialer := websocket.DefaultDialer
    
    // Define message handler
    messageHandler := func(client libws.Client, msg libws.Message) {
        log.Printf("Received message: %s", msg.Data())
        // Process message
    }
    
    // Define event handler
    eventHandler := func(client libws.Client, event libws.EventType) {
        switch event {
        case libws.EventConnect:
            log.Println("Connected")
        case libws.EventReconnect:
            log.Println("Reconnected")
        case libws.EventClose:
            log.Println("Closed")
        }
    }
    
    // Create a connection factory with passive keep-alive support
    connFactory := libws.NewWebsocketFactory(
        logger,
        dialer,
        paramsRepo,
        libws.ErrorAdapters{},
    )
    
    // Create a connection handler with backoff reconnection strategy
    backoffConnFactory := libws.NewBackoffConnectionHandlerFactory(
        logger,
        connFactory,
        libws.ExponentialBackoffSeconds,
        30*time.Second,
    )
    
    // Wrap with passive keep-alive handler
    keepAliveConnFactory := libws.NewPassiveKeepAliveConnectionHandlerFactory(
        backoffConnFactory,
        libws.KeepAliveHandlerReplyPingWithPong,
    )
    
    // Create the client
    clientFactory := libws.NewBasicClientFactory(
        keepAliveConnFactory,
        messageHandler,
        eventHandler,
    )
    
    client := clientFactory()
    
    // Connect
    ctx := context.Background()
    if err := client.Open(ctx); err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    
    // Send a message
    client.Send(libws.NewDataMessage([]byte("Hello, WebSocket server!")))
    
    // Wait for connection to close
    <-client.CloseChan()
}
```

### Using Active Keep-Alive

```go
// Create a connection factory with active keep-alive
activeKeepAliveFactory := libws.NewActiveKeepAliveConnectionHandlerFactory(
    logger,
    backoffConnFactory,
    30*time.Second,
    libws.NewKeepAliveMessageFactory(libws.PingMessage, func() []byte {
        return []byte("ping")
    }),
)

// Create client with active keep-alive
client := libws.NewBasicClientFactory(
    activeKeepAliveFactory,
    messageHandler,
    eventHandler,
)()
```

### Using Periodic Reconnection

```go
// Create a connection factory that reconnects every 5 minutes
reconnectFactory := libws.NewReopenIntervalConnFactory(
    logger,
    5*time.Minute,
    connFactory,
)

// Create client with periodic reconnection
client := libws.NewBasicClientFactory(
    reconnectFactory,
    messageHandler,
    eventHandler,
)()
```

## Architecture

The library is composed of several layers:

1. **Client**: The high-level API that applications interact with
2. **Connection Handlers**: Middleware components that add behavior to connections
3. **Connection**: Low-level component that manages the actual WebSocket connection
4. **Messages**: Structured data types for communication
5. **Events**: Notification system for connection state changes

## Connection Handler Types

- **Basic Connection**: Simple pass-through connection handler
- **Backoff Connection**: Reconnects with exponential backoff on failure
- **Reopen Interval Connection**: Periodically creates a new connection
- **Active Keep-Alive**: Sends periodic ping messages
- **Passive Keep-Alive**: Responds to ping messages with pongs

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
