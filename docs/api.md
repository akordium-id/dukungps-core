# REST API

Base URL: `/api/v1` (default port 8080, override `HTTP_ADDR`).

Semua response berupa JSON.

## Devices

### GET /devices

List semua device.

```json
[
  {
    "ID": 1,
    "UniqueID": "123456789012345",
    "Name": "Truk 01",
    "Model": "FMB920",
    "Protocol": "teltonika",
    "Status": "online",
    "CreatedAt": "2026-08-03T08:32:25Z",
    "UpdatedAt": "2026-08-03T08:32:25Z"
  }
]
```

### GET /devices/:id

Detail satu device by numeric ID.

- `200` device ditemukan
- `404` device tidak ditemukan
- `400` id tidak valid

### POST /devices

Daftarkan device baru.

```json
{ "uniqueId": "123456789012345", "name": "Truk 01", "protocol": "teltonika" }
```

- `201` created
- `422` body tidak valid (unique_id/protocol wajib)
- `409` unique_id sudah ada

> Catatan: device juga otomatis terdaftar saat pertama kali mengirim data
> (register-on-first-contact) via TCP protocol server.

## Positions

### GET /devices/:deviceId/positions?limit=100

History posisi sebuah device, diurutkan `time DESC`.

| Query | Default | Keterangan |
|---|---|---|
| `limit` | 100 | jumlah maksimum record |

```json
[
  {
    "ID": 1,
    "DeviceID": 1,
    "Latitude": -7.2456,
    "Longitude": 112.7389,
    "Speed": 42.0,
    "Course": 180.0,
    "Valid": true,
    "Time": "2026-08-03T12:00:00Z",
    "Attributes": { "ignition": true, "battery": 12.5 }
  }
]
```

## WebSocket

### GET /ws

Real-time position updates. Setiap device mengirim posisi, semua client
terkoneksi menerima:

```json
{
  "deviceId": 1,
  "position": {
    "deviceId": 1,
    "latitude": -7.2456,
    "longitude": 112.7389,
    "valid": true,
    "time": "2026-08-03T12:00:00Z"
  }
}
```

> Endpoint saat ini belum di-wire penuh (handler stub). Upgrade loop
> dijadwalkan pada iterasi berikutnya.

## Error Format

```json
{ "error": "device not found" }
```

| Status | Arti |
|---|---|
| 400 | request tidak valid |
| 404 | resource tidak ditemukan |
| 409 | konflik (duplikat) |
| 422 | body tidak lolos validasi |
| 500 | internal error |
