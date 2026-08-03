# Protocols

DukunGPS mendukung protocol GPS tracker berikut. Semua byte format mengacu ke
implementasi Traccar (referensi utama) dan diverifikasi dengan test fixture.

## Umum

- Multi-byte integer: **big-endian**.
- Tiap protocol mengimplementasikan `FrameDecoder.FrameLen` agar server bisa
  memisahkan frame dari TCP stream yang ter-fragmentasi.
- `Decoder.Decode` mengembalikan `Response` (ack) yang wajib ditulis balik ke device.

## Teltonika (port default 5027)

Famili FMB/FMC (codec 8 dan 8 Extended).

### Identifikasi

```
[u16 imei_length][imei ascii]
```
Response: `[u8 0x01]` (accept).

### Data frame (codec 8)

```
[u32 0x00000000][u32 data_length][codec 0x08][count u8][records...]
```

Record codec 8:

| Field | Size | Keterangan |
|---|---|---|
| timestamp | 8 | ms since epoch |
| priority | 1 | |
| longitude | 4 | int32 / 1e7 |
| latitude | 4 | int32 / 1e7 |
| altitude | 2 | int16 meter |
| course | 2 | uint16 derajat |
| satellites | 1 | valid jika != 0 |
| speed | 2 | uint16 km/h (disimpan sebagai knots) |
| event id | 1 (8E: 2) | |
| total io | 1 (8E: 2) | |
| group 1-byte | var | count + (id,value)* |
| group 2-byte | var | |
| group 4-byte | var | |
| group 8-byte | var | |

IO parameter yang dipetakan: 66 power (V), 67 battery (V), 113 batteryLevel,
181/182 pdop/hdop, 239 ignition, 240 motion, 16 odometer, 21 rssi, lainnya
disimpan sebagai `io<id>`.

Response data: `[u32 count]`.

### Command encoder (codec 12)

Frame command dibungkus: `[u32 0][u32 len+8][0x0C][0x01][0x05][u32 len][content][0x01][u32 crc16-IBM]`.
Mendukung: engineStop (`setdigout 1`), engineResume (`setdigout 0`), custom.

## GT06 (port default 5023)

Famili GT06E/GT06N.

### Frame

```
[0x78 0x78][LEN u8][TYPE][payload][SERIAL u16][CRC16-X25 u16][0x0D 0x0A]
```
Extended: `[0x79 0x79][LEN u16][...]`. LEN = type + payload + serial + crc.
CRC-16/X25 dihitung atas `frame[2 : len-4]`.

### Login (TYPE 0x01)

Payload: 8 byte → hex-encode 16 char → buang char pertama → IMEI 15 digit.

```
Response: [0x78 0x78 0x05 0x01][SERIAL u16][CRC16][0x0D 0x0A]
```

### Lokasi (TYPE 0x12, 0x10, 0x11, 0x16)

GPS section 18 byte:

| Field | Size | Keterangan |
|---|---|---|
| datetime | 6 | Y M D H M S (tahun + 2000) |
| satellites | 1 | low nibble |
| latitude | 4 | uint32 / 1800000 |
| longitude | 4 | uint32 / 1800000 |
| speed | 1 | km/h |
| flags | 2 | bit0-9 course, bit10 lat sign (1=positif), bit11 lon sign (1=negatif), bit12 valid |

Response: ack short frame echo serial.

## Huabao (port default 5020)

Famili Huabao (JT/T-808 style).

### Frame

```
[0x7E][type u16][attribute u16][id 6B][index 2B (5501/5502: 1B)][data][XOR checksum][0x7E]
```

Byte-stuffing pada stream: `0x7D 0x01` → `0x7D`, `0x7D 0x02` → `0x7E`.
Checksum = XOR seluruh byte antara dua delimiter (exclude leading 0x7E dan checksum).

### Device ID

- id berisi ASCII digit → dipakai langsung.
- id biner → `(u16 BE << 32) | u32 BE` = 14 digit + Luhn check digit → 15-digit IMEI.

### Login (0x0102) / Register (0x0100)

Response: `0x8001` ack.

### Lokasi (0x0200)

| Field | Size | Keterangan |
|---|---|---|
| alarm | 4 | bitflags |
| status | 4 | bit1 valid, bit2 lat neg, bit3 lon neg |
| latitude | 4 | uint32 * 1e-6 |
| longitude | 4 | uint32 * 1e-6 |
| altitude | 2 | int16 |
| speed | 2 | uint16 * 0.1 km/h |
| course | 2 | uint16 |
| time | 6 | BCD YYMMDDHHMMSS |

Response: `0x8001` ack.

## Menambah Protocol Baru

1. Buat folder `internal/protocol/<nama>/` dengan `decoder.go` (implement
   `Decoder` + `FrameDecoder`) dan optional `encoder.go`.
2. Definisikan konstanta nama di `internal/protocol/protocol.go`.
3. Buat `protocol.go` (struct `Protocol`) + `DefaultPort`.
4. Daftarkan di `cmd/server/main.go` registry + port.
5. Tulis unit test dengan fixture byte; verifikasi terhadap capture Traccar.
