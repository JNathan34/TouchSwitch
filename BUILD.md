# TouchSwitch v3.4 build guide

Requires Go 1.23+ and Python 3. Runs on Linux or Windows cross-compiling to Windows.

Native ARM64 app:

```sh
python resource_icon.py arm64
GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags='-H=windowsgui -s -w' -o TouchSwitch-ARM64.exe .
```

Then build the self-contained ARM64 installer:

```sh
cp TouchSwitch-ARM64.exe installer/payload/TouchSwitch.exe
cp rsrc.syso installer/rsrc.syso
cd installer
GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags='-H=windowsgui -s -w' -o TouchSwitch_Setup_ARM64.exe .
```

For x64 replace `arm64` with `amd64` for the Go build variables and
`resource_icon.py` argument. Both the app and setup require matching icon
resources, and the app must be copied into `installer/payload` before building.

The `.exe` setup installs to `%LOCALAPPDATA%\\Programs\\TouchSwitch`, creates a
Start menu `.lnk` via the built-in Windows PowerShell COM shortcut facility,
and writes HKCU uninstall registry keys. It stores an uninstaller in the app
folder and offers launch upon completion. No installer elevation is needed.
The app itself requests elevation on launch because Windows requires it for
PnP device enable/disable.

The app uses the standard Windows 11 Segoe Fluent Icons font and GDI+.
No third-party libraries, font files, or bundled Windows DLLs are needed.
