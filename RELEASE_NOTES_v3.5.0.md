# TouchSwitch v3.5.0

### New — Prompt-free startup
- One-time opt-in setup registers a per-user Task Scheduler logon task with **Run with highest privileges**, so TouchSwitch can start on sign-in without repeated UAC prompts.
- New **Settings → Prompt-free startup** toggle to enable or disable the task.
- Migrates the older registry auto-start entry when the elevated startup task is enabled, preventing double launches.
- Installer offers prompt-free startup and requests UAC once for task setup. The task launches an admin-protected copy from Program Files rather than a user-writable executable.
- On subsequent upgrades, setup offers to refresh that protected copy with one UAC approval. Uninstaller removes the task and protected copy with approval before removing files.

### Existing features
- Touchscreen toggle via tray left-click, central button, and configurable keyboard shortcut.
- Windows 11 Fluent-inspired styling, smooth transitions, accent/theme matching.
- Compact, quick notifications even during Do Not Disturb.
- Native Windows ARM64 and x64 builds.

**Note:** The scheduled task affects automatic sign-in startup; manually launching the executable can still produce a Windows elevation prompt. Administrator account required. The elevated startup copy is located in the protected Program Files directory.

**Testing:** Built for both target architectures, but needs hands-on testing of Task Scheduler and the installer on Windows.
