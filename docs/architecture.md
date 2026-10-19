# Architecture & System Design

## 1. System Overview

`dukungps-core` adalah high-performance GPS tracking server berbasis Go yang mengadopsi **Clean Architecture** dan pola **Open-Core**. Desain pipeline pemrosesan posisi dan standardisasi telemetri mengacu pada arsitektur **Traccar** (referensi standar industri open-source untuk pelacak GPS), namun diimplementasikan secara native di Go dengan pemisahan dependensi yang ketat, konkurensi goroutine ringan, dan multi-level in-memory cache.

```
+----------------+     +---------------------+     +-----------------------+
| GPS Device     | --> | Protocol TCP Server | --> | Protocol Decoder      |
| (Teltonika,    |     | (Frame Accumulator  |     | (Teltonika, GT06,     |
|  GT06, Huabao) |     |  & Connection Sess) |     |  Huabao Decoders)     |
+----------------+     +---------------------+     +-----------+-----------+
                                                               |
                                                               v
+----------------+     +---------------------------------------+-----------+
| WebSocket Hub  | <-- | PositionService Pipeline                          |
| (Real-time WS) |     | 1. Pre-commit Filters:                            |
+----------------+     |    - DriftFilter (Jitter suppressor saat diam)    |
                       |    - MaxSpeedFilter (Teleportation/Spike blocker) |
+----------------+     |    - TelemetryNormalizer (Virtual ACC fallback)   |
| Open-Core      | <-- |    - DistanceAndOdometerEnricher (Haversine)      |
| Plugin Hooks   |     |    - TripAndStateDetector (Moving/Idling/Stopped) |
| (OnPosition)   |     | 2. Multi-Level Cache Storage (L1 Ristretto + L2)  |
+----------------+     | 3. Database Persistence (PostgreSQL / pgx)        |
                       +---------------------------------------------------+
```

---

## 2. Layer Structure

Struktur codebase terbagi dalam layer-layer independen:

```
cmd/server/              # Dependency injection, service wiring, entrypoint
internal/
  domain/                # Entity murni tanpa dependency eksternal
    device/              # Entitas perangkat GPS & metadata
    position/            # Entitas posisi koordinat, speed, dan telemetry attributes
    geofence/            # Entitas area geofence (Polygon & Circle)
    user/                # Entitas pengguna & autentikasi
    command/             # Entitas command pengiriman ke perangkat
    alert/               # Entitas alert & notifikasi
  protocol/              # Adaptor protokol perangkat (Network & Serialization)
    server.go            # TCP server generik dengan frame accumulation & buffer session
    protocol.go          # Interface Decoder, Encoder, dan FrameDecoder
    teltonika/           # Codec 8 / 8 Extended decoder + Codec 12 command encoder
    gt06/                # Login, GPS, and status frame decoder + ACK echo
    huabao/              # Framing 0x7E JT/T-808, byte unescaping, XOR checksum
  usecase/               # Business logic core & pipeline orkestrator
    position_service.go  # Pipeline pemrosesan koordinat (filter -> persist -> cache -> broadcast -> hooks)
    positionfilter/      # Pre-commit pipeline filters (Drift, Speed, Normalizer, Odometer, Trip/State)
    device_usecase.go    # Logika manajemen dan auto-registration perangkat
    geofence_usecase.go  # Logika evaluasi dan CRUD geofence
    command_usecase.go   # Dispatch perintah ke protokol
    alert_usecase.go     # Pencatatan dan dispatch peringatan
  repository/            # Implementasi persistensi PostgreSQL menggunakan pgx
  infrastructure/        # Layanan infrastruktur teknis
    cache/               # Multi-level cache (L1 in-memory Ristretto v2 + L2 distributed Redis)
    database/            # PostgreSQL connection pool (pgxpool)
    websocket/           # Hub streaming real-time ke web clients
  api/handler/           # REST API Fiber v3 dan WebSocket connection handler
pkg/
  plugin/                # Plugin SDK contracts (Core interface, Registry, Scoped Config)
plugins/
  webhook/               # Reference open plugin untuk forwarding data koordinat
  geofencealert/         # Plugin deteksi pelanggaran geofence real-time
```

---

## 3. Position Processing Pipeline

Setiap posisi GPS yang berhasil didecode melewati pipeline bertahap sebelum disimpan permanen:

1. **Fix Validation**: Verifikasi apakah posisi memiliki status GPS fix valid (`p.IsValidFix()`). Posisi invalid (misal: mencari satelit di basement) tidak diproses ke database.
2. **L1/L2 Cache Lookup**: Mengambil posisi terakhir yang diketahui (`lastPos`) dari cache untuk dijadikan basis komparasi filter.
3. **Pre-commit Filters (`PrePositionFilter`)**:
   - **`DriftFilter`**: Menghitung jarak terhadap titik terakhir menggunakan formula Haversine. Jika kecepatan kendaraan nol (`speed == 0`) dan jarak perpindahan di bawah batas ambang jitter (default: 20 meter), titik dianggap sebagai *GPS drift* dan di-drop sebelum masuk database.
   - **`MaxSpeedFilter`**: Menolak paket dengan kecepatan abnormal (default: > 180 km/h) akibat noise satelit.
   - **`TelemetryNormalizer`**: Menstandarkan penamaan atribut hardware (`ignition`, `motion`, `battery`, `power`). Menyediakan **Virtual ACC**: jika hardware tidak mengirimkan status ACC fisik, sistem secara otomatis mengevaluasi status pergerakan (`speed > 3 km/h` atau `motion == true`) untuk menyimpulkan status mesin menyala.
   - **`DistanceAndOdometerEnricher`**: Menghitung jarak pergerakan dari titik sebelumnya (Haversine delta) dan mengumpulkan total akumulasi odometer (`total_distance`).
   - **`TripAndStateDetector`**: Mengklasifikasikan status operasional kendaraan menjadi:
     - `Moving`: Mesin menyala (`ignition=true`) dan bergerak (`speed > 3` atau `motion=true`). Akumulasi `trip_distance` dan `trip_duration`.
     - `Idling`: Mesin menyala (`ignition=true`) tetapi kendaraan berhenti. Akumulasi `idle_duration` kontinu.
     - `Stopped`: Mesin mati (`ignition=false`). Reset durasi idle dan metrik trip berjalan.
4. **Database Persistence**: Koordinat dan JSONB attributes disimpan ke tabel `positions` PostgreSQL.
5. **Multi-level Cache Update**: Posisi terbaru disimpan ke L1 Ristretto v2 dan L2 Redis cache.
6. **Real-time Broadcast**: Data dipancarkan ke WebSocket Hub untuk client frontend.
7. **Plugin Hooks (`OnPosition`)**: Trigger seluruh listener plugin yang terdaftar secara terisolasi tanpa memblokir pipeline utama.

---

## 4. Multi-Level Caching Architecture

Untuk mendukung ribuan koneksi perangkat aktif dengan latensi query seminimal mungkin:

```
+-------------------------------------------------------+
|                 Cache Interface                       |
|   GetPosition(deviceID)  |   SetPosition(deviceID, p) |
+-------------------------------------------------------+
                           |
                           v
              +--------------------------+
              |    MultilevelCache       |
              +-------------+------------+
                            |
           +----------------+----------------+
           |                                 |
           v                                 v
+-----------------------+        +-----------------------+
|  L1: In-Memory        |        |  L2: Distributed      |
|  Ristretto v2         |        |  Redis                |
|  - Ultra-low latency  |        |  - Cross-instance     |
|  - Zero GC pressure   |        |  - Persistence sync   |
+-----------------------+        +-----------------------+
```

- **L1 In-Memory**: Menggunakan Ristretto v2 (`dgraph-io/ristretto/v2`). Menyediakan retensi berbasis TinyLFU eviction policy dengan throughput jutaan operasi per detik.
- **L2 Redis**: Menyimpan snapshot posisi terakhir dalam format JSON terkompresi dengan TTL terkonfigurasi. Memungkinkan beberapa instance `dukungps-core` di belakang load balancer berbagi state status online perangkat.

---

## 5. Open-Core Plugin Subsystem

DukunGPS Core mengadopsi arsitektur open-core yang memungkinkan penambahan kapabilitas tanpa memodifikasi modul inti:

- **Isolasi Kontrak**: Plugin hanya berkomunikasi lewat interface [`pkg/plugin.Core`](../pkg/plugin).
- **Scoped Configuration**: Konfigurasi otomatis dipetakan dari environment variable dengan pola:
  `PLUGIN_<PLUGIN_NAME>_<KEY>` (contoh: `PLUGIN_WEBHOOK_URL`).
- **Feature Flags**: Setiap plugin dapat diaktifkan atau dinonaktifkan secara dinamis saat boot menggunakan env `PLUGIN_<NAME>_ENABLED=false`.
- **Sub-Router Integration**: Plugin dapat mengekspos endpoint HTTP kustom di bawah namespace `/api/v1/plugins/<plugin-name>`.

---

## 6. Traccar Architectural Mapping & Comparison

Traccar merupakan acuan standar terbuka terdepan di industri pelacakan GPS berbasis Java. Berikut adalah analisis komparasi arsitektural antara pendekatan Traccar dan implementasi `dukungps-core`:

| Aspek Arsitektur | Traccar (Java / Netty) | DukunGPS Core (Go / Clean Architecture) |
|---|---|---|
| **Networking Runtime** | Java NIO / Netty ChannelPipeline (Event-driven asynchronous). | Go netpoll / Goroutine per-koneksi TCP (Direct blocking I/O dengan thread ringan). |
| **Frame Accumulation** | Netty `ByteToMessageDecoder` (akumulasi buffer di channel context). | `protocol.Server` dengan `FrameDecoder.FrameLen` dynamic byte slicing per connection. |
| **Pipeline Architecture** | `ChannelInboundHandlerAdapter` chaining (`BaseDataHandler`, `FilterHandler`, `MotionHandler`). | `PositionService` dengan `PrePositionFilter` functions slice + `PositionHook` execution. |
| **Filtering Mechanism** | `FilterHandler`: Konfigurasi statis filter (distance, speed, zero coordinate, duplicate). | `positionfilter` modular: `DriftFilter`, `MaxSpeedFilter`, `TelemetryNormalizer`, `TripAndStateDetector`. |
| **State & Trip Detection** | `MotionHandler` berbasis logic threshold ignition & distance. | `TripAndStateDetector` pre-filter yang menghasilkan state `Moving`, `Idling`, `Stopped` + akumulasi durasi kontinu. |
| **Memory & Caching** | Java Heap In-Memory Object Cache (raw ConcurrentHashMap / Ehcache). | Multi-level cache: L1 Ristretto v2 (Zero GC overhead) + L2 Redis terdistribusi. |
| **Extensibility Model** | Modifikasi source code langsung atau JAR packaging berbasis Java SPI. | Open-Core SDK (`pkg/plugin`) dengan hook runtime terisolasi dan scoped sub-routing API. |
| **Protocol Framing** | Menggunakan implementasi decoder ratusan protokol di package `org.traccar.protocol`. | Mengadopsi byte-spec presisi Traccar untuk Big-Endian decode & ACK write-back pada Teltonika, GT06, dan Huabao. |
| **Speed & Units** | Standarisasi internal disimpan dalam satuan **knots** (km/h / 1.852). | Mengikuti Traccar: Kecepatan dinormalisasi dan disimpan dalam satuan **knots** di database. |
| **Resource Footprint** | 500MB - 2GB RAM (JVM overhead). | 20MB - 80MB RAM (Binary Go native, optimal untuk VPS hemat resource / Coolify). |

### Keuntungan Pendekatan DukunGPS

1. **Efisiensi Memori & Startup Cepat**: DukunGPS menggunakan compiled binary native Go dengan footprint memori sangat kecil dibandingkan JVM runtime, sangat ideal untuk deployment modern di Docker/Nixpacks Coolify.
2. **Keterbacaan Pipeline**: Pipeline pemrosesan posisi diimplementasikan sebagai closure `PrePositionFilter` yang transparan dan mudah diuji dengan unit test independen tanpa memerlukan mock Netty pipeline yang kompleks.
3. **Multi-Tenant / Open-Core Ready**: Pemisahan layer plugin yang ketat memungkinkan organisasi membangun modul komersial tertutup (misal: analisis sensor BBM, billing armada, dispatch alert WhatsApp) di atas fondasi core open-source yang stabil.

