# TouchSwitch v3.4.0 — Windows 11 touchscreen control

TouchSwitch is a small, native Windows utility for toggling a touchscreen without opening Device Manager. It supports Windows 11 on both ARM64 and x64.

## What's new in v3.4.0

- Animated theme selector that slides between **System**, **Light**, and **Dark**.
- One-second touchscreen ON/OFF status popup with a smooth slide-in / slide-out animation.
- Status popup can remain visible when Windows **Do Not Disturb** is enabled.
- Proper Windows installer with **Start menu integration** and **Installed apps** uninstall support.
- Existing saved theme, shortcut, and preferences are retained during upgrades.

## Features

- **Left-click** the tray icon to toggle the touchscreen.
- **Right-click** the tray icon for Open, Settings, and Exit.
- Default global shortcut: **Ctrl + Alt + T** (change it in Settings).
- Clean Windows 11-style appearance, Fluent iconography, and Windows accent colours.
- Native Windows ARM64 and x64 executables.

## Which file should I download?

| Windows device | Installer |
| --- | --- |
| ARM64 / Snapdragon, including Galaxy Book4 Edge | `TouchSwitch_Setup_ARM64_v3.4.exe` |
| x64 / Intel or AMD | `TouchSwitch_Setup_x64_v3.4.exe` |

Exit any previous TouchSwitch instance, run the appropriate installer, and open TouchSwitch from the Start menu. Windows will request administrator permission when the application launches to manage the touchscreen hardware device.

**Note:** Builds are currently unsigned and have not been comprehensively tested across Windows devices. Windows may warn about an unknown publisher. Report problems with the Windows version, processor architecture, touchscreen device and relevant screenshots/logs.
