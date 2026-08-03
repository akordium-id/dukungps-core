# Documentation-as-Product Pipeline

## Device/Port Matrix Maintenance

### Workflow

1. **Source of truth**: `docs/device-port-matrix.md` (single canonical file)
2. **Update trigger**: Any PR that adds/removes a device or changes a port
3. **Validation**: CI checks that matrix is consistent with `internal/` packages
4. **Publication**: Matrix auto-published to docs site on merge to main

### Matrix Structure

| Device Type | Protocol | Port | TLS | Notes |
|---|---|---|---|---|
| Teltonika | TCP | 5003 | No | Primary tracker port |
| Teltonika | TCP | 5040 | No | AVL data push |
| WebSocket | WSS | 443 | Yes | Browser/APP real-time |
| GPS102 | TCP | 9000 | No | Legacy GPS102/103 |

### Responsibilities

- **Backend team**: Update matrix when adding new protocol handlers
- **Frontend team**: Update matrix when adding new device type support
- **DevOps**: Verify port availability before merge
