<div align="center">

<img src="app-icon.png" alt="TouchSwitch icon" width="88" />

# TouchSwitch

**A lightweight, native touchscreen toggle for Windows 11.**

Switch touch input on or off in seconds — from the system tray, with a keyboard shortcut, or through a clean Windows 11-style interface.

**Windows 11** · **ARM64 & x64** · **Free & open source**

</div>

## Features

- **One-click control:** Left-click the system tray icon to enable or disable the touchscreen.
- **Customisable global shortcut:** Use `Ctrl + Alt + T` by default, or record your own key combination.
- **Windows 11-inspired design:** Fluent-style icons, responsive hover effects, subtle animations and rounded controls.
- **Light, dark or system theme:** Follows your Windows accent colour; dark theme uses `#191919`.
- **Quick status notifications:** A small animated on/off popup slides in from the right, then disappears after about one second. Because it is drawn by TouchSwitch itself, it can appear while Windows Do Not Disturb is enabled.
- **Runs from the tray:** Right-click for Open, Settings and Exit.
- **Convenient installer:** Adds a Start menu shortcut and an entry under Installed apps.
- **Native Windows builds:** ARM64 for devices such as the Samsung Galaxy Book4 Edge, plus x64 for conventional Windows PCs.

## Download and install

1. Open this repository's **Releases** section and choose the latest version.
2. Download the installer for your device:
   - `TouchSwitch_Setup_ARM64_v3.4.exe` — Windows ARM64 (including Snapdragon-powered laptops).
   - `TouchSwitch_Setup_x64_v3.4.exe` — Windows x64 (Intel / AMD).
3. Exit any older TouchSwitch instance from the system tray.
4. Run the installer, then launch **TouchSwitch** from the Start menu.
5. Accept the Windows administrator prompt when TouchSwitch starts. Windows requires elevated permission to enable and disable hardware devices.

> **Unsigned software:** These builds are not code-signed. Windows may show an unknown-publisher / SmartScreen warning. Download only from a repository or release you trust.

## Using TouchSwitch

| Action | How |
| --- | --- |
| Toggle touchscreen | **Left-click** the tray icon |
| Toggle with keyboard | Press `Ctrl + Alt + T` (configurable) |
| Open app or settings | **Right-click** the tray icon |
| Toggle from main window | Click the large central button |
| Change theme | Settings → **System / Light / Dark** |
| Start automatically | Enable **Start with Windows** in Settings |
| Uninstall | Windows Settings → Apps → Installed apps → TouchSwitch |

Preferences such as the theme and shortcut are saved locally and retained after uninstalling, so they can be restored if you reinstall the app.

## Requirements

- Windows 11 on **ARM64** or **x64**.
- A Windows-recognised **HID touchscreen** device.
- Administrator permission when using the application to control the device.

TouchSwitch is designed to target the touchscreen rather than the touchpad. Compatibility with every touchscreen controller has **not** been verified; test it on your device. You can also re-enable a touchscreen manually through Windows Device Manager if needed.

## Building from source

The application is written in **Go** and uses Windows APIs. It does not need a browser, cloud account, or third-party runtime. Build instructions for both the app and installer are in [BUILD.md](BUILD.md).

You'll need Go 1.23+ and Python 3 to generate the appropriate Windows executable icon resources. If you want to **regenerate the icon artwork**, the `make_icons.py` script additionally requires Pillow and CairoSVG.

## Privacy

The app operates locally to change the touchscreen device state and save its own preferences. It does not require an account or network connection.

## Licence

TouchSwitch is released under the [MIT License](LICENSE).

---

<sub>TouchSwitch v3.4.0 · Developed for Windows 11 · ARM64 and x64</sub>
