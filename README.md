# dukungps

> Mbah Dukun tahu kemana kamu pergi.

[![CI/CD](https://github.com/akordium-id/dukungps-core/actions/workflows/ci.yml/badge.svg)](https://github.com/akordium-id/dukungps-core/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

High-performance GPS tracking server ditulis dalam Go dengan Clean Architecture. Mengacu pada spesifikasi byte format dan standar telemetri **Traccar** sebagai referensi industri terbuka untuk decoding protokol, normalisasi data, dan pipeline pemrosesan posisi.

## Fitur Utama

- **Multi-Protocol TCP Server**:
  - Teltonika (Codec 8, Codec 8 Extended, dan encoder command Codec 12).
  - GT06 (Login handshake, GPS & status decoding, CRC16-X25 verification, ACK echo).
  - Huabao (JT/T-808 style framing `0x7E`, unescaping, XOR checksum, BCD timestamp).
  - Frame accumulation engine untuk menangani fragmentasi TCP stream secara presisi.
- **Pre-commit Filter & Enrichment Pipeline (Traccar-style)**:
  - `DriftFilter`: Eliminasi GPS coordinate jitter saat kendaraan berhenti (`speed == 0`).
  - `MaxSpeedFilter`: Pencegahan lonjakan koordinat anomali / loncatan teleportasi.
  - `TelemetryNormalizer`: Standardisasi atribut (`ignition`, `motion`, `battery`, `power`) dengan fallback Virtual ACC otomatis.
  - `DistanceAndOdometerEnricher`: Kalkulasi jarak Haversine dan akumulasi total odometer.
  - `TripAndStateDetector`: Deteksi status operasional real-time (`Moving`, `Idling`, `Stopped`) serta tracking durasi idle dan metrik trip.
- **Multi-Level High Throughput Caching**:
  - **L1**: In-memory cache super cepat berbasis Ristretto v2 (`github.com/dgraph-io/ristretto/v2`).
  - **L2**: Distributed Redis cache untuk sinkronisasi posisi terakhir dan status online lintas instance.
- **Open-Core Plugin Architecture**:
  - Ekstensibilitas modular via kontrak [`pkg/plugin`](pkg/plugin).
  - Hook posisi non-blocking (`OnPosition`), routing sub-API scoped (`/api/v1/plugins/<name>`), isolasi konfigurasi per-plugin (`PLUGIN_<NAME>_<KEY>`), dan dynamic feature flag (`PLUGIN_<NAME>_ENABLED`).
  - Built-in reference plugins: `webhook` dan `geofencealert`.
- **Layanan Domain & REST API Lengkap**:
  - Manajemen `Devices`, `Positions` (history & search filter), `Geofences` (polygon & circle), `Users`, `Commands` (dispatch ke device), `DeviceGroups` (organisasi armada), dan `Alerts`.
- **Real-Time Streaming**:
  - WebSocket Hub real-time broadcast posisi untuk aplikasi frontend/dashboard.
- **Auto-Register Device**:
  - Opsi registrasi otomatis IMEI saat device pertama kali mengirimkan data ke TCP port.

## Struktur Project

```
cmd/server/              # Entrypoint aplikasi, dependency wiring, initialization
internal/
  api/handler/           # HTTP API Fiber handlers, middleware, routes, dan WebSocket handler
  domain/                # Entity bisnis murni tanpa dependency eksternal
    alert/ command/ device/ geofence/ position/ user/
  infrastructure/        # Implementasi teknis database, caching, dan websocket
    cache/               # Multi-level cache (L1 Ristretto v2 + L2 Redis)
    database/            # PostgreSQL connection pool (pgxpool)
    websocket/           # Real-time WebSocket Hub
  protocol/              # TCP server, frame defragmenter, dan protocol decoders/encoders
    gt06/ huabao/ teltonika/
  repository/            # Implementasi data access PostgreSQL (pgx)
  usecase/               # Business logic, services, dan pipeline filter
    positionfilter/      # Pre-commit filters (drift, speed, normalizer, odometer, trip)
pkg/
  plugin/                # SDK kontrak interface Open-Core Plugin (Core, Plugin, Config)
plugins/
  geofencealert/         # Plugin: deteksi pelanggaran geofence & pembuatan alert
  webhook/               # Plugin: pengiriman data posisi ke endpoint HTTP eksternal
migrations/              # Skema database PostgreSQL
tests/integration/       # E2E & integration test
deployments/             # Docker Compose & konfigurasi deployment
docs/                    # Dokumentasi arsitektur, API, protokol, dan plugin
```

Lihat [docs/architecture.md](docs/architecture.md) untuk detail arsitektur dan komparasi dengan Traccar.

## Quick Start

### Local Development

Prasyarat: Go 1.24+, PostgreSQL 15+, Redis 7+.

```bash
# Jalankan database & redis (atau gunakan Docker)
export DATABASE_URL="postgres://dukungps:dukungps@localhost:5432/dukungps"
export REDIS_URL="localhost:6379"

# Menjalankan server
make run        # go run ./cmd/server

# Menjalankan seluruh test suite
make test       # go test ./...
```

Frontend Dashboard (Next.js):

```bash
cd dukungps-web
npm install
npm run dev    # http://localhost:3000
```

### Docker Compose (Development Stack)

```bash
docker compose up -d --build
```

Default Service Port:
- HTTP REST API & WebSocket: `:8080` (env: `HTTP_ADDR`)
- Teltonika TCP Server: `:5027` (env: `TELTONIKA_PORT`)
- GT06 TCP Server: `:5023` (env: `GT06_PORT`)
- Huabao TCP Server: `:5020` (env: `HUABAO_PORT`)
- Web Dashboard: `:3000` (env: `WEB_PORT`)

Migrasi database SQL dijalankan otomatis oleh script container saat pertama kali dijalankan.

### Deployment Produksi (Coolify / Docker VPS)

```bash
docker compose -f deployments/docker/docker-compose.prod.yml up -d
```

Pastikan mengisi variabel `DATABASE_URL`, `POSTGRES_PASSWORD`, dan `REDIS_URL` di dashboard Coolify / environment host Anda.

## Environment Variables

| Variable | Default | Keterangan |
|---|---|---|
| `DATABASE_URL` | `postgres://dukungps:dukungps@localhost:5432/dukungps` | PostgreSQL connection string (pgxpool) |
| `REDIS_URL` | `localhost:6379` | Alamat server Redis (L2 Cache) |
| `HTTP_ADDR` | `:8080` | Host & port listen HTTP API & WebSocket |
| `TELTONIKA_PORT` | `5027` | Port listen TCP protocol Teltonika |
| `GT06_PORT` | `5023` | Port listen TCP protocol GT06 |
| `HUABAO_PORT` | `5020` | Port listen TCP protocol Huabao |
| `AUTO_REGISTER_DEVICES` | `false` | Izinkan auto-register saat device baru terdeteksi |
| `PLUGIN_<NAME>_ENABLED` | `true` | Feature flag aktif/nonaktif plugin (contoh: `PLUGIN_WEBHOOK_ENABLED=false`) |
| `PLUGIN_<NAME>_<KEY>` | - | Scoped config untuk plugin spesifik (contoh: `PLUGIN_WEBHOOK_URL=https://...`) |

## Dokumentasi Lengkap

- [Arsitektur & Komparasi Traccar](docs/architecture.md)
- [Katalog REST API & WebSocket](docs/api.md)
- [Spesifikasi Protokol & Standardisasi Telemetri](docs/protocols.md)
- [Panduan Pengembangan Plugin](docs/plugins.md)
- [Pipeline Dokumentasi Produk](docs/as-product-pipeline.md)

## Lisensi

Proyek ini dilisensikan di bawah lisensi terbuka [Apache 2.0](LICENSE).

