package messaging

import (
    "context"
    "fmt"
    "os"
    "sort"
    "testing"
    "time"
)

// Simple ping-pong microbenchmark using the existing bus implementation.
func TestPingPongBenchmark(t *testing.T) {
    if os.Getenv("AF_RUN_PINGPONG") != "true" {
        t.Skip("Run manually - set AF_RUN_PINGPONG=true to execute")
    }

    url := os.Getenv("AF_BUS_URL")
    if url == "" {
        url = "nats://127.0.0.1:4222"
    }

    iterations := 200
    ctx := context.Background()
    lat, err := RunPingPong(ctx, url, iterations)
    if err != nil {
        t.Fatalf("pingpong failed: %v", err)
    }

    // report p50/p95/p99
    p50 := lat[len(lat)/2]
    p95 := lat[int(float64(len(lat))*0.95)]
    p99 := lat[int(float64(len(lat))*0.99)]
    t.Logf("pingpong results - iterations=%d p50=%.2fms p95=%.2fms p99=%.2fms", iterations, p50, p95, p99)
}

// RunPingPong runs a ping-pong microbenchmark and returns latencies in ms.
func RunPingPong(ctx context.Context, url string, iterations int) ([]float64, error) {
    config := &BusConfig{
        URL:            url,
        MaxReconnect:   3,
        ReconnectWait:  1 * time.Second,
        AckWait:        5 * time.Second,
        MaxInFlight:    100,
        ConnectTimeout: 5 * time.Second,
        RequestTimeout: 5 * time.Second,
    }

    bus, err := NewNATSBus(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create bus: %w", err)
    }
    defer bus.Close()

    // Use subjects compatible with AF_MESSAGES stream (agents.*.*)
    subject := fmt.Sprintf("agents.pingpong.in-%d", time.Now().UnixNano())
    replySubj := fmt.Sprintf("agents.pingpong.reply-%d", time.Now().UnixNano())

    // Responder: subscription that replies to pings
    _, err = bus.Subscribe(context.Background(), subject, func(ctx context.Context, m *Message) error {
        // echo back on reply subject
        replyMsg := NewMessage(m.ID+"-resp", m.To, m.From, MessageTypeResponse)
        replyMsg.SetPayload(m.Payload)
        // Publish response synchronously via JetStream context to reply subject
        natsBus := bus.(*natsBus)
        natsBus.tracing.InjectTraceContext(ctx, replyMsg)
        // compute envelope hash before publish
        _ = natsBus.serializer.SetEnvelopeHash(replyMsg)
        data, _ := natsBus.serializer.Serialize(replyMsg)
        _, _ = natsBus.js.Publish(replySubj, data)
        return nil
    })
    if err != nil {
        return nil, fmt.Errorf("failed to subscribe responder: %w", err)
    }

    // Client subscriber to receive replies
    received := make(chan struct{}, iterations)
    _, err = bus.Subscribe(context.Background(), replySubj, func(ctx context.Context, m *Message) error {
        received <- struct{}{}
        return nil
    })
    if err != nil {
        return nil, fmt.Errorf("failed to subscribe client replies: %w", err)
    }

    latencies := make([]float64, 0, iterations)

    natsBus := bus.(*natsBus)
    for i := 0; i < iterations; i++ {
        id := fmt.Sprintf("ping-%d", i)
        ping := NewMessage(id, "bench-client", "bench-responder", MessageTypeRequest)
        ping.SetPayload(map[string]interface{}{"i": i})
        natsBus.tracing.InjectTraceContext(ctx, ping)
        _ = natsBus.serializer.SetEnvelopeHash(ping)
        data, _ := natsBus.serializer.Serialize(ping)

        start := time.Now()
        _, err := natsBus.js.Publish(subject, data)
        if err != nil {
            return nil, fmt.Errorf("publish failed: %w", err)
        }

        // wait for reply
        select {
        case <-received:
            lat := time.Since(start).Seconds() * 1000.0
            latencies = append(latencies, lat)
        case <-time.After(5 * time.Second):
            latencies = append(latencies, 5000.0)
        }
    }

    sort.Float64s(latencies)
    return latencies, nil
}
