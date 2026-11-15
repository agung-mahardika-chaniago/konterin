// Command launcher menjalankan FixPhone di Windows tanpa Docker.
//
// Launcher hanyalah jembatan tipis: seluruh logika start/stop/restart/status/
// logs/shortcut tetap berada di scripts/fixphone.ps1 agar satu sumber
// kebenaran. File ini dibuat agar pengguna dapat menjalankan aplikasi dengan
// dobel-klik FixPhone.exe.
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	launcherVersion = "1.0.0"
	scriptRelative  = "scripts/fixphone.ps1"
)

// actions must stay in sync with the ValidateSet in scripts/fixphone.ps1.
var actions = []string{"start", "stop", "restart", "status", "logs", "shortcut"}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	// Dobel-klik di Explorer: tidak ada argumen, jendela harus tetap terbuka
	// agar pesan launcher bisa dibaca.
	interactive := len(argv) == 0

	action := "start"
	rest := argv
	if len(argv) > 0 && !strings.HasPrefix(argv[0], "-") {
		candidate := strings.ToLower(argv[0])
		switch candidate {
		case "version", "--version", "-v":
			fmt.Printf("FixPhone launcher %s\n", launcherVersion)
			return 0
		case "help", "-h", "--help":
			printUsage()
			return 0
		}
		if !isAction(candidate) {
			fmt.Fprintf(os.Stderr, "Perintah tidak dikenal: %s\n\n", argv[0])
			printUsage()
			return 2
		}
		action = candidate
		rest = argv[1:]
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "XX  %v\n", err)
		if interactive {
			pause()
		}
		return 1
	}

	script := filepath.Join(repoRoot, scriptRelative)
	if _, err := os.Stat(script); err != nil {
		fmt.Fprintf(os.Stderr, "XX  Script launcher tidak ditemukan: %s\n", script)
		if interactive {
			pause()
		}
		return 1
	}

	powershell, err := findPowerShell()
	if err != nil {
		fmt.Fprintf(os.Stderr, "XX  %v\n", err)
		if interactive {
			pause()
		}
		return 1
	}

	cmdArgs := []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script, action}
	cmdArgs = append(cmdArgs, rest...)

	cmd := exec.Command(powershell, cmdArgs...)
	cmd.Dir = repoRoot
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			fmt.Fprintf(os.Stderr, "XX  Gagal menjalankan launcher: %v\n", err)
			code = 1
		}
	}

	if interactive {
		pause()
	}
	return code
}

func isAction(name string) bool {
	for _, action := range actions {
		if action == name {
			return true
		}
	}
	return false
}

func printUsage() {
	fmt.Println("FixPhone launcher — menjalankan aplikasi tanpa Docker")
	fmt.Println()
	fmt.Println("Pemakaian: FixPhone.exe [perintah] [opsi launcher]")
	fmt.Println()
	fmt.Println("Perintah:")
	fmt.Println("  start      Jalankan database, API, dan frontend (default)")
	fmt.Println("  stop       Hentikan semua proses")
	fmt.Println("  restart    Hentikan lalu jalankan ulang")
	fmt.Println("  status     Tampilkan status database, API, dan frontend")
	fmt.Println("  logs       Tampilkan log terakhir (-Follow untuk mengikuti)")
	fmt.Println("  shortcut   Buat shortcut FixPhone di Desktop")
	fmt.Println("  version    Tampilkan versi launcher")
	fmt.Println()
	fmt.Println("Contoh:")
	fmt.Println(`  FixPhone.exe start -NoBrowser`)
	fmt.Println(`  FixPhone.exe start -TenantName "Toko Saya" -OwnerEmail owner@toko.id -OwnerPassword "Rahasia123"`)
}

func pause() {
	fmt.Println()
	fmt.Print("Tekan Enter untuk menutup jendela ini...")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}

func findPowerShell() (string, error) {
	if path, err := exec.LookPath("powershell.exe"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("powershell"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("pwsh.exe"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("PowerShell tidak ditemukan di PATH")
}

// findRepoRoot mencari folder yang memuat scripts/fixphone.ps1, dimulai dari
// lokasi exe (FixPhone.exe berada di root repo) lalu folder induknya.
func findRepoRoot() (string, error) {
	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		candidates = append(candidates, filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, cwd)
	}

	for _, start := range candidates {
		dir := start
		for depth := 0; depth < 6; depth++ {
			if info, err := os.Stat(filepath.Join(dir, scriptRelative)); err == nil && !info.IsDir() {
				return dir, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	return "", fmt.Errorf("folder proyek FixPhone tidak ditemukan (%s tidak ada)", scriptRelative)
}
