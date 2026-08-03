# dukungps

> Mbah Dukun tahu kemana kamu pergi.

GPS tracker server dalam Go, Clean Architecture, dengan Traccar sebagai referensi
utama untuk byte format protocol.

## Fitur

- **Protocol support**: Teltonika (codec 8/8E), GT06, Huabao — decoder + encoder + ack
- **TCP protocol server**: frame accumulation dari stream ter-fragmentasi, session IMEI per-koneksi
- **Real-time**: WebSocket hub untuk position update
- **Persistence**: PostgreSQL (pgx) + Redis cache untuk posisi terakhir & status online
- **Auto-register device** saat pertama kali mengirim data
- **REST API** Fiber untuk devices & positions

## Struktur

```
cmd/server/          entrypoint + wiring
internal/domain/     entity murni
internal/protocol/   decoder/encoder/server per protocol
internal/usecase/    business logic
internal/repository/ PostgreSQL access
internal/api/        HTTP handler
tests/integration/   end-to-end tests
migrations/          SQL schema
```

Lihat [docs/architecture.md](docs/architecture.md) untuk detail arsitektur.

## Quick Start

### Local development

```bash
make run        # go run ./cmd/server
make test       # go test ./...
```

Frontend (Next.js):

```bash
cd dukungps-web
npm install
npm run dev    # http://localhost:3000
```

Butuh PostgreSQL + Redis lokal:

```bash
export DATABASE_URL=postgres://dukungps:dukungps@localhost:5432/dukungps
export REDIS_URL=localhost:6379
```

### Docker Compose (stack lengkap)

```bash
docker compose up -d --build
```

Service:
- HTTP API: `:8080` (override `HTTP_PORT`)
- GT06: `:5023` (override `GT06_PORT`)
- Teltonika: `:5027` (override `TELTONIKA_PORT`)
- Huabao: `:5020` (override `HUABAO_PORT`)
- Web Dashboard: `:3000` (override `WEB_PORT`)

Migration schema otomatis dijalankan oleh `entrypoint.sh` saat pertama start.

### Produksi (Coolify/Docker VPS)

```bash
docker compose -f deployments/docker/docker-compose.prod.yml up -d
```

Wajib set `DATABASE_URL` dan `POSTGRES_PASSWORD` via environment.

## Environment Variables

| Variable | Default | Keterangan |
|---|---|---|
| `DATABASE_URL` | `postgres://dukungps:dukungps@localhost:5432/dukungps` | PostgreSQL DSN |
| `REDIS_URL` | `localhost:6379` | Redis address |
| `HTTP_ADDR` | `:8080` | HTTP listen address |
| `TELTONIKA_PORT` | `5027` | Teltonika TCP port |
| `GT06_PORT` | `5023` | GT06 TCP port |
| `HUABAO_PORT` | `5020` | Huabao TCP port |
| `AUTO_REGISTER_DEVICES` | `false` | Auto-register device baru saat pertama kirim data |
| `WEB_PORT` | `3000` | Web dashboard port |

## Dokumentasi

- [Arsitektur](docs/architecture.md)
- [REST API](docs/api.md)
- [Protocol byte format](docs/protocols.md)
- [Implementation plan](docs/plans/2026-08-03-dukungps-implementation-plan.md)

## Lisensi

Lihat [LICENSE](LICENSE).
