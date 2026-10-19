# REST API & WebSocket Specification

Base URL: `/api/v1` (default port 8080, konfigurasi via environment variable `HTTP_ADDR`).

Semua respons menggunakan format standar JSON.

---

## 1. System Health

### GET `/api/v1/health`
Health check untuk readiness probe Docker / Coolify.

**Response (200 OK):**
```json
{
  "status": "ok"
}
```

---

## 2. Devices

### GET `/api/v1/devices`
Menampilkan daftar semua perangkat yang terdaftar.

**Response (200 OK):**
```json
[
  {
    "id": 1,
    "unique_id": "123456789012345",
    "name": "Truk Ekspedisi 01",
    "model": "FMB920",
    "protocol": "teltonika",
    "status": "online",
    "last_position_id": 1042,
    "created_at": "2026-08-03T08:32:25Z",
    "updated_at": "2026-08-03T08:32:25Z"
  }
]
```

### GET `/api/v1/devices/:id`
Detail satu perangkat berdasarkan numeric ID.

### POST `/api/v1/devices`
Mendaftarkan perangkat baru secara manual.

**Request Body:**
```json
{
  "uniqueId": "123456789012345",
  "name": "Truk Ekspedisi 01",
  "model": "FMB920",
  "protocol": "teltonika"
}
```

> **Catatan:** Perangkat juga dapat terdaftar secara otomatis saat pertama kali mengirimkan data koordinat lewat TCP jika `AUTO_REGISTER_DEVICES=true`.

---

## 3. Positions

### GET `/api/v1/devices/:deviceId/positions?limit=100`
Mengambil histori posisi perangkat diurutkan dari yang terbaru (`time DESC`).

**Query Parameters:**
- `limit` (int, default: 100): Jumlah maksimal record yang diambil.

**Response (200 OK):**
```json
[
  {
    "id": 1042,
    "device_id": 1,
    "latitude": -7.2456,
    "longitude": 112.7389,
    "altitude": 15.0,
    "speed": 22.68,
    "course": 180.0,
    "valid": true,
    "time": "2026-10-19T03:20:00Z",
    "device_time": "2026-10-19T03:20:00Z",
    "attributes": {
      "ignition": true,
      "motion": true,
      "status": "moving",
      "distance": 84.5,
      "total_distance": 12540.2,
      "trip_distance": 1420.5,
      "trip_duration": 180,
      "idle_duration": 0,
      "power": 12.6,
      "battery": 4.1
    }
  }
]
```

### GET `/api/v1/devices/:deviceId/positions/search`
Pencarian posisi historis dengan filter rentang waktu dan kecepatan.

**Query Parameters:**
- `from` (string, ISO8601): Waktu awal
- `to` (string, ISO8601): Waktu akhir
- `min_speed` (float): Filter kecepatan minimum (knots)
- `max_speed` (float): Filter kecepatan maksimum (knots)
- `limit` (int): Maksimum record

---

## 4. Device Groups

### GET `/api/v1/device-groups`
Mendapatkan daftar grup armada perangkat.

### GET `/api/v1/device-groups/:id`
Detail satu grup perangkat.

### POST `/api/v1/device-groups`
Membuat grup armada baru.

**Request Body:**
```json
{
  "name": "Armada Wilayah Timur",
  "description": "Kendaraan operasional rute Surabaya - Banyuwangi"
}
```

### PUT `/api/v1/device-groups/:id`
Memperbarui informasi grup.

### DELETE `/api/v1/device-groups/:id`
Menghapus grup.

### GET `/api/v1/device-groups/:groupId/devices`
Menampilkan daftar ID perangkat anggota dalam grup tersebut.

### POST `/api/v1/device-groups/:groupId/devices`
Menambahkan perangkat ke dalam grup.

**Request Body:**
```json
{
  "deviceId": 1
}
```

### DELETE `/api/v1/device-groups/:groupId/devices/:deviceId`
Mengeluarkan perangkat dari grup.

---

## 5. Geofences

### GET `/api/v1/geofences`
Daftar seluruh area geofence (Polygon / Circle).

### GET `/api/v1/geofences/:id`
Detail area geofence.

### POST `/api/v1/geofences`
Membuat geofence baru.

**Request Body (Polygon):**
```json
{
  "name": "Depo Surabaya",
  "type": "polygon",
  "area": {
    "coordinates": [
      [-7.25, 112.74],
      [-7.26, 112.74],
      [-7.26, 112.75],
      [-7.25, 112.75]
    ]
  }
}
```

**Request Body (Circle):**
```json
{
  "name": "Kantor Pusat",
  "type": "circle",
  "center_lat": -7.2575,
  "center_lon": 112.7521,
  "radius": 250.0
}
```

### PUT `/api/v1/geofences/:id`
Memperbarui konfigurasi area geofence.

### DELETE `/api/v1/geofences/:id`
Menghapus geofence.

---

## 6. Commands (Perintah Perangkat)

### GET `/api/v1/commands`
Daftar command yang tersimpan di sistem.

### GET `/api/v1/commands/:id`
Detail status command.

### GET `/api/v1/devices/:deviceId/commands`
Histori command untuk perangkat tertentu.

### POST `/api/v1/commands`
Membuat record command.

### POST `/api/v1/commands/dispatch`
Mengirimkan perintah ke perangkat secara langsung via koneksi TCP aktif.

**Request Body:**
```json
{
  "deviceId": 1,
  "type": "engineStop",
  "data": ""
}
```

Mendukung command Teltonika (Codec 12):
- `engineStop` (`setdigout 1`)
- `engineResume` (`setdigout 0`)
- `custom` (perintah SMS / GPRS bebas)

### PUT `/api/v1/commands/:id`
Update metadata/status command.

### DELETE `/api/v1/commands/:id`
Hapus record command.

---

## 7. Alerts

### GET `/api/v1/alerts`
Daftar semua alert event yang tercatat.

### GET `/api/v1/alerts/:id`
Detail alert event.

### GET `/api/v1/devices/:deviceId/alerts`
Histori alert khusus perangkat tertentu.

### POST `/api/v1/alerts`
Membuat alert baru secara manual.

### POST `/api/v1/alerts/dispatch`
Memicu dispatch alert (misal: geofence exit, SOS, overspeed).

**Request Body:**
```json
{
  "deviceId": 1,
  "type": "geofenceExit",
  "title": "Perangkat Keluar Geofence",
  "description": "Truk 01 keluar dari Depo Surabaya",
  "metadata": {
    "geofenceId": 3,
    "speed": 45.2
  }
}
```

### PUT `/api/v1/alerts/:id`
Memperbarui status alert (misal: `acknowledged`, `resolved`).

### DELETE `/api/v1/alerts/:id`
Menghapus record alert.

---

## 8. Users

### GET `/api/v1/users`
Daftar pengguna terdaftar.

### GET `/api/v1/users/:id`
Detail pengguna.

### POST `/api/v1/users`
Registrasi user baru (`name`, `email`, `password`, `role`).

### PUT `/api/v1/users/:id`
Update profil user.

### DELETE `/api/v1/users/:id`
Menghapus user.

---

## 9. Plugin API Subsystems

Setiap plugin yang terdaftar memiliki isolasi sub-router di bawah:
`/api/v1/plugins/<plugin-name>`

Contoh plugin bawaan:
- **Webhook plugin**: `/api/v1/plugins/webhook`
- **Geofence Alert plugin**: `/api/v1/plugins/geofencealert`

---

## 10. WebSocket Streaming

### GET `/api/v1/ws`
Koneksi WebSocket real-time untuk broadcast update posisi kendaraan.

**Pesan Masuk (Server -> Client):**
```json
{
  "deviceId": 1,
  "position": {
    "id": 1042,
    "deviceId": 1,
    "latitude": -7.2456,
    "longitude": 112.7389,
    "speed": 22.68,
    "course": 180.0,
    "valid": true,
    "time": "2026-10-19T03:20:00Z",
    "attributes": {
      "status": "moving",
      "ignition": true,
      "motion": true,
      "distance": 84.5
    }
  }
}
```

---

## 11. Format Error Standar

Semua error response mengembalikan format JSON konsisten:
```json
{
  "error": "keterangan pesan error"
}
```

| HTTP Status | Keterangan |
|---|---|
| `400 Bad Request` | Parameter atau ID tidak valid |
| `404 Not Found` | Resource yang diminta tidak ditemukan |
| `409 Conflict` | Duplikasi data (misal: IMEI uniqueId sudah ada) |
| `422 Unprocessable` | Validasi payload JSON gagal |
| `500 Internal Error` | Terjadi kesalahan pada server / database |

