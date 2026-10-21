// TouchSwitch 3.5 per-user Windows installer.
// Installs ARM64/x64 native TouchSwitch; exposes Start menu and Windows uninstall.
// This installer needs no third-party runtime or administrator rights.
package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

//go:embed payload/TouchSwitch.exe
var appBinary []byte

const (
	version           = "3.5.0"
	regSz             = 1
	regDword          = 4
	hkcu              = uintptr(0x80000001)
	keyWrite          = 0x20006
	mbOk              = 0
	mbYesNo           = 0x4
	mbIconInformation = 0x40
	mbIconWarning     = 0x30
	mbIconError       = 0x10
	mbDefaultButton2  = 0x100
	yes               = 6
)

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	advapi         = syscall.NewLazyDLL("advapi32.dll")
	msgBox         = user32.NewProc("MessageBoxW")
	findWindow     = user32.NewProc("FindWindowW")
	regCreateKey   = advapi.NewProc("RegCreateKeyExW")
	regSetValue    = advapi.NewProc("RegSetValueExW")
	regDeleteTree  = advapi.NewProc("RegDeleteTreeW")
	regCloseKey    = advapi.NewProc("RegCloseKey")
	regOpenKey     = advapi.NewProc("RegOpenKeyExW")
	regQueryValue  = advapi.NewProc("RegQueryValueExW")
	regDeleteValue = advapi.NewProc("RegDeleteValueW")
)

func w(s string) uintptr { return uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(s))) }
func ask(message string) bool {
	v, _, _ := msgBox.Call(0, w(message), w("TouchSwitch Setup"), mbYesNo|mbIconInformation)
	return v == yes
}
func info(message string) { msgBox.Call(0, w(message), w("TouchSwitch Setup"), mbOk|mbIconInformation) }
func fail(err error)      { msgBox.Call(0, w(err.Error()), w("TouchSwitch Setup"), mbOk|mbIconError) }
func u16Bytes(s string) []byte {
	units := syscall.StringToUTF16(s)
	out := make([]byte, len(units)*2)
	for i, v := range units {
		out[2*i] = byte(v)
		out[2*i+1] = byte(v >> 8)
	}
	return out
}
func writeRegString(key uintptr, name, value string) error {
	d := u16Bytes(value)
	r, _, _ := regSetValue.Call(key, w(name), 0, regSz, uintptr(unsafe.Pointer(&d[0])), uintptr(len(d)))
	if r != 0 {
		return fmt.Errorf("registry value %s: %d", name, r)
	}
	return nil
}
func writeRegDword(key uintptr, name string, value uint32) error {
	r, _, _ := regSetValue.Call(key, w(name), 0, regDword, uintptr(unsafe.Pointer(&value)), 4)
	if r != 0 {
		return fmt.Errorf("registry value %s: %d", name, r)
	}
	return nil
}
func openOrCreate(path string) (uintptr, error) {
	var k uintptr
	r, _, _ := regCreateKey.Call(hkcu, w(path), 0, 0, 0, keyWrite, 0, uintptr(unsafe.Pointer(&k)), 0)
	if r != 0 {
		return 0, fmt.Errorf("could not create registry entry: %d", r)
	}
	return k, nil
}
func registerUninstaller(dir string, appPath, uninstallPath string) error {
	key, err := openOrCreate(`Software\Microsoft\Windows\CurrentVersion\Uninstall\TouchSwitch`)
	if err != nil {
		return err
	}
	defer regCloseKey.Call(key)
	pairs := map[string]string{
		"DisplayName": "TouchSwitch", "DisplayVersion": version,
		"Publisher": "TouchSwitch", "InstallLocation": dir,
		"DisplayIcon":          appPath,
		"UninstallString":      fmt.Sprintf(`"%s" --uninstall`, uninstallPath),
		"QuietUninstallString": fmt.Sprintf(`"%s" --uninstall --quiet`, uninstallPath),
	}
	for n, v := range pairs {
		if err := writeRegString(key, n, v); err != nil {
			return err
		}
	}
	if err := writeRegDword(key, "NoModify", 1); err != nil {
		return err
	}
	if err := writeRegDword(key, "NoRepair", 1); err != nil {
		return err
	}
	if err := writeRegDword(key, "EstimatedSize", uint32((len(appBinary)+1023)/1024)); err != nil {
		return err
	}
	return nil
}
func readRegString(key uintptr, name string) string {
	var kind uint32
	var need uint32
	r, _, _ := regQueryValue.Call(key, w(name), 0, uintptr(unsafe.Pointer(&kind)), 0, uintptr(unsafe.Pointer(&need)))
	if r != 0 || kind != regSz || need == 0 || need > 65536 {
		return ""
	}
	buf := make([]uint16, (need+1)/2)
	r, _, _ = regQueryValue.Call(key, w(name), 0, uintptr(unsafe.Pointer(&kind)), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&need)))
	if r != 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}
func migrateStartupIfEnabled(appPath string) {
	data, _ := os.ReadFile(filepath.Join(os.Getenv("APPDATA"), "TouchSwitch", "settings.json"))
	var c struct {
		Start bool   `json:"start_with_windows"`
		Mode  string `json:"startup_mode"`
	}
	// v3.5 task based startup must not be converted back to the old
	// UAC-producing Run entry when upgrading the installed executable.
	if json.Unmarshal(data, &c) != nil || !c.Start || c.Mode == "scheduled" {
		return
	}
	key, err := openOrCreate(`Software\Microsoft\Windows\CurrentVersion\Run`)
	if err != nil {
		return
	}
	defer regCloseKey.Call(key)
	_ = writeRegString(key, "TouchSwitch", fmt.Sprintf(`"%s" -startup`, appPath))
}
func removeStartupIfOwned(dir string) {
	var key uintptr
	r, _, _ := regOpenKey.Call(hkcu, w(`Software\Microsoft\Windows\CurrentVersion\Run`), 0, 0x2001f /* KEY_READ | KEY_WRITE */, uintptr(unsafe.Pointer(&key)))
	if r != 0 {
		return
	}
	defer regCloseKey.Call(key)
	if s := readRegString(key, "TouchSwitch"); strings.Contains(strings.ToLower(s), strings.ToLower(filepath.Clean(dir))) {
		regDeleteValue.Call(key, w("TouchSwitch"))
	}
}
func shortcutPath() string {
	return filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "TouchSwitch.lnk")
}
func psString(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func createShortcut(link, target string) error {
	script := fmt.Sprintf(`$shell=New-Object -ComObject WScript.Shell; $shortcut=$shell.CreateShortcut(%s); $shortcut.TargetPath=%s; $shortcut.WorkingDirectory=%s; $shortcut.Description='Toggle your Windows touchscreen'; $shortcut.Save()`, psString(link), psString(target), psString(filepath.Dir(target)))
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Start menu shortcut: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Launches the *installed* app in elevated helper mode only after the user
// explicitly opts in. This asks for UAC once, instead of at every sign-in.
func manageStartupTask(appPath string, enable bool) error {
	mode := "--disable-startup-task"
	if enable {
		mode = "--enable-startup-task"
	}
	script := fmt.Sprintf(`$ErrorActionPreference='Stop'; $p=Start-Process -FilePath %s -ArgumentList %s -Verb RunAs -Wait -PassThru; if ($p.ExitCode -ne 0) { exit 1 }`, psString(appPath), psString(mode))
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		result := strings.TrimSpace(string(out))
		if len(result) > 240 {
			result = result[:240]
		}
		if result == "" {
			result = "The administrator permission was cancelled or task setup failed."
		}
		return fmt.Errorf("could not change scheduled startup: %s", result)
	}
	return nil
}
func startupTaskWasEnabled() bool {
	data, _ := os.ReadFile(filepath.Join(os.Getenv("APPDATA"), "TouchSwitch", "settings.json"))
	var c struct {
		Start bool   `json:"start_with_windows"`
		Mode  string `json:"startup_mode"`
	}
	return json.Unmarshal(data, &c) == nil && c.Start && c.Mode == "scheduled"
}
func installDir() string { return filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "TouchSwitch") }
func appIsRunning() bool {
	hwnd, _, _ := findWindow.Call(w("TouchSwitch.Tray.Popup.V3"), 0)
	return hwnd != 0
}
func install() error {
	if os.Getenv("LOCALAPPDATA") == "" || os.Getenv("APPDATA") == "" {
		return errors.New("Windows app data folders could not be resolved")
	}
	if appIsRunning() {
		info("Please exit the previous TouchSwitch app from its system tray menu, then run Setup again. Your settings will be kept.")
		return nil
	}
	dir := installDir()
	if !ask("Install TouchSwitch v" + version + "?\n\nInstall location:\n" + dir + "\n\nAdds a Start menu shortcut and an Uninstall entry in Windows Settings. Your existing theme and shortcut settings are preserved.") {
		return nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	appPath := filepath.Join(dir, "TouchSwitch.exe")
	if err := os.WriteFile(appPath, appBinary, 0755); err != nil {
		return fmt.Errorf("Cannot install TouchSwitch.exe. Exit the old app and retry: %w", err)
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	b, err := os.ReadFile(self)
	if err != nil {
		return err
	}
	uninstallPath := filepath.Join(dir, "Uninstall TouchSwitch.exe")
	if err := os.WriteFile(uninstallPath, b, 0755); err != nil {
		return err
	}
	if err := registerUninstaller(dir, appPath, uninstallPath); err != nil {
		return err
	}
	migrateStartupIfEnabled(appPath)
	shortcutError := createShortcut(shortcutPath(), appPath)
	if shortcutError != nil {
		info("TouchSwitch is installed, but Windows couldn't create its Start menu shortcut.\n\n" + shortcutError.Error())
	}
	if startupTaskWasEnabled() {
		// An upgrade refreshes the protected Program Files copy, not just the
		// user-profile executable. Requires one approval for upgrades too.
		if ask("Update the protected TouchSwitch startup copy to v" + version + "?\n\nThis requires one administrator approval for the new version. Your prompt-free startup will continue to work at sign-in.") {
			if err := manageStartupTask(appPath, true); err != nil {
				info("TouchSwitch was installed, but the protected auto-start copy was not updated.\n\n" + err.Error())
			}
		}
	} else {
		if ask("Enable prompt-free startup?\n\nTouchSwitch will start automatically when you sign in, with administrator permission but without a UAC prompt each restart.\n\nWindows will ask you to approve the scheduled task ONCE. You can disable it in TouchSwitch Settings.") {
			if err := manageStartupTask(appPath, true); err != nil {
				info("TouchSwitch was installed, but automatic startup could not be enabled.\n\n" + err.Error() + "\n\nYou can try again in TouchSwitch Settings.")
			}
		}
	}
	if ask("TouchSwitch was installed successfully!\n\nYou can find it in Start > All apps or Windows Settings > Installed apps.\n\nLaunch TouchSwitch now?") {
		cmd := exec.Command(appPath)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("installed successfully but couldn't launch: %w", err)
		}
	}
	return nil
}
func uninstall(quiet bool) error {
	dir := installDir()
	if appIsRunning() {
		info("Exit TouchSwitch from its system tray icon before uninstalling, then try again.")
		return nil
	}
	if !quiet && !ask("Uninstall TouchSwitch from this PC?\n\nYour saved theme and keyboard shortcut will be retained for a future reinstall.") {
		return nil
	}
	_, protectedError := os.Stat(filepath.Join(os.Getenv("ProgramFiles"), "TouchSwitch", "TouchSwitch.exe"))
	if startupTaskWasEnabled() || protectedError == nil {
		// A highest-privileges task should not be orphaned on uninstall.
		// Removal requires one UAC approval, and is handled by the installed
		// app while its executable is still present.
		if err := manageStartupTask(filepath.Join(dir, "TouchSwitch.exe"), false); err != nil {
			return fmt.Errorf("disable prompt-free startup before uninstalling (Settings > Prompt-free startup), or approve the Windows permission prompt: %w", err)
		}
	}
	_ = os.Remove(shortcutPath())
	removeStartupIfOwned(dir)
	regDeleteTree.Call(hkcu, w(`Software\Microsoft\Windows\CurrentVersion\Uninstall\TouchSwitch`))
	_ = os.Remove(filepath.Join(dir, "TouchSwitch.exe"))
	// The uninstaller itself cannot be deleted while running. Spawn a hidden
	// cmd that waits for this process to exit, then removes only our files.
	self, err := os.Executable()
	if err != nil {
		return err
	}
	expected := filepath.Join(dir, "Uninstall TouchSwitch.exe")
	if strings.EqualFold(filepath.Clean(self), filepath.Clean(expected)) {
		command := fmt.Sprintf(`ping 127.0.0.1 -n 3 >NUL & del /F /Q "%s" & rmdir "%s"`, expected, dir)
		cmd := exec.Command("cmd.exe", "/C", command)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := cmd.Start(); err != nil {
			return err
		}
	} else {
		_ = os.Remove(expected)
		_ = os.Remove(dir)
	}
	if !quiet {
		info("TouchSwitch was uninstalled. Your preferences were left intact.")
	}
	return nil
}
func main() {
	// Installation needs no elevation. Opting into prompt-free startup (or
	// removing its elevated task) requests one consent through the app helper.
	if len(os.Args) > 1 && os.Args[1] == "--uninstall" {
		quiet := len(os.Args) > 2 && os.Args[2] == "--quiet"
		if err := uninstall(quiet); err != nil {
			fail(err)
		}
		return
	}
	if err := install(); err != nil {
		fail(err)
	}
}
