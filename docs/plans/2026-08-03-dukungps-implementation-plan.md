# DukunGPS Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a production-ready GPS tracker server in Go using Clean Architecture, with Traccar as protocol reference.

**Architecture:** Clean Architecture (handler → usecase → repository) with modular protocol layer. Each protocol (Teltonika, GT06, Huabao, etc.) is a self-contained package that decodes device-specific formats into normalized Position objects. The core engine is protocol-agnostic — protocols plug into a shared pipeline for validation, geofencing, caching, and persistence.

**Tech Stack:**
- Language: Go 1.24+
- Framework: Fiber v3 (HTTP server, WebSocket)
- Database: PostgreSQL + pgx
- Cache: Redis (optional, for real-time position cache)
- Protocol: Netty-inspired TCP server using gorilla/websocket + raw net.Conn
- Build: Makefile
- Testing: testify + testcontainers (integration)

---

## Phase 1: Project Foundation

### Task 1: Initialize Go module and folder structure

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `Makefile`

**Step 1: Initialize Go module**

```bash
go mod init github.com/akordium-id/dukungps
```

**Step 2: Add dependencies**

```bash
go get github.com/gofiber/fiber/v3
go get github.com/jackc/pgx/v5
go get github.com/redis/go-redis/v9
go get github.com/stretchr/testify
```

**Step 3: Create folder structure**

```
dukungps/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── device/
│   │   ├── position/
│   │   ├── geofence/
│   │   ├── user/
│   │   └── command/
│   ├── protocol/
│   │   ├── teltonika/
│   │   ├── gt06/
│   │   └── huabao/
│   ├── usecase/
│   ├── repository/
│   └── infrastructure/
│       ├── database/
│       ├── cache/
│       └── websocket/
├── pkg/
│   └── utils/
├── migrations/
├── deployments/docker/
└── docs/plans/
```

**Step 4: Commit**

```bash
git add go.mod go.sum .gitignore Makefile
git commit -m "chore: initialize project structure"
```

---

### Task 2: Define domain entities

**Files:**
- Create: `internal/domain/device/entity.go`
- Create: `internal/domain/position/entity.go`
- Create: `internal/domain/geofence/entity.go`
- Create: `internal/domain/user/entity.go`
- Create: `internal/domain/command/entity.go`

**Step 1: Define Device entity**

```go
package device

type Device struct {
    ID           int64
    UniqueID     string
    Name         string
    Model        string
    Protocol     string
    Status       string
    LastPosition *Position
    CreatedAt    int64
    UpdatedAt    int64
}
```

**Step 2: Define Position entity**

```go
package position

type Position struct {
    ID          int64
    DeviceID    int64
    Latitude    float64
    Longitude   float64
    Altitude    float64
    Speed       float64
    Course      float64
    Valid       bool
    Time        int64
    DeviceTime  int64
    Attributes  map[string]interface{}
    CreatedAt   int64
}
```

**Step 3: Define Geofence entity**

```go
package geofence

type Geofence struct {
    ID          int64
    Name        string
    Description string
    Area        []Coordinate
    Type        string
    CreatedAt   int64
    UpdatedAt   int64
}

type Coordinate struct {
    Latitude  float64
    Longitude float64
}
```

**Step 4: Define User entity**

```go
package user

type User struct {
    ID        int64
    Name      string
    Email     string
    Password  string
    Role      string
    CreatedAt int64
    UpdatedAt int64
}
```

**Step 5: Define Command entity**

```go
package command

type Command struct {
    ID        int64
    DeviceID  int64
    Type      string
    Data      string
    Status    string
    SentAt    int64
    CreatedAt int64
}
```

**Step 6: Write tests**

```go
func TestDevice_IsValid() {
    d := Device{UniqueID: "123456789012345", Name: "Test"}
    assert.NotEmpty(t, d.UniqueID)
}
```

**Step 7: Commit**

```bash
git add internal/domain/
git commit -m "feat: define domain entities"
```

---

## Phase 2: Protocol Layer

### Task 3: Define protocol interfaces and types

**Files:**
- Create: `internal/protocol/protocol.go`

**Step 1: Define protocol interface**

```go
package protocol

import (
    "net"
)

type Protocol interface {
    Name() string
    Decoder() Decoder
    Encoder() Encoder
}

type Decoder interface {
    Decode(data []byte, conn net.Conn, remoteAddr net.Addr) ([]Position, error)
}

type Encoder interface {
    Encode(cmd Command) ([]byte, error)
}

type Position struct {
    DeviceID    int64
    Latitude    float64
    Longitude   float64
    Altitude    float64
    Speed       float64
    Course      float64
    Valid       bool
    Time        int64
    DeviceTime  int64
    Attributes  map[string]interface{}
}

type Command struct {
    Type string
    Data string
}
```

**Step 2: Write tests**

```go
func TestProtocolInterface() {
    var p Protocol
    assert.NotNil(t, p)
}
```

**Step 3: Commit**

```bash
git add internal/protocol/protocol.go
git commit -m "feat: define protocol interfaces"
```

---

### Task 4: Implement Teltonika protocol decoder

**Files:**
- Create: `internal/protocol/teltonika/decoder.go`
- Create: `internal/protocol/teltonika/encoder.go`
- Create: `internal/protocol/teltonika/teltonika_test.go`

**Step 1: Implement decoder**

Reference: Traccar TeltonikaProtocolDecoder.java (codec 8, 8 extended, 12, 13)

```go
package teltonika

import (
    "encoding/binary"
    "time"
)

const (
    CODEC_8       = 0x08
    CODEC_8_EXT   = 0x8E
    CODEC_12      = 0x0C
    CODEC_13      = 0x0D
    CODEC_GH3000  = 0x07
)

type TeltonikaDecoder struct {
    data []byte
}

func NewTeltonikaDecoder(data []byte) *TeltonikaDecoder {
    return &TeltonikaDecoder{data: data}
}

func (d *TeltonikaDecoder) Decode() ([]Position, error) {
    // Implementation based on Traccar's TeltonikaProtocolDecoder
    // Parse codec, record count, AVL data
}
```

**Step 2: Write failing test**

```go
func TestTeltonikaDecoder_Codec8() {
    // Use Traccar's known test payload for codec 8
    payload := []byte{0x08, ...}
    decoder := NewTeltonikaDecoder(payload)
    positions, err := decoder.Decode()
    assert.NoError(t, err)
    assert.Len(t, positions, 1)
    assert.Equal(t, 51.5074, positions[0].Latitude)
}
```

**Step 3: Run test (expect fail)**

```bash
go test ./internal/protocol/teltonika/
```

**Step 4: Implement minimal decoder**

```go
func (d *TeltonikaDecoder) Decode() ([]Position, error) {
    // Minimal implementation to pass test
}
```

**Step 5: Run test (expect pass)**

```bash
go test ./internal/protocol/teltonika/ -v
```

**Step 6: Commit**

```bash
git add internal/protocol/teltonika/
git commit -m "feat: implement Teltonika protocol decoder"
```

---

### Task 5: Implement GT06 protocol decoder

**Files:**
- Create: `internal/protocol/gt06/decoder.go`
- Create: `internal/protocol/gt06/gt06_test.go`

**Step 1-6:** Follow same TDD pattern as Task 4

Reference: Traccar Gt06ProtocolDecoder.java (start marker 0x7878, CRC16, message types)

---

### Task 6: Implement Huabao protocol decoder

**Files:**
- Create: `internal/protocol/huabao/decoder.go`
- Create: `internal/protocol/huabao/huabao_test.go`

**Step 1-6:** Follow same TDD pattern as Task 4

Reference: Traccar HuabaoProtocolDecoder.java (0x7e framing, byte stuffing)

---

## Phase 3: Infrastructure Layer

### Task 7: Set up PostgreSQL repository

**Files:**
- Create: `internal/infrastructure/database/postgres.go`
- Create: `internal/repository/device_repo.go`
- Create: `internal/repository/position_repo.go`
- Create: `migrations/001_initial_schema.sql`

**Step 1: Write database schema**

```sql
CREATE TABLE devices (
    id SERIAL PRIMARY KEY,
    unique_id VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(255),
    model VARCHAR(255),
    protocol VARCHAR(64),
    status VARCHAR(32),
    last_position_id INT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE positions (
    id SERIAL PRIMARY KEY,
    device_id INT NOT NULL REFERENCES devices(id),
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),
    altitude DECIMAL(8,2),
    speed DECIMAL(6,2),
    course DECIMAL(6,2),
    valid BOOLEAN,
    time TIMESTAMP,
    device_time TIMESTAMP,
    attributes JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE geofences (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255),
    description TEXT,
    area JSONB,
    type VARCHAR(32),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

**Step 2: Implement repository interfaces**

```go
type DeviceRepository interface {
    Create(device *device.Device) error
    FindByUniqueID(uniqueID string) (*device.Device, error)
    Update(device *device.Device) error
}

type PositionRepository interface {
    Create(position *position.Position) error
    FindByDeviceID(deviceID int64, limit int) ([]position.Position, error)
}
```

**Step 3: Implement PostgreSQL repository**

```go
type PostgresDeviceRepo struct {
    db *pgx.Conn
}

func (r *PostgresDeviceRepo) Create(d *device.Device) error {
    _, err := r.db.Exec(ctx,
        "INSERT INTO devices (unique_id, name, protocol) VALUES ($1, $2, $3)",
        d.UniqueID, d.Name, d.Protocol)
    return err
}
```

**Step 4: Write tests with testcontainers**

```go
func TestPostgresDeviceRepo_Create(t *testing.T) {
    // Spin up PostgreSQL in testcontainers
    // Test CRUD operations
}
```

**Step 5: Commit**

```bash
git add internal/repository/ internal/infrastructure/database/ migrations/
git commit -m "feat: implement PostgreSQL repository"
```

---

### Task 8: Set up Redis cache

**Files:**
- Create: `internal/infrastructure/cache/redis.go`
- Create: `internal/infrastructure/cache/cache.go`

**Step 1: Define cache interface**

```go
type Cache interface {
    SetPosition(deviceID int64, pos position.Position) error
    GetPosition(deviceID int64) (*position.Position, error)
    SetDeviceOnline(deviceID int64) error
    IsDeviceOnline(deviceID int64) (bool, error)
}
```

**Step 2: Implement Redis cache**

```go
type RedisCache struct {
    client *redis.Client
}

func (c *RedisCache) SetPosition(deviceID int64, pos position.Position) error {
    data, _ := json.Marshal(pos)
    return c.client.Set(ctx, fmt.Sprintf("device:%d:position", deviceID), data, 0).Err()
}
```

**Step 3: Write tests**

```go
func TestRedisCache_SetGetPosition(t *testing.T) {
    // Test with testcontainers-redis or mock
}
```

**Step 4: Commit**

```bash
git add internal/infrastructure/cache/
git commit -m "feat: implement Redis cache"
```

---

### Task 9: Set up WebSocket broadcaster

**Files:**
- Create: `internal/infrastructure/websocket/hub.go`
- Create: `internal/infrastructure/websocket/client.go`

**Step 1: Define hub**

```go
type Hub struct {
    clients map[*Client]bool
    register chan *Client
    unregister chan *Client
    broadcast chan PositionUpdate
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client] = true
        case client := <-h.unregister:
            delete(h.clients, client)
        case update := <-h.broadcast:
            for client := range h.clients {
                client.send <- update
            }
        }
    }
}
```

**Step 2: Commit**

```bash
git add internal/infrastructure/websocket/
git commit -m "feat: implement WebSocket hub"
```

---

## Phase 4: Use Case Layer

### Task 10: Implement position processing pipeline

**Files:**
- Create: `internal/usecase/position_service.go`

**Step 1: Define service**

```go
type PositionService struct {
    positionRepo repository.PositionRepository
    deviceRepo   repository.DeviceRepository
    cache        cache.Cache
    hub          *websocket.Hub
}

func (s *PositionService) ProcessPosition(pos position.Position) error {
    if err := s.positionRepo.Create(&pos); err != nil {
        return err
    }
    s.cache.SetPosition(pos.DeviceID, pos)
    s.hub.broadcast <- PositionUpdate{DeviceID: pos.DeviceID, Position: pos}
    return nil
}
```

**Step 2: Write tests**

```go
func TestPositionService_ProcessPosition(t *testing.T) {
    // Mock repositories, test pipeline
}
```

**Step 3: Commit**

```bash
git add internal/usecase/
git commit -m "feat: implement position processing service"
```

---

## Phase 5: Protocol Server

### Task 11: Implement TCP server for protocols

**Files:**
- Create: `internal/protocol/server.go`

**Step 1: Define server**

```go
type Server struct {
    protocols map[string]Protocol
    positionService *usecase.PositionService
}

func (s *Server) Start() error {
    for name, proto := range s.protocols {
        go s.listen(name, proto)
    }
    return nil
}
```

**Step 2: Commit**

```bash
git add internal/protocol/server.go
git commit -m "feat: implement TCP protocol server"
```

---

## Phase 6: HTTP API

### Task 12: Implement Fiber API

**Files:**
- Create: `cmd/server/main.go`
- Create: `internal/api/handler/position_handler.go`
- Create: `internal/api/handler/device_handler.go`
- Create: `internal/api/handler/geofence_handler.go`

**Step 1: Define routes**

```go
func SetupRoutes(app *fiber.App, services *Services) {
    api := app.Group("/api/v1")
    api.Get("/positions", services.PositionHandler.List)
    api.Get("/devices", services.DeviceHandler.List)
    api.Get("/devices/:id", services.DeviceHandler.Get)
    api.Post("/geofences", services.GeofenceHandler.Create)
    api.Get("/ws", services.WebSocketHandler.Handle)
}
```

**Step 2: Write tests**

```go
func TestPositionHandler_List(t *testing.T) {
    // Integration test with Fiber test client
}
```

**Step 3: Commit**

```bash
git add cmd/server/ internal/api/
git commit -m "feat: implement REST API"
```

---

## Phase 7: Integration & Testing

### Task 13: End-to-end protocol test

**Files:**
- Create: `tests/integration/teltonika_test.go`
- Create: `tests/integration/gt06_test.go`

**Step 1: Start protocol server with testcontainers**

```go
func TestTeltonikaIntegration(t *testing.T) {
    // Start PostgreSQL + Redis in testcontainers
    // Start protocol server
    // Connect mock Teltonika device
    // Verify position stored in database
    // Verify WebSocket broadcast received
}
```

**Step 2: Commit**

```bash
git add tests/
git commit -m "test: add end-to-end integration tests"
```

---

## Phase 8: Production Readiness

### Task 14: Docker and deployment config

**Files:**
- Create: `Dockerfile`
- Create: `docker-compose.yml`
- Create: `deployments/docker/docker-compose.prod.yml`

**Step 1: Write Dockerfile**

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
COPY deployments/docker/entrypoint.sh .
EXPOSE 5023 5027 5020 8080
CMD ["./server"]
```

**Step 2: Write docker-compose**

```yaml
services:
  dukungps:
    build: .
    ports:
      - "5023:5023"
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://dukungps:password@postgres:5432/dukungps
      - REDIS_URL=redis://redis:6379
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:16
    environment:
      - POSTGRES_DB=dukungps
      - POSTGRES_USER=dukungps
      - POSTGRES_PASSWORD=password
    volumes:
      - postgres-data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - redis-data:/data
```

**Step 3: Commit**

```bash
git add Dockerfile docker-compose.yml deployments/
git commit -m "chore: add Docker and deployment config"
```

---

## Phase 9: Documentation

### Task 15: Write documentation

**Files:**
- Create: `docs/architecture.md`
- Create: `docs/api.md`
- Create: `docs/protocols.md`
- Update: `README.md`

**Step 1: Write architecture docs**

```markdown
# Architecture

## System Overview
[Explain Clean Architecture layers]

## Protocol Layer
[Explain how protocols plug into the system]

## Data Flow
[Explain how position data flows from device to database]
```

**Step 2: Write API docs**

```markdown
# REST API

## GET /api/v1/positions
[Document parameters and response]
```

**Step 3: Commit**

```bash
git add docs/
git commit -m "docs: add architecture and API documentation"
```

---

## Summary

**Total Tasks:** 15
**Estimated Timeline:** 8-12 weeks (assuming 2-3 tasks per week)

**Key Milestones:**
1. Week 1-2: Project foundation (Tasks 1-2)
2. Week 3-4: Protocol layer (Tasks 3-6)
3. Week 5-6: Infrastructure layer (Tasks 7-9)
4. Week 7: Use case + Protocol server (Tasks 10-11)
5. Week 8: HTTP API (Task 12)
6. Week 9: Integration testing (Task 13)
7. Week 10: Production readiness (Tasks 14-15)

**Next Steps After Plan:**
1. Initialize git worktree for isolation
2. Execute plan task-by-task using subagent-driven development
3. Regular commits to main branch
4. Push to origin/main when ready
