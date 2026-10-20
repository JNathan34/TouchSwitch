# Publishing TouchSwitch on GitHub

## Repository details

**Repository name:** `TouchSwitch`

**Description:** A lightweight Windows 11 utility to instantly toggle your touchscreen using a keyboard shortcut or system tray icon. Native ARM64 and x64 support, Fluent UI, and customisable settings.

**Topics:** `windows11`, `touchscreen`, `touchscreen-toggle`, `system-tray`, `windows-arm64`, `golang`, `fluent-design`, `utility`, `keyboard-shortcuts`

**Licence:** MIT (included)

## Upload source

1. Create a new public GitHub repository called `TouchSwitch`.
2. Upload the files and folders from this package to the repository root (the `README.md` file should be in the root, not a nested folder).
3. Do not add compiled `.exe` installers directly to the repository; upload them as release assets instead.
4. Choose **MIT** if GitHub prompts for a licence, but you already have the included `LICENSE` file, so avoid adding a second copy.

## Publish first release

1. Click **Releases → Draft a new release**.
2. Create a tag `v3.4.0`.
3. Set release title to `TouchSwitch v3.4.0`.
4. Paste the content of `RELEASE_NOTES_v3.4.0.md` into the release description.
5. Attach these two prebuilt installer files:
   - `TouchSwitch_Setup_ARM64_v3.4.exe`
   - `TouchSwitch_Setup_x64_v3.4.exe`
6. Publish the release.

If you want your README to show the interface, capture a screenshot of the actual v3.4 app, save it to an `assets/` directory, and add a Markdown image reference.

## Build note

The `installer/payload/` directory is empty in the repository. Build the app executable first and copy it into that directory as `TouchSwitch.exe` before compiling the installer. The generated `rsrc.syso` and `.exe` build files are ignored by `.gitignore`.
