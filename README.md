<div align="center">

<img src="app-icon.png" alt="TouchSwitch icon" width="88" />

# TouchSwitch

**A lightweight native touchscreen toggle for Windows 11.**

Windows 11 · ARM64 & x64 · Free and open source

</div>

TouchSwitch quickly enables or disables your touchscreen from the system tray, a configurable keyboard shortcut, or a small Windows 11-style window.

## Features

- Left-click the tray icon to toggle touch immediately.
- Global shortcut: **Ctrl + Alt + T** by default; choose your own in Settings.
- Windows 11-inspired icons, accent colour, animated switches, and light/dark/system themes.
- Quick animated status notification visible even with Windows Do Not Disturb enabled.
- **Optional prompt-free startup (v3.5):** Registers a Windows Task Scheduler logon task for your signed-in administrator account. Grant UAC permission **once** when enabling it, then TouchSwitch can start elevated without prompting at each sign-in.
- Start menu shortcut, installer, and Windows Installed apps uninstall entry.
- Native ARM64 for Snapdragon Windows laptops and x64 for Intel/AMD devices.

## Install

1. Download the ARM64 or x64 setup executable from **Releases**.
2. Exit any previous TouchSwitch instance using **right-click tray icon → Exit**.
3. Run setup; your settings are preserved during upgrades.
4. When asked **Enable prompt-free startup?**, choose **Yes** to register the logon task. Windows will ask for administrator approval once.
5. Alternatively open **Settings → Prompt-free startup** to enable/disable it later.

With prompt-free startup enabled, the scheduled task starts the app when **you sign in**, rather than via the older registry Run entry. This does **not** disable Windows UAC globally. Opening the executable directly from Start may still request UAC; the prompt-free benefit applies to automatic sign-in startup.

If you use a standard account rather than an administrator account, this method may not be supported; accepting elevation using a different administrator's credentials is intentionally rejected for task creation. Prompt-free startup requires the installed version, not a portable executable.

**Security note:** The scheduled task runs TouchSwitch with elevated permissions. Enabling it first copies the executable to an administrator-protected folder under `Program Files`, and the scheduled task launches that protected copy, rather than the user-writable app file. Future updates to that protected copy require administrator approval once per update. Only install builds you trust.

## Requirements

- Windows 11 ARM64 or x64
- HID-compliant touchscreen
- An administrator account for elevated device control and prompt-free startup

## Uninstall

Use **Windows Settings → Apps → Installed apps → TouchSwitch → Uninstall**. If prompt-free startup is enabled or a protected copy remains, uninstall requests administrator approval to remove the scheduled task and protected copy before removing the application. Theme and shortcut settings remain for future reinstalls.

## Build

The Go project has no external Go dependencies. See [BUILD.md](BUILD.md) for cross-compilation and installer instructions. The bundled project also includes the Go installer. Regenerate icon resources for each target architecture with `python resource_icon.py arm64` or `python resource_icon.py amd64`.

## Licence

MIT — see [LICENSE](LICENSE).

> Unsigned binary: SmartScreen may warn about an unrecognised publisher. The actual installer and Task Scheduler functionality must be validated on Windows hardware; compilation alone cannot prove that UAC/scheduling behaves correctly.
