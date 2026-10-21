package main

// Windows Task Scheduler auto-start. This is deliberately opt-in: one UAC
// approval is required to register an interactive, elevated *same-user* task.
// No system-wide UAC policy is modified, and we never run as SYSTEM.

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
)

const scheduledStartupName = "TouchSwitch Startup"

func taskPowerShellString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func runScheduledTaskScript(script string) error {
	units := utf16.Encode([]rune(script))
	buf := make([]byte, 2*len(units))
	for i, unit := range units {
		buf[2*i] = byte(unit)
		buf[2*i+1] = byte(unit >> 8)
	}
	encoded := base64.StdEncoding.EncodeToString(buf)
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", encoded)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		// PowerShell 5.1 emits encoded output in the OEM locale; its English
		// diagnostic isn't always readable. Give a stable, useful error.
		if len(msg) > 220 {
			msg = msg[:220]
		}
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("Task Scheduler: %s", msg)
	}
	return nil
}

// An elevated task must not launch an ordinary user-writable executable:
// a non-elevated process could otherwise replace it and gain privileges at
// next logon. Copy the signed/verified-by-user executable into Program Files
// with inherited admin-only write ACLs, then make the task target THAT copy.
func protectedStartupExecutable() (string, error) {
	programFiles := os.Getenv("ProgramFiles")
	if programFiles == "" {
		return "", fmt.Errorf("Program Files location is unavailable")
	}
	return filepath.Join(programFiles, "TouchSwitch", "TouchSwitch.exe"), nil
}

// The source is restricted to our installed executable in the same account.
// Accept our protected copy too (for re-enabling from the running tray app).
// Elevation with a DIFFERENT administrator account is rejected because its
// LOCALAPPDATA will not match the user's installed exe path.
func installedStartupExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return "", fmt.Errorf("LOCALAPPDATA is unavailable")
	}
	expected := filepath.Join(local, "Programs", "TouchSwitch", "TouchSwitch.exe")
	protected, err := protectedStartupExecutable()
	if err != nil {
		return "", err
	}
	if strings.EqualFold(filepath.Clean(exe), filepath.Clean(expected)) || strings.EqualFold(filepath.Clean(exe), filepath.Clean(protected)) {
		return exe, nil
	}
	return "", fmt.Errorf("install TouchSwitch for this Windows administrator account before enabling prompt-free startup")
}

func ensureProtectedStartupCopy(src, dst string) error {
	if strings.EqualFold(filepath.Clean(src), filepath.Clean(dst)) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("creating protected startup folder: %w", err)
	}
	bytes, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading installed TouchSwitch: %w", err)
	}
	if err := os.WriteFile(dst, bytes, 0755); err != nil {
		return fmt.Errorf("updating the protected app copy: %w (exit TouchSwitch before upgrading)", err)
	}
	return nil
}

func cleanupProtectedStartupCopy() {
	// The protected exe may currently be running (sign-in tray instance).
	// Windows doesn't allow deleting an executable in use; deleting the task
	// is the important part. A later uninstall cleans up the file.
	dst, err := protectedStartupExecutable()
	if err != nil {
		return
	}
	_ = os.Remove(dst)
	_ = os.Remove(filepath.Dir(dst))
}

func createTouchSwitchTask() error {
	source, err := installedStartupExecutable()
	if err != nil {
		return err
	}
	exe, err := protectedStartupExecutable()
	if err != nil {
		return err
	}
	if err := ensureProtectedStartupCopy(source, exe); err != nil {
		return err
	}
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$Name = %s
$User = [Security.Principal.WindowsIdentity]::GetCurrent().Name
$Action = New-ScheduledTaskAction -Execute %s -Argument '-startup'
$Trigger = New-ScheduledTaskTrigger -AtLogOn -User $User
$Principal = New-ScheduledTaskPrincipal -UserId $User -LogonType Interactive -RunLevel Highest
$Settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -MultipleInstances IgnoreNew -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName $Name -Action $Action -Trigger $Trigger -Principal $Principal -Settings $Settings -Force -ErrorAction Stop | Out-Null
`, taskPowerShellString(scheduledStartupName), taskPowerShellString(exe))
	return runScheduledTaskScript(script)
}

func deleteTouchSwitchTask() error {
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$Name = %s
$Task = Get-ScheduledTask -TaskName $Name -ErrorAction SilentlyContinue
if ($null -ne $Task) {
    Unregister-ScheduledTask -TaskName $Name -Confirm:$false -ErrorAction Stop
}
`, taskPowerShellString(scheduledStartupName))
	return runScheduledTaskScript(script)
}

func setWindowsStartup(enabled bool) error {
	if enabled {
		if err := createTouchSwitchTask(); err != nil {
			return err
		}
		// A failed cleanup must not cause two startup attempts. If it fails,
		// roll back the task rather than silently leaving both enabled.
		if err := removeLegacyStartup(); err != nil {
			_ = deleteTouchSwitchTask()
			return fmt.Errorf("couldn't remove previous startup entry: %w", err)
		}
		config.StartWithWindows = true
		config.StartupMode = "scheduled"
	} else {
		if err := deleteTouchSwitchTask(); err != nil {
			return err
		}
		if err := removeLegacyStartup(); err != nil {
			return err
		}
		cleanupProtectedStartupCopy()
		config.StartWithWindows = false
		config.StartupMode = "disabled"
	}
	saveSettings()
	return nil
}

// A v3.4 registry startup setting is not silently treated as the new,
// confirmed task-based startup setting.
func promptFreeStartupEnabled() bool {
	return config.StartWithWindows && config.StartupMode == "scheduled"
}
