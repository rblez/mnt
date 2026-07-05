#!/bin/bash

# mount.sh - CLI para gestionar dispositivos de bloque (montar/desmontar/listar)
# Uso: mount.sh <comando> [opciones]

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

ensure_sudo() {
    if ! sudo -v >/dev/null 2>&1; then
        echo -e "${RED}Error: Se requieren permisos sudo.${NC}" >&2
        exit 1
    fi
}

# --- LIST ---
cmd_list() {
    local show_all="${1:-false}"
    if [ "$show_all" = "true" ]; then
        echo -e "${BOLD}Dispositivos de bloque detectados:${NC}"
        echo ""
        lsblk -l -o NAME,SIZE,TYPE,FSTYPE,MOUNTPOINT,LABEL,MODEL | \
            awk 'NR==1 || /^(sd|nvme|mmcblk|vd|hd|loop)/'
    else
        echo -e "${BOLD}Dispositivos no montados:${NC}"
        echo ""
        local header
        header=$(printf "%-10s %-8s %-8s %-20s" "DISPOSITIVO" "TAMANO" "FS" "ETIQUETA")
        echo -e "${CYAN}${header}${NC}"
        echo "----------------------------------------------"
        lsblk -l -n -o NAME,SIZE,FSTYPE,MOUNTPOINT,LABEL,TYPE | awk '
            $3 != "" && $4 == "" && $6 == "part" && $3 !~ /swap|crypto|LVM|lvm|luks/ {
                printf "%-10s %-8s %-8s %-20s\n", "/dev/"$1, $2, $3, ($5==""?"(sin etiqueta)":$5)
            }
        '
    fi
}

# --- INFO ---
cmd_info() {
    local dev="$1"
    if [ ! -b "$dev" ]; then
        echo -e "${RED}Error: $dev no es un dispositivo de bloque valido.${NC}" >&2
        exit 1
    fi
    echo -e "${BOLD}Informacion de $dev${NC}"
    echo ""

    local model size fstype label uuid mountpoint
    model=$(lsblk -n -o MODEL "$dev" 2>/dev/null || echo "N/A")
    size=$(lsblk -n -o SIZE "$dev")
    fstype=$(blkid -s TYPE -o value "$dev" 2>/dev/null || echo "desconocido")
    label=$(blkid -s LABEL -o value "$dev" 2>/dev/null || echo "(sin etiqueta)")
    uuid=$(blkid -s UUID -o value "$dev" 2>/dev/null || echo "N/A")
    mountpoint=$(lsblk -n -o MOUNTPOINT "$dev" 2>/dev/null || echo "")

    printf "  %-20s %s\n" "Modelo:" "$model"
    printf "  %-20s %s\n" "Dispositivo:" "$dev"
    printf "  %-20s %s\n" "Tamano:" "$size"
    printf "  %-20s %s\n" "Sistema archivos:" "$fstype"
    printf "  %-20s %s\n" "Etiqueta:" "$label"
    printf "  %-20s %s\n" "UUID:" "$uuid"
    printf "  %-20s %s\n" "Punto montaje:" "${mountpoint:-"(no montado)"}"

    # Smartmontools si estan disponibles
    if command -v smartctl &>/dev/null && [[ "$dev" =~ ^/dev/sd ]]; then
        local health=$(sudo smartctl -H "$dev" 2>/dev/null | grep "SMART overall-health" | awk -F': ' '{print $2}')
        if [ -n "$health" ]; then
            printf "  %-20s %s\n" "SMART Health:" "$health"
        fi
    fi
}

# --- MOUNT ---
cmd_mount() {
    ensure_sudo

    if [ -n "${1:-}" ]; then
        mount_device "$1"
        return
    fi

    echo -e "${YELLOW}Buscando dispositivos no montados...${NC}"
    local devices
    devices=$(get_unmounted_devices)

    if [ -z "$devices" ]; then
        echo -e "${GREEN}No se encontraron dispositivos no montados.${NC}"
        exit 0
    fi

    echo -e "${BOLD}Dispositivos encontrados:${NC}"
    local i=1
    local devs=()
    while IFS= read -r dev; do
        devs+=("$dev")
        local label
        label=$(blkid -s LABEL -o value "$dev" 2>/dev/null)
        echo "  $i) $dev  [${label:-(sin etiqueta)}]"
        i=$((i+1))
    done <<< "$devices"

    echo ""
    echo -e "${YELLOW}Opciones:${NC}"
    echo "  a) Montar todos"
    echo "  q) Cancelar"
    read -p "Seleccione un numero (o 'a' / 'q'): " choice

    case "$choice" in
        a|A)
            for dev in "${devs[@]}"; do
                mount_device "$dev"
            done
            ;;
        q|Q)
            echo "Cancelado."
            exit 0
            ;;
        *)
            if [[ "$choice" =~ ^[0-9]+$ ]] && [ "$choice" -ge 1 ] && [ "$choice" -le "${#devs[@]}" ]; then
                mount_device "${devs[$((choice-1))]}"
            else
                echo -e "${RED}Opcion invalida.${NC}" >&2
                exit 1
            fi
            ;;
    esac
}

mount_device() {
    local dev="$1"
    local label fstype mountpoint
    local mounted=0

    if [ ! -b "$dev" ]; then
        echo -e "${RED}Error: $dev no es un dispositivo de bloque valido.${NC}" >&2
        return 1
    fi

    fstype=$(blkid -s TYPE -o value "$dev" 2>/dev/null)
    label=$(blkid -s LABEL -o value "$dev" 2>/dev/null)

    if [ -z "$label" ]; then
        label=$(basename "$dev")
    fi
    label="${label// /_}"

    if mount | grep -q "^$dev "; then
        echo -e "${YELLOW}$dev ya esta montado en $(lsblk -n -o MOUNTPOINT "$dev").${NC}"
        return 0
    fi

    if [ -z "$fstype" ]; then
        fstype=$(lsblk -n -o FSTYPE "$dev" 2>/dev/null)
    fi
    if [ -z "$fstype" ]; then
        echo -e "${RED}No se pudo determinar el sistema de archivos de $dev.${NC}" >&2
        return 1
    fi

    echo -e "   ${YELLOW}$detectado como ${fstype}${NC}"

    case "$fstype" in
        ntfs)
            echo -e "   ${YELLOW}Verificando sistema de archivos...${NC}"
            sudo ntfsfix "$dev" >/dev/null 2>&1 || true
            ;;
        vfat|exfat)
            echo -e "   ${YELLOW}Verificando sistema de archivos...${NC}"
            sudo fsck -y "$dev" >/dev/null 2>&1 || true
            ;;
    esac

    mountpoint="/media/$label"
    sudo mkdir -p "$mountpoint"

    echo -e "   ${YELLOW}Montando en $mountpoint...${NC}"
    case "$fstype" in
        ntfs)
            sudo mount -t ntfs-3g "$dev" "$mountpoint" 2>/dev/null || \
            sudo mount -t ntfs-3g -o rw,uid=$(id -u),gid=$(id -g) "$dev" "$mountpoint" 2>/dev/null && mounted=1 || true
            ;;
        exfat)
            sudo mount -t exfat "$dev" "$mountpoint" 2>/dev/null && mounted=1 || true
            ;;
        vfat|fat)
            sudo mount -t vfat "$dev" "$mountpoint" 2>/dev/null && mounted=1 || true
            ;;
        ext4|ext3|ext2)
            sudo mount -t "$fstype" "$dev" "$mountpoint" 2>/dev/null && mounted=1 || true
            ;;
        btrfs)
            sudo mount -t btrfs "$dev" "$mountpoint" 2>/dev/null && mounted=1 || true
            ;;
        xfs)
            sudo mount -t xfs "$dev" "$mountpoint" 2>/dev/null && mounted=1 || true
            ;;
        *)
            echo -e "${RED}Sistema de archivos $fstype no soportado.${NC}" >&2
            return 1
            ;;
    esac

    if [ "$mounted" -eq 1 ]; then
        echo -e "${GREEN}$dev montado en $mountpoint${NC}"
        sudo chown -R $USER:$USER "$mountpoint" 2>/dev/null || true
        return 0
    else
        echo -e "${RED}Fallo el montaje de $dev.${NC}" >&2
        return 1
    fi
}

# --- UNMOUNT ---
cmd_unmount() {
    ensure_sudo
    local dev="$1"

    if [ ! -b "$dev" ]; then
        echo -e "${RED}Error: $dev no es un dispositivo de bloque valido.${NC}" >&2
        exit 1
    fi

    local mountpoint
    mountpoint=$(lsblk -n -o MOUNTPOINT "$dev" 2>/dev/null)
    if [ -z "$mountpoint" ]; then
        echo -e "${YELLOW}$dev no esta montado.${NC}"
        return
    fi

    echo -e "Desmontando $dev desde $mountpoint..."
    sudo umount "$dev" && {
        echo -e "${GREEN}$dev desmontado.${NC}"
        rmdir "$mountpoint" 2>/dev/null || true
    } || {
        echo -e "${RED}Error al desmontar $dev. Puede estar en uso.${NC}" >&2
        fuser -vm "$mountpoint" 2>/dev/null || true
        exit 1
    }
}

# --- STATUS ---
cmd_status() {
    echo -e "${BOLD}Dispositivos montados:${NC}"
    echo ""
    printf "%-12s %-20s %-10s %-8s\n" "DISPOSITIVO" "PUNTO MONTAJE" "TAMANO" "FS"
    echo "----------------------------------------------------------"
    lsblk -l -n -o NAME,MOUNTPOINT,SIZE,FSTYPE | awk '$2 != "" && $2 !~ "^/" {next} $2 != "" { printf "%-12s %-20s %-10s %-8s\n", "/dev/"$1, $2, $3, $4 }'
}

# --- HELP ---
cmd_help() {
    echo -e "${BOLD}mount.sh - Gestion de dispositivos de bloque${NC}"
    echo ""
    echo "Uso: mount.sh <comando> [opciones]"
    echo ""
    echo -e "${BOLD}Comandos:${NC}"
    echo "  list [--all]              Lista dispositivos (solo no montados por defecto)"
    echo "  info <dispositivo>        Muestra informacion detallada"
    echo "  mount [dispositivo]       Monta dispositivo(s) (interactivo si no se especifica)"
    echo "  unmount <dispositivo>     Desmonta un dispositivo"
    echo "  status                    Muestra dispositivos montados actualmente"
    echo "  help                      Muestra esta ayuda"
    echo ""
    echo -e "${BOLD}Ejemplos:${NC}"
    echo "  mount.sh list"
    echo "  mount.sh list --all"
    echo "  mount.sh info /dev/sdb1"
    echo "  mount.sh mount /dev/sdb1"
    echo "  mount.sh mount            (modo interactivo)"
    echo "  mount.sh unmount /dev/sdb1"
    echo "  mount.sh status"
}

get_unmounted_devices() {
    lsblk -l -n -o NAME,FSTYPE,MOUNTPOINT,TYPE | awk '
        $2 != "" && $3 == "" && $4 == "part" && $2 !~ /swap|crypto|LVM|lvm|luks/ {
            if ($1 !~ /[^a-zA-Z0-9]/) {
                print "/dev/" $1
            }
        }
    '
}

# --- MAIN ---
if [ ! -d /proc ]; then
    echo "Este script solo funciona en Linux." >&2
    exit 1
fi

if [ $# -eq 0 ]; then
    cmd_help
    exit 0
fi

COMMAND="$1"
shift

case "$COMMAND" in
    list)
        cmd_list "${1:-false}"
        ;;
    info)
        if [ $# -lt 1 ]; then
            echo -e "${RED}Uso: mount.sh info <dispositivo>${NC}" >&2
            exit 1
        fi
        cmd_info "$1"
        ;;
    mount)
        cmd_mount "${1:-}"
        ;;
    unmount|umount)
        if [ $# -lt 1 ]; then
            echo -e "${RED}Uso: mount.sh unmount <dispositivo>${NC}" >&2
            exit 1
        fi
        cmd_unmount "$1"
        ;;
    status)
        cmd_status
        ;;
    help|--help|-h)
        cmd_help
        ;;
    *)
        echo -e "${RED}Comando desconocido: $COMMAND${NC}" >&2
        echo "Ejecute 'mount.sh help' para ver los comandos disponibles." >&2
        exit 1
        ;;
esac
