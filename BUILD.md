# TouchSwitch v3.5 build instructions

Requires Go 1.23+ and Python 3. No third-party Go dependencies.

Build per architecture, regenerating the icon resource before each build:

```sh
python resource_icon.py arm64
GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags='-H=windowsgui -s -w' -o TouchSwitch-ARM64.exe .
cp TouchSwitch-ARM64.exe installer/payload/TouchSwitch.exe
cp rsrc.syso installer/rsrc.syso
(cd installer && GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags='-H=windowsgui -s -w' -o ../TouchSwitch_Setup_ARM64_v3.5.exe .)

python resource_icon.py amd64
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-H=windowsgui -s -w' -o TouchSwitch-x64.exe .
cp TouchSwitch-x64.exe installer/payload/TouchSwitch.exe
cp rsrc.syso installer/rsrc.syso
(cd installer && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-H=windowsgui -s -w' -o ../TouchSwitch_Setup_x64_v3.5.exe .)
```

The installer writes to `%LOCALAPPDATA%\\Programs\\TouchSwitch` and adds a Start menu shortcut and uninstall registration. When a user opts into prompt-free startup, installer invokes the installed app in elevated helper mode to create a Windows Task Scheduler logon task (`TouchSwitch Startup`) for the current signed-in administrative user. The startup task launches the installed exe with `-startup`. App Settings can remove the task; uninstall also attempts to remove it before deleting the app.

The installer and UAC/Task Scheduler behaviour require real-world Windows testing; cross-compilation checks do not validate task registration at runtime.
