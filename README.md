# mnt

CLI en Go para gestionar dispositivos de bloque en Linux (montar, desmontar, listar e inspeccionar).

## Requisitos

- Go 1.21+
- `sudo`, `lsblk`, `blkid`, `mount`, `umount` (util-linux)
- Opcional: `ntfs-3g` (NTFS), `exfatprogs` (exFAT), `smartctl` (SMART)

## Instalacion

```bash
# Desde el release (recomendado)
curl -sSL https://github.com/rblez/mnt/releases/latest/download/mnt-linux-amd64.tar.gz | tar xz -C /usr/local/bin mnt

# O via go install
go install github.com/rblez/mnt@latest

# O desde el repositorio
git clone https://github.com/rblez/mnt.git && cd mnt
go build -o /usr/local/bin/mnt
```

## Uso

```
mnt <comando> [opciones]
```

## Comandos

| Comando | Sudo | Descripcion |
|---------|------|-------------|
| `list [--all]` | No | Lista dispositivos (solo no montados por defecto) |
| `info <dispositivo>` | No | Informacion detallada del dispositivo |
| `mount [dispositivo]` | Si | Monta dispositivo(s). Sin argumento, modo interactivo |
| `unmount <dispositivo>` | Si | Desmonta un dispositivo |
| `status` | No | Muestra los dispositivos montados actualmente |
| `help` | No | Muestra la ayuda |

## Configuracion (dotfiles)

```bash
mkdir -p ~/.config/mnt
```

```json
// ~/.config/mnt/config.json
{
  "mount_base_dir": "/media"
}
```

## Ejemplos

```bash
./mnt list
./mnt list --all
./mnt info /dev/sdb1
./mnt mount /dev/sdb1
./mnt mount            # modo interactivo
./mnt unmount /dev/sdb1
./mnt status
```
