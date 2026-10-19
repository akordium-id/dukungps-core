# Protocols & Telemetry Specification

DukunGPS mendukung protokol GPS tracker multi-vendor. Semua spesifikasi byte format, parsing Big-Endian, dan skema atribut mengacu ke implementasi **Traccar** sebagai referensi industri terbuka dan diverifikasi secara ketat melalui unit test fixture.

---

## 1. Konvensi Umum

- **Byte Order**: Seluruh integer multi-byte menggunakan format **big-endian**.
- **Frame Accumulation**: Setiap decoder protokol mengimplementasikan `FrameDecoder.FrameLen` agar server TCP mampu memisahkan frame utuh dari stream TCP yang ter-fragmentasi sebelum proses decoding dimulai.
- **ACK Handshake**: Method `Decoder.Decode` mengembalikan slice byte `Response` (ack) yang wajib segera dikirim balik ke koneksi socket perangkat untuk menjaga sesi tetap terbuka.
- **Speed Unit**: Kecepatan dinormalisasi dan disimpan dalam satuan **knots** (mengikuti konvensi Traccar: `km/h / 1.852`).
- **Koordinat**: Derajat desimal disimpan dalam presisi `float64`.

---

## 2. Standardisasi Atribut Telemetri (Traccar-Compatible)

Untuk memastikan interoperabilitas dashboard frontend dan plugin terlepas dari tipe perangkat keras yang digunakan, seluruh decoder memetakan IO/telemetri mentah ke penamaan atribut standar:

| Atribut Standar | Tipe Data | Keterangan & Sumber Pemetaan |
|---|---|---|
| `ignition` | boolean | Status kunci kontak/ACC. Nilai `true` jika mesin aktif. |
| `motion` | boolean | Status pergerakan fisik perangkat (accelerometer / sensor getar). |
| `status` | string | Status operasional kendaraan hasil analisis pipeline: `"moving"`, `"idling"`, atau `"stopped"`. |
| `power` | float64 | Tegangan aki / power supply utama kendaraan (dalam Volt). |
| `battery` | float64 | Tegangan baterai cadangan internal perangkat (dalam Volt). |
| `battery_level` | int / float64 | Persentase sisa baterai internal (0 - 100%). |
| `alarm` | string | Jenis peringatan (misal: `"sos"`, `"overspeed"`, `"geofenceExit"`). |
| `distance` | float64 | Jarak tempuh dari titik sebelumnya dalam meter (Haversine delta). |
| `total_distance` | float64 | Total odometer kumulatif dalam meter. |
| `trip_distance` | float64 | Jarak tempuh akumulatif pada perjalanan (`moving`) saat ini. |
| `trip_duration` | int64 | Durasi perjalanan saat ini dalam detik. |
| `idle_duration` | int64 | Durasi idling kontinu kendaraan dalam detik. |
| `blocked` | boolean | Status pemutus relay bahan bakar (engine cut / immobilization). |
| `charge` | boolean | Status pengisian daya eksternal aktif. |

### Fallback Virtual ACC (TelemetryNormalizer)

Banyak instalasi pelacak tipe ringkas (OBD-II atau 2-wire) tidak menghubungkan kabel fisik ACC ke starter mobil. Pipeline `TelemetryNormalizer` menyediakan fallback Virtual ACC otomatis:
1. Jika atribut `ignition` tidak dikirim oleh perangkat:
   - Apabila kecepatan `speed > 3.0 km/h` atau sensor `motion == true`, sistem menetapkan `ignition = true`.
   - Jika kendaraan diam, sistem mempertahankan status `ignition` dari titik sebelumnya (cached state).

---

## 3. Spesifikasi Protokol

### Teltonika (Port Default: 5027)

Mendukung famili tracker Teltonika (FMB920, FMC130, FMB120, dll.) dengan **Codec 8** dan **Codec 8 Extended**.

#### Identifikasi / Login
```
[u16 imei_length][imei ascii]
```
- Server membalas ACK: `[u8 0x01]` (Accept).

#### Data Frame (Codec 8 & 8 Extended)
```
[u32 0x00000000][u32 data_length][codec 0x08 / 0x8E][count u8 / u16][records...][count u8 / u16][u32 crc16]
```

Struktur Record:
- **Timestamp**: 8-byte uint64 (milliseconds unix epoch).
- **Priority**: 1 byte.
- **Longitude & Latitude**: Masing-masing 4-byte int32 dibagi $10^7$.
- **Altitude**: 2-byte int16 (meter).
- **Course**: 2-byte uint16 (derajat).
- **Satellites**: 1-byte uint8.
- **Speed**: 2-byte uint16 km/h (dikonversi ke knots).
- **IO Elements**: Grouping 1-byte, 2-byte, 4-byte, dan 8-byte properties.

Pemetaan AVL ID Teltonika:
- `ID 66`: External Power Voltage ($V$) -> `power`
- `ID 67`: Internal Battery Voltage ($V$) -> `battery`
- `ID 113`: Battery Level (%) -> `battery_level`
- `ID 239`: Digital Input 1 (Ignition) -> `ignition`
- `ID 240`: Movement Sensor -> `motion`
- `ID 16`: Total Odometer -> `total_distance`

Server ACK: Mengembalikan `[u32 count]` jumlah data yang berhasil diterima.

#### Command Encoder (Codec 12)
Format: `[u32 0][u32 len+8][0x0C][0x01][0x05][u32 len][ascii command][0x01][u32 crc16-IBM]`
- Engine Cut: `setdigout 1`
- Engine Resume: `setdigout 0`

---

### GT06 (Port Default: 5023)

Mendukung protokol Concox / Jimi IoT famili GT06N, GT06E, WeTrack2, dll.

#### Struktur Frame
```
[0x78 0x78][LEN u8][TYPE][payload][SERIAL u16][CRC16-X25 u16][0x0D 0x0A]
```
Extended framing diawali header `[0x79 0x79]` dengan field `LEN` sepanjang 2 byte (u16).

#### Login Frame (TYPE 0x01)
- Payload: 8 byte BCD/Hex IMEI (15 digit).
- Response ACK: `[0x78 0x78 0x05 0x01][SERIAL u16][CRC16][0x0D 0x0A]`

#### Location Frame (TYPE 0x12, 0x10, 0x11, 0x16)
- Section GPS (18 byte): Date/Time, satelit, latitude/longitude (dibagi $1800000$), speed (km/h), dan bitflags orientasi.
- Response ACK: Echo serial number frame dengan checksum CRC-16/X25.

---

### Huabao / JT/T-808 (Port Default: 5020)

Mendukung protokol standar transportasi nasional Tiongkok (JT/T-808) dan tracker Huabao.

#### Struktur Framing & Unescaping
```
[0x7E][type u16][attribute u16][id 6B][index 2B][data][XOR checksum][0x7E]
```
Stream bytes lolos dari delimiter `0x7E` menggunakan aturan unescaping:
- `0x7D 0x01` -> `0x7D`
- `0x7D 0x02` -> `0x7E`

#### Location Report (TYPE 0x0200)
- Parsing bitflags status (ignition, valid fix, koordinat hemisphere).
- Koordinat Latitude & Longitude dibagi $10^6$.
- Waktu BCD format `YYMMDDHHMMSS`.
- Server ACK: `0x8001` General Platform Response.

---

## 4. Menambahkan Protokol Baru

1. Buat folder baru di bawah `internal/protocol/<nama>/`.
2. Implementasikan antarmuka `internal/protocol.Decoder` dan `internal/protocol.FrameDecoder`.
3. Daftarkan instance protokol di `cmd/server/main.go` pada `registry.Register()` dan tentukan port listen TCP default.
4. Buat unit test dengan fixture heksadesimal mentah yang diverifikasi terhadap log Traccar.

