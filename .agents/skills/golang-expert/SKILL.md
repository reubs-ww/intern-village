---
name: golang-expert
description: Expert Go developer for high-performance systems, concurrent programming, and cloud-native microservices. Use when writing Go code, designing Go APIs, implementing concurrency patterns, optimizing performance, writing tests, building gRPC/REST services, or any Go 1.21+ development task including CLI tools, system programming, and Kubernetes operators.
---

# Go Expert Development

Senior Go developer specializing in idiomatic, efficient, and concurrent systems.

## Before Writing Code

1. Check existing `go.mod` for module structure and dependencies
2. Review project layout and existing patterns
3. Run `go vet` and `golangci-lint` on existing code to understand standards

## Core Principles

**Simplicity over cleverness.** Go favors explicit, readable code.

```go
// Good: explicit
if err != nil {
    return fmt.Errorf("failed to process: %w", err)
}

// Bad: clever one-liners that obscure intent
```

**Accept interfaces, return structs:**
```go
func NewService(repo Repository) *Service { // interface in, struct out
    return &Service{repo: repo}
}
```

**Functional options for configurable APIs:**
```go
type Option func(*Config)

func WithTimeout(d time.Duration) Option {
    return func(c *Config) { c.Timeout = d }
}

func New(opts ...Option) *Client {
    cfg := defaultConfig()
    for _, opt := range opts {
        opt(&cfg)
    }
    return &Client{cfg: cfg}
}
```

## Error Handling

Always wrap errors with context:
```go
if err != nil {
    return fmt.Errorf("query user %s: %w", userID, err)
}
```

Custom errors for typed handling:
```go
type NotFoundError struct {
    Resource string
    ID       string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s %s not found", e.Resource, e.ID)
}

// Check with errors.As
var notFound *NotFoundError
if errors.As(err, &notFound) {
    // handle not found
}
```

## Concurrency Patterns

**Context propagation in all APIs:**
```go
func (s *Service) Process(ctx context.Context, req Request) (Response, error) {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    // ...
}
```

**Worker pool with bounded concurrency:**
```go
func process(ctx context.Context, items []Item, workers int) error {
    sem := make(chan struct{}, workers)
    g, ctx := errgroup.WithContext(ctx)

    for _, item := range items {
        item := item
        g.Go(func() error {
            select {
            case sem <- struct{}{}:
                defer func() { <-sem }()
            case <-ctx.Done():
                return ctx.Err()
            }
            return processItem(ctx, item)
        })
    }
    return g.Wait()
}
```

**Graceful shutdown:**
```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    srv := &http.Server{Addr: ":8080", Handler: handler}

    go func() {
        <-ctx.Done()
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        srv.Shutdown(shutdownCtx)
    }()

    srv.ListenAndServe()
}
```

## Testing

**Table-driven tests:**
```go
func TestParse(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    Result
        wantErr bool
    }{
        {"valid", "foo", Result{Value: "foo"}, false},
        {"empty", "", Result{}, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Parse(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("Parse() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

**Benchmark before optimizing:**
```go
func BenchmarkProcess(b *testing.B) {
    data := setupTestData()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        Process(data)
    }
}
```

## Performance

**Pre-allocate slices when size is known:**
```go
result := make([]Item, 0, len(input))
```

**Use sync.Pool for frequently allocated objects:**
```go
var bufPool = sync.Pool{
    New: func() any { return new(bytes.Buffer) },
}

func process() {
    buf := bufPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufPool.Put(buf)
    }()
    // use buf
}
```

**Efficient string building:**
```go
var b strings.Builder
b.Grow(estimatedSize)
for _, s := range parts {
    b.WriteString(s)
}
return b.String()
```

## gRPC Services

```go
type server struct {
    pb.UnimplementedServiceServer
    repo Repository
}

func (s *server) GetItem(ctx context.Context, req *pb.GetItemRequest) (*pb.Item, error) {
    if req.Id == "" {
        return nil, status.Error(codes.InvalidArgument, "id required")
    }

    item, err := s.repo.Get(ctx, req.Id)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return nil, status.Error(codes.NotFound, "item not found")
        }
        return nil, status.Error(codes.Internal, "internal error")
    }

    return toProto(item), nil
}
```

## Structured Logging (slog)

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger.Info("request processed",
    slog.String("method", r.Method),
    slog.String("path", r.URL.Path),
    slog.Duration("duration", time.Since(start)),
)
```

## Quality Checklist

Before completing:
- [ ] `gofmt -s` applied
- [ ] `go vet ./...` passes
- [ ] `golangci-lint run` clean
- [ ] Context propagated in APIs
- [ ] Errors wrapped with context
- [ ] Tests cover edge cases
- [ ] Race detector clean (`go test -race`)
- [ ] Exported items documented
