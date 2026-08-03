# Architecture

## System Overview

DukunGPS adalah GPS tracker server berbahasa Go dengan Clean Architecture. Desain mengikuti arsitektur Traccar (protocol decoder → pipeline → persistence) namun diimplementasikan native di Go dengan pemisahan layer yang ketat.

```
+----------------+     +---------------------+     +-------------+
| GPS Device     | --> | Protocol TCP Server | --> | Decoder     |
| (Teltonika,    |     | (per-protocol port) |     | (per-proto) |
|  GT06, Huabao) |     +---------------------+     +------+------+
+----------------+                                          |
                                                            v
+----------------+     +---------------------+     +--------+--------+
| WebSocket      | <-- | PositionService     | <-- | DeviceSession   |
| Hub (real-time)|     | (usecase layer)     |     | (IMEI -> device)|
+----------------+     +----------+----------+     +-----------------+
                                  |
                        +---------+---------+
                        | PostgreSQL (persist) |
                        | Redis (cache/status) |
                        +-----------------------+
```

## Layer Structure

```
cmd/server/              entrypoint: wiring semua layer + start TCP & HTTP server
internal/
  domain/                entity murni (tanpa dependency), behavior + unit test
    device/ position/ geofence/ user/ command/
  protocol/              adapter device <-> domain
    teltonika/ gt06/ huabao/   decoder + encoder per protocol
    protocol.go           interface (Protocol, Decoder, Encoder, FrameDecoder)
    server.go             TCP server: frame accumulation, session, ack
  usecase/               business logic (PositionService pipeline)
  repository/            interface + implementasi PostgreSQL (pgx)
  infrastructure/
    database/            pgxpool setup
    cache/               Redis cache (position terbaru, status online)
    websocket/           Hub real-time broadcast
  api/handler/           HTTP handler Fiber (devices, positions, ws)
tests/integration/       end-to-end: TCP device -> DB -> WebSocket
migrations/              SQL schema
deployments/docker/      entrypoint + compose produksi
```

## Protocol Layer

Setiap protocol mengimplementasikan 3 kontrak dari `internal/protocol`:

- `Decoder.Decode(data) (*DecodeResult, error)` — parse frame menjadi positions,
  plus `DeviceID` (IMEI) dan `Response` (ack bytes yang wajib dikirim balik).
- `Encoder.Encode(cmd)` — encode command (contoh: Teltonika engine stop).
- `FrameDecoder.FrameLen(data) (int, error)` — melaporkan panjang frame lengkap.
  Ini wajib karena TCP stream ter-fragmentasi; server mengumpulkan buffer sampai
  satu frame utuh sebelum men-decode.

Alur decode Teltonika codec 8 sebagai contoh: u32 prefix nol → u32 data length →
codec byte → count → per record (timestamp u64, priority, lon/lat int32/1e7,
altitude i16, course u16, satellites, speed, lalu IO parameter groups).

## Data Flow

1. Device konek ke port protocol (default: GT06 5023, Teltonika 5027, Huabao 5020).
2. Frame login/identifikasi → `DecodeResult.DeviceID` berisi IMEI.
3. Server menyimpan IMEI untuk koneksi tersebut (session per-connection).
4. Frame data → decoder menghasilkan positions, server mengirim `Response` (ack).
5. `PositionHandler` resolve IMEI → device ID (auto-register jika belum ada),
   lalu `PositionService.ProcessPosition`:
   a. validasi fix
   b. persist ke PostgreSQL
   c. update Redis cache
   d. broadcast ke WebSocket hub
6. Client web menerima update real-time via WebSocket, atau query history via REST.

## Konvensi Unit

- **Speed** disimpan dalam **knots** (konversi dari km/h mengikuti Traccar: /1.852).
- **Koordinat** dalam derajat desimal (float64).
- **Timestamp** `TIMESTAMPTZ` di PostgreSQL, `time.Time` di Go.

## Testing

- Unit test: tiap layer (domain, protocol, usecase, handler).
- Integration: PostgreSQL via testcontainers, Redis via miniredis.
- E2E (`tests/integration`): server TCP asli + mock device + verifikasi DB & WS.

Jalankan: `make test`
