# Go Generics Analysis - notify-core

## Executive Summary

After comprehensive code review, **1 strong candidate** and **2 potential candidates** were identified where Go generics would genuinely improve code quality, maintainability, and type safety.

**Recommendation**: Implement generics for the Queue system (high value, low risk).

---

## 🟢 STRONG RECOMMENDATION: Generic Queue

### Current Implementation (interface{} based)

**Location**: `internal/queue/queue.go`

**Problems**:
```go
type Job struct {
    ID        string
    Request   interface{}  // ❌ No type safety
    Status    JobStatus
    Error     error
    CreatedAt time.Time
    UpdatedAt time.Time
    Retries   int
}

type Handler func(ctx context.Context, req interface{}) (messageID string, err error)
type Queue struct {
    jobs       chan *Job
    jobStore   map[string]*Job
    handler    Handler
    // ...
}

// Usage requires type assertion:
func (q *Queue) worker(id int) {
    messageID, err := q.handler(ctx, job.Request)  // ❌ job.Request is interface{}
}
```

**Issues**:
1. ❌ **No compile-time type safety** - can enqueue any type
2. ❌ **Runtime errors** - type assertions can panic
3. ❌ **Poor IDE support** - no autocomplete for Request fields
4. ❌ **Harder to reason about** - what type is Request?
5. ❌ **Testing complexity** - need to handle all possible types

### Proposed Generic Implementation

```go
// Generic Job type
type Job[T any] struct {
    ID        string
    Request   T  // ✅ Type-safe!
    Status    JobStatus
    Error     error
    CreatedAt time.Time
    UpdatedAt time.Time
    Retries   int
}

// Generic Handler
type Handler[T any] func(ctx context.Context, req T) (messageID string, err error)

// Generic Queue
type Queue[T any] struct {
    jobs       chan *Job[T]
    results    chan *Result
    jobStore   map[string]*Job[T]
    storeMu    sync.RWMutex
    handler    Handler[T]
    workers    int
    maxRetries int
    wg         sync.WaitGroup
    ctx        context.Context
    cancel     context.CancelFunc
}

// Constructor
func New[T any](handler Handler[T], cfg Config) *Queue[T] {
    // ... same implementation
}

// Enqueue with type safety
func (q *Queue[T]) Enqueue(req T) (string, error) {
    job := &Job[T]{
        ID:        uuid.New().String(),
        Request:   req,  // ✅ Type-checked at compile time
        Status:    JobStatusPending,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
        Retries:   0,
    }
    // ... rest of implementation
}

// GetJob returns type-safe job
func (q *Queue[T]) GetJob(jobID string) (*Job[T], error) {
    // ... implementation
}
```

### Usage Example

**Before (interface{}):**
```go
// Can enqueue anything - no type safety
queue.Enqueue("wrong type")  // ✅ Compiles but will fail at runtime!
queue.Enqueue(123)           // ✅ Compiles but will fail at runtime!
queue.Enqueue(&notify.SendRequest{...})  // ✅ Correct, but not enforced
```

**After (generics):**
```go
// Type-safe queue
type NotificationQueue = Queue[*notify.SendRequest]

handler := func(ctx context.Context, req *notify.SendRequest) (string, error) {
    return notifyService.Send(ctx, req)  // ✅ req is *notify.SendRequest
}

queue := New[*notify.SendRequest](handler, cfg)

// Type checked at compile time
queue.Enqueue(&notify.SendRequest{...})  // ✅ Correct
queue.Enqueue("wrong type")              // ❌ Compile error!
queue.Enqueue(123)                       // ❌ Compile error!

// Get job with type safety
job, err := queue.GetJob(jobID)
fmt.Println(job.Request.To)  // ✅ IDE autocomplete works!
```

### Benefits

1. ✅ **Compile-time type safety** - wrong types caught at compile time
2. ✅ **No runtime overhead** - generics compile away, zero cost abstraction
3. ✅ **Better IDE support** - autocomplete, refactoring, go-to-definition
4. ✅ **Easier testing** - mock with concrete types
5. ✅ **Self-documenting** - `Queue[*notify.SendRequest]` is clear
6. ✅ **Future-proof** - can create specialized queues easily

### Migration Path

**Step 1**: Create generic version alongside existing:
```go
// Keep old version for compatibility
type Queue = QueueLegacy

// New generic version
type QueueGeneric[T any] struct { ... }
```

**Step 2**: Update tests to use generic version:
```go
func TestQueue(t *testing.T) {
    handler := func(ctx context.Context, req *TestRequest) (string, error) {
        return req.Process()
    }
    queue := New[*TestRequest](handler, DefaultConfig())
    // ... tests
}
```

**Step 3**: Update production code:
```go
// In service initialization
notifyQueue := queue.New[*notify.SendRequest](
    func(ctx context.Context, req *notify.SendRequest) (string, error) {
        resp, err := notifyService.Send(ctx, req)
        if err != nil {
            return "", err
        }
        return resp.MessageID, nil
    },
    queue.DefaultConfig(),
)
```

**Step 4**: Remove legacy version after verification

### Risk Assessment

- **Risk**: 🟢 **LOW** - Isolated change, doesn't affect adapters or API
- **Effort**: 🟡 **MEDIUM** - ~2 hours for implementation + tests
- **Value**: 🟢 **HIGH** - Significant type safety improvement

---

## 🟡 POTENTIAL: Generic Adapter Interface

### Current Implementation

**Location**: `internal/notify/service.go`, adapter implementations

**Problem**:
```go
type Adapter interface {
    Send(ctx context.Context, req interface{}) (messageID string, err error)
    Name() string
}

// Stored in map - all adapters must use same interface type
type Service struct {
    adapters map[Channel]Adapter
}
```

### Why Generics DON'T Help Here

**Attempted generic approach:**
```go
type Adapter[T any] interface {
    Send(ctx context.Context, req T) (messageID string, err error)
    Name() string
}

// Problem: Can't store different generic types in same map
type Service struct {
    adapters map[Channel]Adapter[???]  // ❌ What type parameter?
}

// Would need:
adapters map[Channel]Adapter[any]  // ❌ Defeats the purpose
```

**Architectural constraint**: The service needs to store multiple adapter types (email, whatsapp, sms) in the same map. Each has different request types. Generics don't help with heterogeneous collections.

**Current reflection-based approach is correct** for this use case.

### Alternative: Type-safe Adapter Registration

If you want type safety at registration time:

```go
// Keep interface{} in adapter interface
type Adapter interface {
    Send(ctx context.Context, req interface{}) (messageID string, err error)
    Name() string
}

// Generic registration function
func RegisterAdapter[T any](s *Service, channel Channel, adapter Adapter, validator func(interface{}) (*T, bool)) {
    s.adapters[channel] = &typedAdapter[T]{
        underlying: adapter,
        validator:  validator,
    }
}
```

**Verdict**: ❌ **Not recommended** - adds complexity without significant benefit

---

## 🟡 POTENTIAL: Generic Result Type

### Current Implementation

Go idiomatic multiple return values:
```go
func Send(ctx context.Context, req *SendRequest) (*SendResponse, error) {
    // ...
    return response, nil
}
```

### Potential Generic Approach

```go
type Result[T any, E any] struct {
    Value T
    Error E
}

func (r Result[T, E]) IsOk() bool {
    // ...
}

func (r Result[T, E]) Unwrap() (T, error) {
    // ...
}

// Usage
func Send(ctx context.Context, req *SendRequest) Result[*SendResponse, error] {
    // ...
}
```

### Why NOT Recommended

1. ❌ **Against Go idioms** - multiple return values are standard
2. ❌ **Reduces readability** - more verbose than `(value, error)`
3. ❌ **No real benefit** - Go already has excellent error handling
4. ❌ **Community expects** `(T, error)` pattern
5. ❌ **Breaks compatibility** - all existing code would need changes

**Verdict**: ❌ **Not recommended** - stick with Go conventions

---

## ❌ NOT RECOMMENDED: Generic Adapter Common

### Current Implementation

**Location**: `internal/adapters/common.go`

Uses reflection to extract fields:
```go
func ExtractBaseRequest(req interface{}) (BaseRequest, error) {
    val := reflect.ValueOf(req)
    // ... reflection code to extract fields
}
```

### Why Reflection is Correct Here

**Architectural reason**: Avoids import cycles

```
internal/notify/service.go
    ↓ imports
internal/email/adapter.go
    ↓ would import (if using direct types)
internal/notify/service.go  ← CYCLE!
```

**Attempted generic approach:**
```go
type BaseRequestProvider interface {
    GetTo() string
    GetTemplate() string
    GetSubject() string
    GetFrom() string
    GetData() map[string]interface{}
}

func ExtractBaseRequest[T BaseRequestProvider](req T) BaseRequest {
    return BaseRequest{
        To:       req.GetTo(),
        Template: req.GetTemplate(),
        // ...
    }
}

// Problem: notify.SendRequest would need to implement interface
// This requires importing notify package → import cycle!
```

**Alternative**: Struct embedding, but creates tight coupling

```go
// In adapters package
type BaseRequest struct {
    To       string
    Template string
    // ...
}

// In notify package
type SendRequest struct {
    adapters.BaseRequest  // ❌ Creates dependency on adapters
    Channel Channel
}
```

**Verdict**: ❌ **Not recommended** - reflection is the right tool here

---

## 📊 Summary & Recommendations

| Area | Current | Generic? | Recommendation | Value | Risk | Effort |
|------|---------|----------|----------------|-------|------|--------|
| **Queue** | `interface{}` | ✅ **YES** | **IMPLEMENT** | 🟢 HIGH | 🟢 LOW | 🟡 MEDIUM |
| Adapter Interface | `interface{}` | ❌ NO | Keep as-is | - | - | - |
| Result Type | `(T, error)` | ❌ NO | Keep as-is | - | - | - |
| Common Extraction | Reflection | ❌ NO | Keep as-is | - | - | - |

### Priority Actions

**1. Implement Generic Queue** ⭐ **HIGH PRIORITY**

```bash
# Implementation checklist:
□ Create Queue[T any] with all methods
□ Update queue tests to use concrete types
□ Add example usage in comments
□ Update WEEK2_COMPLETION.md with generic queue
□ Create migration guide for existing code
```

**Expected Outcome**:
- ✅ Type-safe job enqueueing
- ✅ Better developer experience
- ✅ Fewer runtime errors
- ✅ Same performance (zero-cost abstraction)

**2. Keep Everything Else As-Is** ✅ **CORRECT APPROACH**

The rest of the codebase correctly uses:
- `interface{}` where heterogeneous types are needed
- Reflection where import cycles would occur
- Standard Go idioms for error handling

---

## Implementation Example: Generic Queue

### Complete Implementation

```go
package queue

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/google/uuid"
)

// Job represents a notification job in the queue (GENERIC)
type Job[T any] struct {
    ID        string
    Request   T
    Status    JobStatus
    Error     error
    CreatedAt time.Time
    UpdatedAt time.Time
    Retries   int
}

// Handler is the function signature for processing jobs (GENERIC)
type Handler[T any] func(ctx context.Context, req T) (messageID string, err error)

// Queue is an in-memory job queue with goroutine workers (GENERIC)
type Queue[T any] struct {
    jobs       chan *Job[T]
    results    chan *Result
    jobStore   map[string]*Job[T]
    storeMu    sync.RWMutex
    handler    Handler[T]
    workers    int
    maxRetries int
    wg         sync.WaitGroup
    ctx        context.Context
    cancel     context.CancelFunc
}

// New creates a new job queue (GENERIC CONSTRUCTOR)
func New[T any](handler Handler[T], cfg Config) *Queue[T] {
    ctx, cancel := context.WithCancel(context.Background())

    q := &Queue[T]{
        jobs:       make(chan *Job[T], cfg.BufferSize),
        results:    make(chan *Result, cfg.BufferSize),
        jobStore:   make(map[string]*Job[T]),
        handler:    handler,
        workers:    cfg.Workers,
        maxRetries: cfg.MaxRetries,
        ctx:        ctx,
        cancel:     cancel,
    }

    // Start worker goroutines
    for i := 0; i < cfg.Workers; i++ {
        q.wg.Add(1)
        go q.worker(i)
    }

    // Start result processor
    q.wg.Add(1)
    go q.processResults()

    return q
}

// Enqueue adds a job to the queue (TYPE-SAFE)
func (q *Queue[T]) Enqueue(req T) (string, error) {
    job := &Job[T]{
        ID:        uuid.New().String(),
        Request:   req,
        Status:    JobStatusPending,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
        Retries:   0,
    }

    q.storeMu.Lock()
    q.jobStore[job.ID] = job
    q.storeMu.Unlock()

    select {
    case q.jobs <- job:
        return job.ID, nil
    case <-q.ctx.Done():
        return "", fmt.Errorf("queue is shutting down")
    default:
        return "", fmt.Errorf("queue is full")
    }
}

// GetJob returns the status of a job by ID (TYPE-SAFE)
func (q *Queue[T]) GetJob(jobID string) (*Job[T], error) {
    q.storeMu.RLock()
    defer q.storeMu.RUnlock()

    job, ok := q.jobStore[jobID]
    if !ok {
        return nil, fmt.Errorf("job not found: %s", jobID)
    }

    jobCopy := *job
    return &jobCopy, nil
}

// worker processes jobs from the queue
func (q *Queue[T]) worker(id int) {
    defer q.wg.Done()

    for {
        select {
        case <-q.ctx.Done():
            return
        case job, ok := <-q.jobs:
            if !ok {
                return
            }

            q.updateJobStatus(job.ID, JobStatusProcessing, nil)

            ctx, cancel := context.WithTimeout(q.ctx, 30*time.Second)
            messageID, err := q.handler(ctx, job.Request)  // ✅ Type-safe!
            cancel()

            result := &Result{
                JobID:     job.ID,
                Success:   err == nil,
                Error:     err,
                MessageID: messageID,
            }

            select {
            case q.results <- result:
            case <-q.ctx.Done():
                return
            }
        }
    }
}

// ... rest of methods unchanged
```

### Usage in Production

```go
// In internal/notify/service.go or main.go

import (
    "github.com/huzaifabhutta/notify-core/internal/notify"
    "github.com/huzaifabhutta/notify-core/internal/queue"
)

// Create type-safe notification queue
type NotificationQueue = queue.Queue[*notify.SendRequest]

// Initialize with type-safe handler
func NewNotificationQueue(service *notify.Service) *NotificationQueue {
    handler := func(ctx context.Context, req *notify.SendRequest) (string, error) {
        resp, err := service.Send(ctx, req)
        if err != nil {
            return "", err
        }
        return resp.MessageID, nil
    }

    return queue.New[*notify.SendRequest](handler, queue.DefaultConfig())
}

// Usage
queue := NewNotificationQueue(notifyService)

// Type-safe enqueueing
jobID, err := queue.Enqueue(&notify.SendRequest{
    To:       "user@example.com",
    Channel:  notify.ChannelEmail,
    Template: "welcome",
    Subject:  "Welcome!",
    Data: map[string]interface{}{
        "name": "John",
    },
})

// Type-safe job retrieval
job, err := queue.GetJob(jobID)
if err == nil {
    // job.Request is *notify.SendRequest - full type safety!
    fmt.Println("Sending to:", job.Request.To)
    fmt.Println("Template:", job.Request.Template)
}
```

---

## Testing Strategy

```go
package queue

import (
    "context"
    "testing"
)

type TestRequest struct {
    ID   string
    Data string
}

func TestGenericQueue(t *testing.T) {
    handler := func(ctx context.Context, req *TestRequest) (string, error) {
        return "msg-" + req.ID, nil
    }

    q := New[*TestRequest](handler, DefaultConfig())
    defer q.Shutdown(context.Background())

    // Type-safe enqueueing
    jobID, err := q.Enqueue(&TestRequest{
        ID:   "123",
        Data: "test",
    })
    if err != nil {
        t.Fatalf("Enqueue failed: %v", err)
    }

    // Type-safe retrieval
    job, err := q.GetJob(jobID)
    if err != nil {
        t.Fatalf("GetJob failed: %v", err)
    }

    // Can access typed fields directly
    if job.Request.ID != "123" {
        t.Errorf("Expected ID=123, got %s", job.Request.ID)
    }
}

// Compile-time type safety test
func TestTypeError(t *testing.T) {
    handler := func(ctx context.Context, req *TestRequest) (string, error) {
        return "", nil
    }

    q := New[*TestRequest](handler, DefaultConfig())

    // This would be a compile error:
    // q.Enqueue("wrong type")  // ❌ Compile error!
    // q.Enqueue(123)           // ❌ Compile error!

    // Only correct type works:
    q.Enqueue(&TestRequest{ID: "456"})  // ✅ Compiles!
}
```

---

## Conclusion

**Go generics should be used for the Queue system** - this is a clear win for type safety with zero runtime cost.

**Generics should NOT be used** for adapter interfaces or common extraction - the current implementation correctly uses reflection and interface{} where architectural constraints require it.

**Key Principle**: Use generics when you have:
1. ✅ Type-safe containers/collections needed
2. ✅ Single concrete type per instance
3. ✅ No import cycle issues
4. ✅ Real benefit over interface{}

Don't use generics when:
1. ❌ Need heterogeneous collections
2. ❌ Would create import cycles
3. ❌ Go idioms work better (e.g., multiple return values)
4. ❌ No real type safety benefit

**Status**: Queue generics implementation is **recommended and ready for implementation**.
