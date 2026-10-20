TouchSwitch v3.4 — Windows 11 touchscreen controller

Built for Samsung Galaxy Book4 Edge ARM64 and regular Windows x64.

INSTALL (recommended)
1. Exit any previously running TouchSwitch from the system tray (right-click > Exit).
2. Run TouchSwitch_Setup_ARM64_v3.4.exe on the Galaxy Book4 Edge.
3. Confirm Install. Setup stores TouchSwitch in your user Applications folder,
   creates a Start menu shortcut, and registers an entry in Installed apps.
4. After installation, launch it from Start. The app still asks for Windows
   administrator approval when controlling the touchscreen.

NEW IN v3.4
- Theme selector indicator slides smoothly to System, Light or Dark.
- DND-friendly one-second touchscreen feedback flies in from the right,
  stays visible briefly, and slides away.
- Windows installer and uninstaller with Start menu integration.
- Existing settings are preserved when installing/upgrading/uninstalling.

CONTROLS
- Left-click tray icon: immediately toggle the touchscreen.
- Right-click tray icon: Open TouchSwitch, Settings, Exit.
- Default keyboard shortcut: Ctrl + Alt + T; change it in Settings.
- Theme follows the Windows accent, with #191919 in dark mode.

UNINSTALL
Windows Settings > Apps > Installed apps > TouchSwitch > Uninstall.
The uninstaller keeps settings.json (so your shortcut/theme survive reinstall).

INSTALLATION LOCATION
%LOCALAPPDATA%\Programs\TouchSwitch

SOURCE / BUILD
This project is native Go, with no network dependencies.
See BUILD.md for build commands. The setup executable is a self-contained
installer, not a Microsoft Store/MSIX package. It is unsigned, so Windows
may warn that it is from an unknown publisher.

NOTE
Compiled and structurally checked on Linux for Windows ARM64/x64. Actual
Windows UI, installer, and touchscreen operation require testing on device.
