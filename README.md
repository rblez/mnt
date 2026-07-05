# mount.sh

CLI para gestionar dispositivos de bloque en Linux (montar, desmontar, listar e inspeccionar).

## Uso

```
mount.sh <comando> [opciones]
```

## Comandos

| Comando | Descripcion |
|---------|-------------|
| `list [--all]` | Lista dispositivos de bloque (solo no montados por defecto) |
| `info <dispositivo>` | Muestra informacion detallada del dispositivo |
| `mount [dispositivo]` | Monta dispositivo(s). Sin argumento, modo interactivo |
| `unmount <dispositivo>` | Desmonta un dispositivo |
| `status` | Muestra los dispositivos montados actualmente |
| `help` | Muestra la ayuda |

## Ejemplos

```bash
# Listar dispositivos no montados
./mount.sh list

# Ver todos los dispositivos
./mount.sh list --all

# Informacion detallada de un dispositivo
./mount.sh info /dev/sdb1

# Montar todos los dispositivos (interactivo)
./mount.sh mount

# Montar un dispositivo especifico
./mount.sh mount /dev/sdb1

# Desmontar
./mount.sh unmount /dev/sdb1

# Ver estado actual
./mount.sh status
```

## Dependencias

- `sudo` - permisos de superusuario
- `blkid`, `lsblk`, `mount`, `umount` (parte de util-linux)
- `ntfs-3g` - para sistemas de archivos NTFS
- `exfat-utils` o `exfatprogs` - para sistemas de archivos exFAT

## Instalacion

```bash
curl -sSL https://raw.githubusercontent.com/rblez/mount.sh/main/mount.sh -o /usr/local/bin/mount.sh
chmod +x /usr/local/bin/mount.sh
```
