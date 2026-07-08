package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Config struct {
	MountBaseDir string `json:"mount_base_dir"`
}

type blockDevice struct {
	Name       string  `json:"name"`
	Size       string  `json:"size"`
	Type       string  `json:"type"`
	FSType     *string `json:"fstype"`
	Mountpoint *string `json:"mountpoint"`
	Label      *string `json:"label"`
	Model      *string `json:"model,omitempty"`
}

type lsblkOutput struct {
	BlockDevices []blockDevice `json:"blockdevices"`
}

var cfg Config

func main() {
	cfg = loadConfig()

	if len(os.Args) < 2 {
		cmdHelp()
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "list":
		cmdList(args)
	case "info":
		cmdInfo(args)
	case "mount":
		cmdMount(args)
	case "unmount", "umount":
		cmdUnmount(args)
	case "status":
		cmdStatus()
	case "help", "--help", "-h":
		cmdHelp()
	default:
		fmt.Fprintf(os.Stderr, "\033[0;31mComando desconocido: %s\033[0m\n", cmd)
		fmt.Fprintln(os.Stderr, "Ejecute 'mnt help' para ver los comandos disponibles.")
		os.Exit(1)
	}
}

func loadConfig() Config {
	c := Config{MountBaseDir: "/media"}
	home, err := os.UserHomeDir()
	if err != nil {
		return c
	}
	p := filepath.Join(home, ".config", "mnt", "config.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return c
	}
	json.Unmarshal(data, &c)
	return c
}

func ensureSudo() {
	cmd := exec.Command("sudo", "-v")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "\033[0;31mError: Se requieren permisos sudo.\033[0m")
		os.Exit(1)
	}
}

func isBlockDevice(dev string) bool {
	info, err := os.Stat(dev)
	if err != nil {
		return false
	}
	m := info.Mode()
	return m&os.ModeDevice != 0 && m&os.ModeCharDevice == 0
}

func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func getBlockDevices(fields ...string) ([]blockDevice, error) {
	args := []string{"-J", "-n", "-o", strings.Join(fields, ",")}
	out, err := runCmd("lsblk", args...)
	if err != nil {
		return nil, err
	}
	var result lsblkOutput
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return nil, err
	}
	return result.BlockDevices, nil
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func isUnmountedPart(d blockDevice) bool {
	if d.Type != "part" {
		return false
	}
	if strVal(d.FSType) == "" {
		return false
	}
	if strVal(d.Mountpoint) != "" {
		return false
	}
	fs := strVal(d.FSType)
	for _, skip := range []string{"swap", "crypto", "LVM", "lvm", "luks"} {
		if fs == skip {
			return false
		}
	}
	return true
}

// --- LIST ---

func cmdList(args []string) {
	showAll := len(args) > 0 && args[0] == "--all"
	if showAll {
		fmt.Println("\033[1mDispositivos de bloque detectados:\033[0m\n")
		cmd := exec.Command("lsblk", "-l", "-o", "NAME,SIZE,TYPE,FSTYPE,MOUNTPOINT,LABEL,MODEL")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
		return
	}

	devices, err := getBlockDevices("NAME", "SIZE", "FSTYPE", "MOUNTPOINT", "LABEL", "TYPE")
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[0;31mError al listar dispositivos: %v\033[0m\n", err)
		os.Exit(1)
	}

	fmt.Println("\033[1mDispositivos no montados:\033[0m\n")
	fmt.Printf("\033[0;36m%-10s %-8s %-8s %-20s\033[0m\n", "DISPOSITIVO", "TAMANO", "FS", "ETIQUETA")
	fmt.Println("----------------------------------------------")
	for _, d := range devices {
		if !isUnmountedPart(d) {
			continue
		}
		label := strVal(d.Label)
		if label == "" {
			label = "(sin etiqueta)"
		}
		fmt.Printf("%-10s %-8s %-8s %-20s\n", "/dev/"+d.Name, d.Size, strVal(d.FSType), label)
	}
}

// --- INFO ---

func cmdInfo(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "\033[0;31mUso: mnt info <dispositivo>\033[0m")
		os.Exit(1)
	}
	dev := args[0]
	if !isBlockDevice(dev) {
		fmt.Fprintf(os.Stderr, "\033[0;31mError: %s no es un dispositivo de bloque valido.\033[0m\n", dev)
		os.Exit(1)
	}

	fmt.Printf("\033[1mInformacion de %s\033[0m\n\n", dev)

	model, _ := runCmd("lsblk", "-n", "-o", "MODEL", dev)
	if model == "" {
		model = "N/A"
	}
	size, _ := runCmd("lsblk", "-n", "-o", "SIZE", dev)
	fstype, _ := runCmd("blkid", "-s", "TYPE", "-o", "value", dev)
	if fstype == "" {
		fstype = "desconocido"
	}
	label, _ := runCmd("blkid", "-s", "LABEL", "-o", "value", dev)
	if label == "" {
		label = "(sin etiqueta)"
	}
	uuid, _ := runCmd("blkid", "-s", "UUID", "-o", "value", dev)
	if uuid == "" {
		uuid = "N/A"
	}
	mountpoint, _ := runCmd("lsblk", "-n", "-o", "MOUNTPOINT", dev)
	if mountpoint == "" {
		mountpoint = "(no montado)"
	}

	fmt.Printf("  %-20s %s\n", "Modelo:", model)
	fmt.Printf("  %-20s %s\n", "Dispositivo:", dev)
	fmt.Printf("  %-20s %s\n", "Tamano:", size)
	fmt.Printf("  %-20s %s\n", "Sistema archivos:", fstype)
	fmt.Printf("  %-20s %s\n", "Etiqueta:", label)
	fmt.Printf("  %-20s %s\n", "UUID:", uuid)
	fmt.Printf("  %-20s %s\n", "Punto montaje:", mountpoint)

	if strings.HasPrefix(dev, "/dev/sd") {
		if _, err := exec.LookPath("smartctl"); err == nil {
			out, err := exec.Command("sudo", "smartctl", "-H", dev).CombinedOutput()
			if err == nil {
				for _, line := range strings.Split(string(out), "\n") {
					if strings.Contains(line, "SMART overall-health") {
						parts := strings.SplitN(line, ": ", 2)
						if len(parts) == 2 {
							fmt.Printf("  %-20s %s\n", "SMART Health:", parts[1])
						}
					}
				}
			}
		}
	}
}

// --- MOUNT ---

func cmdMount(args []string) {
	ensureSudo()
	if len(args) > 0 && args[0] != "" {
		mountDevice(args[0])
		return
	}
	devices, err := getUnmountedDevices()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[0;31mError: %v\033[0m\n", err)
		os.Exit(1)
	}
	if len(devices) == 0 {
		fmt.Println("\033[0;32mNo se encontraron dispositivos no montados.\033[0m")
		return
	}
	fmt.Println("\033[1mDispositivos encontrados:\033[0m")
	for i, dev := range devices {
		label, _ := runCmd("blkid", "-s", "LABEL", "-o", "value", dev)
		if label == "" {
			label = "(sin etiqueta)"
		}
		fmt.Printf("  %d) %s  [%s]\n", i+1, dev, label)
	}
	fmt.Println("\n\033[1;33mOpciones:\033[0m")
	fmt.Println("  a) Montar todos")
	fmt.Println("  q) Cancelar")
	fmt.Print("Seleccione un numero (o 'a' / 'q'): ")
	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	switch strings.ToLower(choice) {
	case "a":
		for _, dev := range devices {
			mountDevice(dev)
		}
	case "q":
		fmt.Println("Cancelado.")
	default:
		var idx int
		if _, err := fmt.Sscanf(choice, "%d", &idx); err == nil && idx >= 1 && idx <= len(devices) {
			mountDevice(devices[idx-1])
		} else {
			fmt.Fprintln(os.Stderr, "\033[0;31mOpcion invalida.\033[0m")
			os.Exit(1)
		}
	}
}

func getUnmountedDevices() ([]string, error) {
	devices, err := getBlockDevices("NAME", "FSTYPE", "MOUNTPOINT", "TYPE")
	if err != nil {
		return nil, err
	}
	var result []string
	for _, d := range devices {
		if isUnmountedPart(d) {
			result = append(result, "/dev/"+d.Name)
		}
	}
	return result, nil
}

func mountDevice(dev string) {
	if !isBlockDevice(dev) {
		fmt.Fprintf(os.Stderr, "\033[0;31mError: %s no es un dispositivo de bloque valido.\033[0m\n", dev)
		return
	}

	fstype, _ := runCmd("blkid", "-s", "TYPE", "-o", "value", dev)
	label, _ := runCmd("blkid", "-s", "LABEL", "-o", "value", dev)
	if fstype == "" {
		fstype, _ = runCmd("lsblk", "-n", "-o", "FSTYPE", dev)
	}
	if label == "" {
		label = filepath.Base(dev)
	}
	label = strings.ReplaceAll(label, " ", "_")

	mountpoint, _ := runCmd("lsblk", "-n", "-o", "MOUNTPOINT", dev)
	if mountpoint != "" {
		fmt.Printf("\033[1;33m%s ya esta montado en %s.\033[0m\n", dev, mountpoint)
		return
	}
	if fstype == "" {
		fmt.Fprintf(os.Stderr, "\033[0;31mNo se pudo determinar el sistema de archivos de %s.\033[0m\n", dev)
		return
	}

	fmt.Printf("   \033[1;33mDetectado como %s\033[0m\n", fstype)

	switch fstype {
	case "ntfs":
		fmt.Print("   \033[1;33mVerificando sistema de archivos...\033[0m\n")
		exec.Command("sudo", "ntfsfix", dev).Run()
	case "vfat", "exfat":
		fmt.Print("   \033[1;33mVerificando sistema de archivos...\033[0m\n")
		exec.Command("sudo", "fsck", "-y", dev).Run()
	}

	mp := filepath.Join(cfg.MountBaseDir, label)
	exec.Command("sudo", "mkdir", "-p", mp).Run()

	fmt.Printf("   \033[1;33mMontando en %s...\033[0m\n", mp)

	var mounted bool
	switch fstype {
	case "ntfs":
		err1 := exec.Command("sudo", "mount", "-t", "ntfs-3g", dev, mp).Run()
		if err1 != nil {
			err2 := exec.Command("sudo", "mount", "-t", "ntfs-3g", "-o",
				fmt.Sprintf("rw,uid=%d,gid=%d", os.Getuid(), os.Getgid()), dev, mp).Run()
			mounted = err2 == nil
		} else {
			mounted = true
		}
	case "exfat":
		mounted = exec.Command("sudo", "mount", "-t", "exfat", dev, mp).Run() == nil
	case "vfat", "fat":
		mounted = exec.Command("sudo", "mount", "-t", "vfat", dev, mp).Run() == nil
	case "ext4", "ext3", "ext2":
		mounted = exec.Command("sudo", "mount", "-t", fstype, dev, mp).Run() == nil
	case "btrfs":
		mounted = exec.Command("sudo", "mount", "-t", "btrfs", dev, mp).Run() == nil
	case "xfs":
		mounted = exec.Command("sudo", "mount", "-t", "xfs", dev, mp).Run() == nil
	default:
		fmt.Fprintf(os.Stderr, "\033[0;31mSistema de archivos %s no soportado.\033[0m\n", fstype)
		return
	}
	if mounted {
		fmt.Printf("\033[0;32m%s montado en %s\033[0m\n", dev, mp)
		exec.Command("sudo", "chown", "-R", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()), mp).Run()
	} else {
		fmt.Fprintf(os.Stderr, "\033[0;31mFallo el montaje de %s.\033[0m\n", dev)
	}
}

// --- UNMOUNT ---

func cmdUnmount(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "\033[0;31mUso: mnt unmount <dispositivo>\033[0m")
		os.Exit(1)
	}
	ensureSudo()
	dev := args[0]
	if !isBlockDevice(dev) {
		fmt.Fprintf(os.Stderr, "\033[0;31mError: %s no es un dispositivo de bloque valido.\033[0m\n", dev)
		os.Exit(1)
	}
	mountpoint, _ := runCmd("lsblk", "-n", "-o", "MOUNTPOINT", dev)
	if mountpoint == "" {
		fmt.Printf("\033[1;33m%s no esta montado.\033[0m\n", dev)
		return
	}
	fmt.Printf("Desmontando %s desde %s...\n", dev, mountpoint)
	if err := exec.Command("sudo", "umount", dev).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "\033[0;31mError al desmontar %s. Puede estar en uso.\033[0m\n", dev)
		fuser := exec.Command("fuser", "-vm", mountpoint)
		fuser.Stdout = os.Stdout
		fuser.Stderr = os.Stderr
		fuser.Run()
		os.Exit(1)
	}
	fmt.Printf("\033[0;32m%s desmontado.\033[0m\n", dev)
	os.Remove(mountpoint)
}

// --- STATUS ---

func cmdStatus() {
	fmt.Println("\033[1mDispositivos montados:\033[0m\n")
	fmt.Printf("%-12s %-20s %-10s %-8s\n", "DISPOSITIVO", "PUNTO MONTAJE", "TAMANO", "FS")
	fmt.Println("----------------------------------------------------------")

	devices, err := getBlockDevices("NAME", "MOUNTPOINT", "SIZE", "FSTYPE")
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[0;31mError: %v\033[0m\n", err)
		os.Exit(1)
	}
	for _, d := range devices {
		mp := strVal(d.Mountpoint)
		if mp == "" {
			continue
		}
		if strings.HasPrefix(mp, "/") {
			fmt.Printf("%-12s %-20s %-10s %-8s\n", "/dev/"+d.Name, mp, d.Size, strVal(d.FSType))
		}
	}
}

// --- HELP ---

func cmdHelp() {
	fmt.Println("\033[1mmnt - Gestion de dispositivos de bloque\033[0m\n")
	fmt.Println("Uso: mnt <comando> [opciones]\n")
	fmt.Println("\033[1mComandos:\033[0m")
	fmt.Println("  list [--all]              Lista dispositivos (solo no montados por defecto)")
	fmt.Println("  info <dispositivo>        Muestra informacion detallada")
	fmt.Println("  mount [dispositivo]       Monta dispositivo(s) (interactivo si no se especifica)")
	fmt.Println("  unmount <dispositivo>     Desmonta un dispositivo")
	fmt.Println("  status                    Muestra dispositivos montados actualmente")
	fmt.Println("  help                      Muestra esta ayuda\n")
	fmt.Println("\033[1mEjemplos:\033[0m")
	fmt.Println("  mnt list")
	fmt.Println("  mnt list --all")
	fmt.Println("  mnt info /dev/sdb1")
	fmt.Println("  mnt mount /dev/sdb1")
	fmt.Println("  mnt mount            (modo interactivo)")
	fmt.Println("  mnt unmount /dev/sdb1")
	fmt.Println("  mnt status")
}
