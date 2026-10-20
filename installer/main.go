// TouchSwitch 3.4 per-user Windows installer.
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
	version           = "3.4.0"
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
		Start bool `json:"start_with_windows"`
	}
	if json.Unmarshal(data, &c) != nil || !c.Start {
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
	// No installer privilege elevation: only the installed app itself asks for
	// UAC because Windows requires admin rights for PnP touchscreen controls.
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
