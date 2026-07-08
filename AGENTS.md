# mnt — AGENTS.md

Go CLI for mounting/unmounting/listening block devices on Linux.

## Essentials

- **Source**: `main.go` + `go.mod` — the entire application.
- **No external deps**: only Go stdlib + system tools.
- **Build**: `go build -o mnt` produces a static binary.

## Common commands

```bash
go build -o mnt              # build binary
go vet ./...                 # static analysis
go build ./...               # verify compilation

# dry-run read-only commands — no sudo needed
./mnt list
./mnt list --all
./mnt info /dev/sdb1
./mnt status
```

## Commands

| Command | Sudo? | Description |
|---------|-------|-------------|
| `list [--all]` | No | Block devices (unmounted parts only by default) |
| `info <device>` | No | Detailed device info + optional SMART health |
| `mount [device]` | Yes | Mount device(s); interactive picker if no arg |
| `unmount <device>` | Yes | Unmount + remove mountpoint dir |
| `status` | No | Currently mounted devices |
| `help` | No | Usage text |

## Key details

- Mountpoint pattern: `<config.mount_base_dir>/<label>` (default `/media`).
- Config via `~/.config/mnt/config.json` (XDG Base Directory).
- Runs `ntfsfix` before NTFS mounts and `fsck -y` before vfat/exFAT.
- `mount` (interactive) and `mount all` mount sequentially; one failure doesn't stop the rest.
- `unmount` failure shows `fuser -vm <mountpoint>` to identify blockers.
- Supports: `ntfs-3g`, `exfat`, `vfat`/`fat`, `ext{2,3,4}`, `btrfs`, `xfs`.
- `list`/`mount` interactive exclude swap/crypto/LVM/luks partitions.
- Uses `lsblk -J` (JSON) for device parsing.
