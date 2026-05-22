# Howe Roadmap

> A living document for widgets and features that extend Howe's usefulness on Debian (and Linux) systems. PRs welcome.

---

## Quick Wins (high value, small scope)

### 1. `network-interfaces` widget ✅
Show link state, IP addresses, and RX/TX counters for network interfaces, with an optional regex filter.

**Status**: Implemented in `widgets/handlers/network-interfaces/`.

**Config:**
```yaml
  - type: network-interfaces
    include:
      - "^eth"
      - "^enp"
      - "^wlan"
    show_ips: true   # default: true
    show_mac: false  # default: false
```

**Output:**
```
Network:
    eth0:    up  10.0.0.42/24
    wlan0:   down
```

**Implementation notes:**
- Uses `net.Interfaces()` (stdlib) for enumeration and IPs.
- Reads `/sys/class/net/<iface>/operstate` on Linux; falls back to `FlagUp` on other platforms.
- Skips `lo` by default unless explicitly matched in `include`.

**Proposed config:**
```yaml
  - type: network-interfaces
    include:
      - "^eth"
      - "^enp"
      - "^wlan"
    # or shorthand: include: ["^eth", "^enp"]
    show_ips: true        # default: true
    show_counters: false  # default: false
```

**Suggested output:**
```
Network:
    eth0:     up  10.0.0.42/24
    wlan0:    down
```

**Implementation notes:**
- Read from `/sys/class/net/<iface>/operstate`, `/sys/class/net/<iface>/address`
- Use `rtnetlink` (Go `golang.org/x/sys/unix` or `vishvananda/netlink`) for IPs to avoid shelling out.
- Skip `lo` unless explicitly matched.

---

### 2. `usb-devices` widget ✅
List connected USB devices filtered by vendor ID, product ID, vendor name regex, or product name regex.

**Status**: Implemented in `widgets/handlers/usb-devices/`.

**Config:**
```yaml
  - type: usb-devices
    vendor_id: "046d"
    product_id: "c52b"
    vendor_name: "Logitech"
    product_name: "Mouse"
```

**Output:**
```
USB:
    Logitech USB Receiver (046d:c52b)  @ 1-2
```

**Implementation notes:**
- Pure sysfs parsing (`/sys/bus/usb/devices/*/idVendor`, `idProduct`, `manufacturer`, `product`).
- No CGO/libusb dependency.
- Skips interface entries (`1-1:1.0`) and lists only device entries.
- `vendor_name` matches the `manufacturer` string; `product_name` matches the `product` string.
- Falls back to `USB Device <vid>:<pid>` when string descriptors are unavailable.
- Returns empty output on non-Linux platforms.

**Proposed config:**
```yaml
  - type: usb-devices
    vendor_id: "046d"          # hex, optional
    product_id: "c52b"         # hex, optional
    vendor_name: "Logitech"    # regex, optional
    product_name: "Mouse"      # regex, optional
```

**Suggested output:**
```
USB:
    Logitech USB Receiver (046d:c52b)  @ usb1-2
```

**Implementation notes:**
- Parse `/sys/bus/usb/devices/*/uevent` or `/sys/kernel/debug/usb/devices` (root)
- `github.com/google/gousb` is an option but adds a CGO dependency (`libusb`); pure-go parsing of sysfs is preferable for a static MOTD binary.

---

### 3. `docker` widget enhancements — health & compose services
The current `docker` widget already shows container status. Two enhancements are useful:

#### 3a. Surface explicit health check status
When a container has a health check, distinguish:
- `healthy`
- `unhealthy`
- `starting`
- `no healthcheck`

**Suggested output change:**
```
Docker:
    plex:    Running, Up 2 hours (healthy)
    db:      Running, Up 5 days (unhealthy)
```
(The current code already includes status string from Docker; verify it captures `Health.Status` if available.)

#### 3b. `docker-services` (or extend `docker` with `services` key)
For Docker Swarm or docker-compose projects, show service state and replica counts.

**Proposed config:**
```yaml
  - type: docker-services
    project: "myapp"          # filter by com.docker.compose.project label
    services:
      - regexp:api-
      - worker
```

**Suggested output:**
```
Docker Services:
    api-web:     3/3 replicas  healthy
    api-worker:  2/2 replicas  healthy
    db:          1/1 replica   unhealthy
```

**Implementation notes:**
- Swarm: `client.ServiceList` + `client.TaskList`
- Compose (standalone): list containers by label `com.docker.compose.project` and `com.docker.compose.service`, then aggregate.

---

### 4. `memory` widget
Show RAM and swap usage with a bar, similar to `disks`.

**Proposed config:**
```yaml
  - type: memory
    show_swap: true   # default: true
```

**Suggested output:**
```
Memory:
    RAM  16.0G  8.2G  7.8G   51%
    [========================================          ]
    Swap  2.0G  0.2G  1.8G   10%
    [====                                              ]
```

**Implementation notes:**
- Use `/proc/meminfo` (pure go) or `gosigar` if it already exposes this.

---

### 5. `cpu` widget
Show per-core or aggregate CPU usage, temperature if available, and load average (could supersede the existing `load` widget or complement it).

**Proposed config:**
```yaml
  - type: cpu
    per_core: false
    show_temp: true
```

**Suggested output:**
```
CPU:
    Usage: 12%  Temp: 42°C  Load: 0.45, 0.38, 0.30
```

**Implementation notes:**
- Usage: read `/proc/stat` twice (or once and use a cache file with a short TTL if MOTD runs frequently).
- Temperature: glob `/sys/class/thermal/thermal_zone*/temp` and pick the highest, or read `/sys/class/hwmon/hwmon*/temp1_input`.

---

## Medium Effort

### 6. `zfs` widget
Show ZFS pool health, allocated/available space, and any scrub errors.

**Proposed config:**
```yaml
  - type: zfs
    pools:
      - "tank"
      - "*"
```

**Suggested output:**
```
ZFS:
    tank:  ONLINE  4.2T / 8.0T  52%
```

**Implementation notes:**
- Parse `zpool list -H -o name,size,allocated,free,capacity,health`.
- Optional: parse `zpool status` for scrub/resilver progress (more verbose, maybe behind `show_scrub: true`).

---

### 7. `raid` widget (`mdadm`)
Show software RAID array sync status and health.

**Proposed config:**
```yaml
  - type: raid
```

**Suggested output:**
```
RAID:
    md0:  active raid1  sda1[0] sdb1[1]  931G  healthy
```

**Implementation notes:**
- Read `/proc/mdstat`.

---

### 8. `last-login` widget
Show the last few successful logins and/or recent failed attempts (via `lastb`).

**Proposed config:**
```yaml
  - type: last-login
    count: 3
    show_failed: true
```

**Suggested output:**
```
Last logins:
    alice    192.168.1.42    Fri May 22 09:14   still logged in
    bob      10.0.0.5        Thu May 21 18:32 - 19:10  (00:38)

Failed attempts:
    root     203.0.113.7     Fri May 22 03:12
```

**Implementation notes:**
- `last -n 3` and `lastb -n 3` (may require group `utmp`/`btmp` permissions; degrade gracefully).

---

### 9. `smart` widget
Show SMART health for specified block devices.

**Proposed config:**
```yaml
  - type: smart
    devices:
      - /dev/sda
      - /dev/nvme0
```

**Suggested output:**
```
SMART:
    /dev/sda:    PASSED  42°C  23481h POH
    /dev/nvme0:  PASSED  38°C   5127h POH
```

**Implementation notes:**
- Shell out to `smartctl -H -A` if available; skip widget if binary missing.
- Parse `PASSED`/`FAILED` and temperature.

---

### 10. `certificates` widget
Show TLS certificate expiry for local files or remote ports (useful for servers with self-hosted services).

**Proposed config:**
```yaml
  - type: certificates
    files:
      - /etc/letsencrypt/live/example.com/fullchain.pem
    ports:
      - "localhost:8443"
```

**Suggested output:**
```
Certificates:
    example.com        valid 42 days
    localhost:8443     valid 328 days
```

**Implementation notes:**
- Use `crypto/x509` to parse PEM files or dial TLS and inspect the peer certificate.
- Warn (yellow) if < 30 days, critical (red) if expired or < 7 days.

---

### 11. `vpn` widget (`wireguard`, `tailscale`)
Show VPN interface state and peer connectivity.

**Proposed config:**
```yaml
  - type: vpn
    interfaces:
      - wg0
```

**Suggested output:**
```
VPN:
    wg0:  up  2 peers  latest handshake 3m ago
```

**Implementation notes:**
- WireGuard: read `/sys/class/net/wg0/operstate` and `wg show` or parse `wg` JSON output.
- Tailscale: `tailscale status --json` if the CLI is present.

---

### 12. `processes` widget
Show top CPU or memory consumers, or just a count of running/total processes.

**Proposed config:**
```yaml
  - type: processes
    mode: top-memory   # or top-cpu, count
    count: 5
```

**Suggested output:**
```
Top Memory:
    mysqld      1.2G
    python3     412M
    node        398M
```

**Implementation notes:**
- Reading `/proc/*/stat` and `/proc/*/status` is pure Go but tedious; shelling out to `ps` is acceptable for MOTD.

---

### 13. `ports` widget
Show listening ports and associated services.

**Proposed config:**
```yaml
  - type: ports
    include:
      - "22"
      - "80"
      - "443"
```

**Suggested output:**
```
Listening:
    :22     sshd
    :80     nginx
    :443    nginx
```

**Implementation notes:**
- Parse `/proc/net/tcp` and `/proc/net/tcp6`, map inodes to processes via `/proc/*/fd/` (tedious), or shell out to `ss -ltnp` / `netstat -ltnp`.

---

## Nice to Have / Stretch Goals

### 14. `firewall` widget
Show active UFW/NFTables rules count or default policy status.

**Suggested output:**
```
Firewall (UFW):
    Status: active  12 rules
    Default: deny incoming
```

### 15. `lvm` widget
Show thin-pool usage or VG free space.

### 16. `snap` widget
Show pending snap refreshes, similar to `updates` but for snap.

### 17. `weather` widget
Show current weather if a location is configured. (Less "system info", more "daily glance", but fits MOTD.)

### 18. `public-ip` widget
Show WAN IP and optional geolocation/ASN info.

### 19. `unattended-upgrades` widget
Show the last run timestamp and whether a reboot is required after auto-upgrades.

**Suggested output:**
```
Unattended-upgrades:
    Last run: 2025-05-21 06:42  Reboot required: yes
```

### 20. `reboot-required` widget (Debian/Ubuntu specific)
A tiny widget that only appears when `/var/run/reboot-required` exists.

```yaml
  - type: reboot-required
```

**Suggested output:**
```
*** System restart required ***
```

---

## Cross-Cutting Concerns

### Refresh / cache strategy
MOTD runs on every interactive SSH login. Some widgets (CPU usage, Docker, weather, public IP) are expensive or need two samples. Consider:
- A lightweight cache file in `/run/howe/cache.json` with a per-widget TTL.
- A `--cached` CLI flag so the cron/systemd timer can pre-warm expensive widgets.

### Error handling consistency
Today widgets say "Could not read disk information" and log to syslog. Standardise:
- `helpers.SilentError(...)` — log only, show nothing in output (widget collapses).
- `helpers.Warn(...)` — yellow inline warning.
- `helpers.FatalWidget(...)` — red inline error but keep processing other widgets.

### Tests
Add table-driven tests for each new widget using fake sysfs/proc files in `testdata/`.

---

## Prioritisation Suggestion

| Priority | Widget | Why |
|----------|--------|-----|
| P0 | `network-interfaces` | User asked; common sysadmin need; pure sysfs |
| P0 | `usb-devices` | User asked; useful for headless NAS/Pi; sysfs only |
| P0 | `docker` health + services | User asked; extends existing widget |
| P1 | `memory` | Classic MOTD info; trivial with `/proc/meminfo` |
| P1 | `reboot-required` | One-file check; huge UX win on Debian |
| P1 | `certificates` | Great for homelab servers; pure Go (`crypto/x509`) |
| P2 | `zfs`, `raid`, `smart` | Storage/sysadmin oriented; requires parsing tools |
| P2 | `vpn`, `last-login` | Security/visibility oriented |
| P3 | `weather`, `public-ip` | Fun; needs external calls |

---

## Contributing a Widget

1. Create a new package under `widgets/handlers/<widget-name>/`.
2. Implement `func handle(ctx context.Context, payload map[string]any, output chan any, wait *sync.WaitGroup)`.
3. Call `widgets.Register("<widget-name>", handle)` in `init()`.
4. Add a sample to `support/config.default.yml` (commented out).
5. Update `README.md` with the new widget documentation.
6. Update this ROADMAP: move the widget to a "Done" section and link the PR.
