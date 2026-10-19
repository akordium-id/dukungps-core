# Panduan Pembuatan Plugin untuk dukungps-core

`dukungps-core` mengadopsi pola arsitektur **Open-Core** (mirip dengan WordPress Core):
- **Core (Apache-2.0 / Open Source Permisif)**: Menangani penerimaan stream TCP GPS tracker, decoding protokol (Teltonika, GT06, Huabao), penyimpanan data ke database, caching redis, dan WebSocket broadcast.
- **Plugins (Open / Commercial / Closed Source)**: Modul ekstensi yang menambahkan fitur spesifik seperti alert geofence tingkat lanjut, integrasi sensor bahan bakar, webhook dispatcher, laporan armada, dll.

---

## 1. Arsitektur Plugin

Plugin berinteraksi dengan core melalui paket publik [`pkg/plugin`](../../pkg/plugin):

```
+--------------------------------------------------------------+
|                        dukungps Core                         |
|  (TCP Decoders -> PositionService Pipeline -> Storage & WS)  |
+------------------------------+-------------------------------+
                               |
                   Exposes Core API & Hooks
                               |
                               v
            +-------------------------------------+
            |        plugin.Core Interface        |
            | - OnPosition(hook)                  |
            | - Router(pluginName)                |
            | - Devices(), Positions(), Geofences()|
            | - DB(), Config(key, fallback)       |
            +------------------+------------------+
                               |
               +---------------+---------------+
               |                               |
               v                               v
    [ plugins/webhook ]             [ plugins/geofencealert ]
   (Free / OSS reference)           (Commercial / Premium)
```

---

## 2. Struktur Dasar Plugin

Setiap plugin wajib mengimplementasikan interface `plugin.Plugin`:

```go
package myplugin

import (
	"context"
	"log"

	"github.com/akordium-id/dukungps/internal/domain/position"
	"github.com/akordium-id/dukungps/pkg/plugin"
)

type MyPlugin struct {
	core plugin.Core
}

func New() *MyPlugin {
	return &MyPlugin{}
}

func (p *MyPlugin) Name() string {
	return "my-plugin" // Huruf kecil, gunakan tanda hubung
}

func (p *MyPlugin) Version() string {
	return "1.0.0"
}

func (p *MyPlugin) Init(core plugin.Core) error {
	p.core = core

	// 1. Dengarkan koordinat GPS yang masuk
	core.OnPosition(p.handlePosition)

	// 2. Daftarkan rute REST API di bawah /api/v1/plugins/my-plugin
	router := core.Router(p.Name())
	if router != nil {
		router.Get("/status", p.handleStatus)
	}

	return nil
}

func (p *MyPlugin) handlePosition(ctx context.Context, pos *position.Position) error {
	// Logika bisnis saat posisi baru diterima
	return nil
}

func (p *MyPlugin) Close() error {
	// Graceful shutdown: tutup koneksi worker / flush buffer
	return nil
}
```

---

## 3. Konfigurasi & Feature Flags

Gunakan helper `plugin.NewPluginConfig(pluginName, core)` untuk membaca environment variable secara scoped:

```go
cfg := plugin.NewPluginConfig(p.Name(), core)
apiKey := cfg.GetString("API_KEY", "")
threshold := cfg.GetInt("THRESHOLD", 50)
```

Variabel di atas otomatis terbaca dari env:
- `PLUGIN_MY_PLUGIN_API_KEY`
- `PLUGIN_MY_PLUGIN_THRESHOLD`

### Menonaktifkan Plugin Tanpa Ubah Kode
Secara default semua plugin yang di-register berstatus aktif. Untuk menonaktifkannya lewat environment Coolify/VPS:
```bash
PLUGIN_MY_PLUGIN_ENABLED=false
```

---

## 4. Cara Mendaftarkan Plugin ke Core

Daftarkan instance plugin di file entrypoint [`cmd/server/main.go`](../../cmd/server/main.go):

```go
import (
    "github.com/akordium-id/dukungps/plugins/myplugin"
)

// di dalam func main():
_ = pluginRegistry.Register(myplugin.New())
```

---

## 5. Menjalankan Pengujian Plugin

Buat test file `plugin_test.go` menggunakan mock core adapter. Jalankan test:

```bash
go test -v ./plugins/myplugin
```
