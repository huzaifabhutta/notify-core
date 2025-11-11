# Generic Queue Usage Examples

## Overview

The queue package now uses Go generics to provide **type-safe job processing**. This means:
- ✅ Compile-time type checking for all enqueued jobs
- ✅ No runtime type assertions needed
- ✅ Full IDE autocomplete and type inference
- ✅ Impossible to enqueue wrong types (caught at compile time)

---

## Basic Usage with Notification Requests

### Example 1: Email Notification Queue

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/huzaifabhutta/notify-core/internal/notify"
    "github.com/huzaifabhutta/notify-core/internal/queue"
)

func main() {
    // Create type-safe handler for notification requests
    handler := func(ctx context.Context, req *notify.SendRequest) (string, error) {
        // Handler receives typed request - no type assertion needed!
        log.Printf("Processing notification to: %s (channel: %s)", req.To, req.Channel)

        // Send via notification service
        resp, err := notifyService.Send(ctx, req)
        if err != nil {
            return "", err
        }

        return resp.MessageID, nil
    }

    // Create generic queue for *notify.SendRequest
    cfg := queue.DefaultConfig()
    q := queue.New[*notify.SendRequest](handler, cfg)
    defer q.Shutdown(context.Background())

    // Enqueue email notification (type-safe!)
    jobID, err := q.Enqueue(&notify.SendRequest{
        To:       "user@example.com",
        Channel:  notify.ChannelEmail,
        Template: "welcome",
        Subject:  "Welcome to our platform!",
        Data: map[string]interface{}{
            "Name": "John Doe",
            "URL":  "https://example.com",
        },
    })

    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Job enqueued: %s\n", jobID)

    // Check job status (type-safe!)
    job, err := q.GetJob(jobID)
    if err != nil {
        log.Fatal(err)
    }

    // Access request fields directly with full type safety
    fmt.Printf("Job status: %s\n", job.Status)
    fmt.Printf("Sending to: %s\n", job.Request.To)
    fmt.Printf("Template: %s\n", job.Request.Template)
}
```

### Type Safety in Action

```go
// ✅ CORRECT: Enqueue the right type
q.Enqueue(&notify.SendRequest{
    To:      "user@example.com",
    Channel: notify.ChannelEmail,
})

// ❌ COMPILE ERROR: Wrong type rejected at compile time
q.Enqueue("string")           // Compile error!
q.Enqueue(123)                // Compile error!
q.Enqueue(&DifferentType{})   // Compile error!
```

---

## Example 2: Custom Job Types

### Define Your Own Job Type

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/huzaifabhutta/notify-core/internal/queue"
)

// CustomJob represents a custom processing job
type CustomJob struct {
    ID        string
    UserID    int
    Action    string
    Timestamp time.Time
    Metadata  map[string]string
}

func main() {
    // Type-safe handler for CustomJob
    handler := func(ctx context.Context, job *CustomJob) (string, error) {
        // Full type safety - no assertions needed
        fmt.Printf("Processing job %s for user %d\n", job.ID, job.UserID)
        fmt.Printf("Action: %s at %v\n", job.Action, job.Timestamp)

        // Process the job
        result := processCustomJob(job)

        return result.ID, nil
    }

    // Create generic queue for *CustomJob
    cfg := queue.Config{
        Workers:    5,
        BufferSize: 100,
        MaxRetries: 3,
    }
    q := queue.New[*CustomJob](handler, cfg)
    defer q.Shutdown(context.Background())

    // Enqueue custom jobs
    jobID, err := q.Enqueue(&CustomJob{
        ID:        "job-123",
        UserID:    42,
        Action:    "process_payment",
        Timestamp: time.Now(),
        Metadata: map[string]string{
            "amount":   "99.99",
            "currency": "USD",
        },
    })

    if err != nil {
        panic(err)
    }

    fmt.Printf("Job enqueued: %s\n", jobID)

    // Get job with type safety
    job, _ := q.GetJob(jobID)

    // Access fields directly - IDE autocomplete works!
    fmt.Printf("User ID: %d\n", job.Request.UserID)
    fmt.Printf("Action: %s\n", job.Request.Action)
    fmt.Printf("Metadata: %v\n", job.Request.Metadata)
}
```

---

## Example 3: Multiple Queue Types

You can create multiple queues for different job types:

```go
package main

import (
    "context"

    "github.com/huzaifabhutta/notify-core/internal/notify"
    "github.com/huzaifabhutta/notify-core/internal/queue"
)

type EmailJob struct {
    To      string
    Subject string
    Body    string
}

type SMSJob struct {
    PhoneNumber string
    Message     string
}

type WebhookJob struct {
    URL     string
    Payload map[string]interface{}
}

func setupQueues() {
    cfg := queue.DefaultConfig()

    // Email queue
    emailHandler := func(ctx context.Context, job *EmailJob) (string, error) {
        // Send email
        return sendEmail(job.To, job.Subject, job.Body)
    }
    emailQueue := queue.New[*EmailJob](emailHandler, cfg)

    // SMS queue
    smsHandler := func(ctx context.Context, job *SMSJob) (string, error) {
        // Send SMS
        return sendSMS(job.PhoneNumber, job.Message)
    }
    smsQueue := queue.New[*SMSJob](smsHandler, cfg)

    // Webhook queue
    webhookHandler := func(ctx context.Context, job *WebhookJob) (string, error) {
        // Send webhook
        return sendWebhook(job.URL, job.Payload)
    }
    webhookQueue := queue.New[*WebhookJob](webhookHandler, cfg)

    // Use queues
    emailQueue.Enqueue(&EmailJob{
        To:      "user@example.com",
        Subject: "Hello",
        Body:    "World",
    })

    smsQueue.Enqueue(&SMSJob{
        PhoneNumber: "+1234567890",
        Message:     "Your code is 123456",
    })

    webhookQueue.Enqueue(&WebhookJob{
        URL:     "https://api.example.com/webhook",
        Payload: map[string]interface{}{"event": "user.created"},
    })
}
```

---

## Example 4: Integration with Notify Service

### Complete Integration Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/huzaifabhutta/notify-core/internal/config"
    "github.com/huzaifabhutta/notify-core/internal/logger"
    "github.com/huzaifabhutta/notify-core/internal/notify"
    "github.com/huzaifabhutta/notify-core/internal/queue"
)

// NotificationQueue is a type alias for clarity
type NotificationQueue = queue.Queue[*notify.SendRequest]

func NewNotificationQueue(service *notify.Service) *NotificationQueue {
    // Handler integrates with notify service
    handler := func(ctx context.Context, req *notify.SendRequest) (string, error) {
        log := logger.FromContext(ctx)

        log.Info().
            Str("channel", string(req.Channel)).
            Str("to", logger.MaskEmail(req.To)).
            Str("template", req.Template).
            Msg("Processing queued notification")

        // Send via notify service
        resp, err := service.Send(ctx, req)
        if err != nil {
            log.Error().
                Err(err).
                Str("channel", string(req.Channel)).
                Msg("Failed to send queued notification")
            return "", err
        }

        log.Info().
            Str("message_id", resp.MessageID).
            Str("channel", resp.Channel).
            Msg("Queued notification sent successfully")

        return resp.MessageID, nil
    }

    // Configure queue for high throughput
    cfg := queue.Config{
        Workers:    10,
        BufferSize: 200,
        MaxRetries: 3,
    }

    return queue.New[*notify.SendRequest](handler, cfg)
}

func main() {
    // Initialize logger
    logger.Init(logger.Config{
        Level:      "info",
        JSONFormat: false,
    })

    // Load config
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }

    // Create notify service
    notifyService := notify.NewService(cfg)

    // Create notification queue
    nq := NewNotificationQueue(notifyService)
    defer nq.Shutdown(context.Background())

    // Enqueue multiple notifications
    notifications := []*notify.SendRequest{
        {
            To:       "user1@example.com",
            Channel:  notify.ChannelEmail,
            Template: "welcome",
            Subject:  "Welcome!",
            Data: map[string]interface{}{
                "Name": "User 1",
            },
        },
        {
            To:       "user2@example.com",
            Channel:  notify.ChannelEmail,
            Template: "order-confirmation",
            Subject:  "Order Confirmed",
            Data: map[string]interface{}{
                "OrderID": "12345",
                "Total":   "99.99",
            },
        },
        {
            To:       "+1234567890",
            Channel:  notify.ChannelWhatsApp,
            Template: "otp",
            Data: map[string]interface{}{
                "code": "123456",
            },
        },
    }

    // Enqueue all notifications
    jobIDs := make([]string, 0, len(notifications))
    for _, notif := range notifications {
        jobID, err := nq.Enqueue(notif)
        if err != nil {
            log.Printf("Failed to enqueue: %v", err)
            continue
        }
        jobIDs = append(jobIDs, jobID)
        fmt.Printf("Enqueued job: %s\n", jobID)
    }

    // Wait for processing
    time.Sleep(2 * time.Second)

    // Check status of all jobs
    stats := nq.Stats()
    fmt.Printf("\nQueue Statistics:\n")
    fmt.Printf("Total: %d\n", stats["total"])
    fmt.Printf("Pending: %d\n", stats["pending"])
    fmt.Printf("Processing: %d\n", stats["processing"])
    fmt.Printf("Completed: %d\n", stats["completed"])
    fmt.Printf("Failed: %d\n", stats["failed"])

    // Check individual job status
    for _, jobID := range jobIDs {
        job, err := nq.GetJob(jobID)
        if err != nil {
            log.Printf("Failed to get job %s: %v", jobID, err)
            continue
        }

        fmt.Printf("\nJob %s:\n", jobID)
        fmt.Printf("  Status: %s\n", job.Status)
        fmt.Printf("  To: %s\n", job.Request.To)
        fmt.Printf("  Channel: %s\n", job.Request.Channel)
        fmt.Printf("  Template: %s\n", job.Request.Template)
        if job.Error != nil {
            fmt.Printf("  Error: %v\n", job.Error)
        }
    }
}
```

---

## Example 5: Advanced Usage - Monitoring and Metrics

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/huzaifabhutta/notify-core/internal/notify"
    "github.com/huzaifabhutta/notify-core/internal/queue"
)

type MetricsCollector struct {
    totalProcessed int
    totalFailed    int
    avgDuration    time.Duration
}

func (m *MetricsCollector) RecordSuccess(duration time.Duration) {
    m.totalProcessed++
    m.avgDuration = (m.avgDuration + duration) / 2
}

func (m *MetricsCollector) RecordFailure() {
    m.totalFailed++
}

func SetupMonitoredQueue(service *notify.Service, metrics *MetricsCollector) *queue.Queue[*notify.SendRequest] {
    handler := func(ctx context.Context, req *notify.SendRequest) (string, error) {
        start := time.Now()

        // Send notification
        resp, err := service.Send(ctx, req)

        duration := time.Since(start)

        if err != nil {
            metrics.RecordFailure()
            return "", err
        }

        metrics.RecordSuccess(duration)
        return resp.MessageID, nil
    }

    cfg := queue.DefaultConfig()
    return queue.New[*notify.SendRequest](handler, cfg)
}

func MonitorQueue(q *queue.Queue[*notify.SendRequest]) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        stats := q.Stats()
        fmt.Printf("[%s] Queue Stats: Total=%d, Pending=%d, Processing=%d, Completed=%d, Failed=%d\n",
            time.Now().Format("15:04:05"),
            stats["total"],
            stats["pending"],
            stats["processing"],
            stats["completed"],
            stats["failed"],
        )
    }
}
```

---

## Benefits of Generic Implementation

### 1. Compile-Time Type Safety

**Before (interface{}):**
```go
// Could enqueue anything - errors at runtime
q.Enqueue("wrong")     // ✅ Compiles, ❌ fails at runtime
q.Enqueue(123)         // ✅ Compiles, ❌ fails at runtime
q.Enqueue(&RightType{}) // ✅ Compiles, ✅ works at runtime
```

**After (generics):**
```go
// Only correct type compiles
q := New[*RightType](handler, cfg)
q.Enqueue(&RightType{})  // ✅ Compiles, ✅ works
q.Enqueue("wrong")       // ❌ Compile error!
q.Enqueue(123)           // ❌ Compile error!
```

### 2. Better IDE Support

```go
job, _ := q.GetJob(jobID)

// With generics: IDE knows job.Request is *notify.SendRequest
job.Request.To        // ✅ Autocomplete works!
job.Request.Channel   // ✅ Autocomplete works!
job.Request.Template  // ✅ Autocomplete works!

// Without generics: IDE shows interface{}
job.Request.To        // ❌ No autocomplete
// Need: req := job.Request.(*notify.SendRequest) // Type assertion
```

### 3. No Runtime Overhead

Generics are **compile-time only** - they have zero runtime cost:
- Same performance as interface{} version
- No reflection needed
- Monomorphization during compilation

### 4. Self-Documenting Code

```go
// Clear from the type what this queue handles
var emailQueue *queue.Queue[*EmailJob]
var smsQueue *queue.Queue[*SMSJob]
var notifyQueue *queue.Queue[*notify.SendRequest]

// VS unclear interface{} version
var emailQueue *queue.Queue  // What does it handle? 🤷
```

---

## Migration from interface{} Version

If you have existing code using the old `interface{}` version:

### Step 1: Update imports (no change needed)
```go
import "github.com/huzaifabhutta/notify-core/internal/queue"
```

### Step 2: Update queue creation
```go
// Old:
q := queue.New(handler, cfg)

// New:
q := queue.New[*YourRequestType](handler, cfg)
```

### Step 3: Update handler signature
```go
// Old:
handler := func(ctx context.Context, req interface{}) (string, error) {
    myReq := req.(*notify.SendRequest)  // Type assertion
    // ...
}

// New:
handler := func(ctx context.Context, req *notify.SendRequest) (string, error) {
    // req is already typed!
    // ...
}
```

### Step 4: Update Enqueue calls
```go
// Old:
q.Enqueue(request)  // Accepts anything

// New:
q.Enqueue(request)  // Only accepts *YourRequestType
```

That's it! The rest of the API is identical.

---

## Best Practices

1. **Use type aliases for clarity:**
   ```go
   type NotificationQueue = queue.Queue[*notify.SendRequest]
   type EmailQueue = queue.Queue[*EmailJob]
   ```

2. **Create factory functions:**
   ```go
   func NewNotificationQueue(service *notify.Service) *NotificationQueue {
       handler := func(ctx context.Context, req *notify.SendRequest) (string, error) {
           return service.Send(ctx, req)
       }
       return queue.New[*notify.SendRequest](handler, queue.DefaultConfig())
   }
   ```

3. **Use pointer types for structs:**
   ```go
   // ✅ Recommended
   queue.New[*MyStruct](handler, cfg)

   // ⚠️ Avoid (unnecessary copying)
   queue.New[MyStruct](handler, cfg)
   ```

4. **Monitor queue statistics:**
   ```go
   stats := q.Stats()
   if stats["failed"] > 0 {
       log.Printf("Warning: %d failed jobs", stats["failed"])
   }
   ```

---

## Conclusion

The generic queue provides:
- ✅ **Type safety** at compile time
- ✅ **Better developer experience** with autocomplete
- ✅ **Zero runtime overhead** (same performance)
- ✅ **Clear, self-documenting code**
- ✅ **Impossible to use incorrectly** (compile errors prevent mistakes)

Use it wherever you need reliable, type-safe job processing!
