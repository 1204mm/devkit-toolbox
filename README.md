<p align="center">
  <img src="icon-1024-rounded.png" width="128" alt="DevKit" />
</p>

<h1 align="center">DevKit — Developer Toolbox for Windows</h1>

<p align="center">
  A Windows desktop developer toolbox built with <a href="https://wails.io">Wails v2</a> (Go + Vue 3)
</p>

<p align="center">
  <a href="./README.zh-CN.md"><img alt="Chinese (简体中文)" src="https://img.shields.io/badge/中文文档-%E7%AE%80%E4%BD%93%E4%B8%AD%E6%96%87-blue"></a>
  <a href="./LICENSE"><img alt="License: MIT" src="https://img.shields.io/badge/License-MIT-yellow.svg"></a>
  <img alt="Platform: Windows" src="https://img.shields.io/badge/Platform-Windows%2010%2F11-0078D6.svg">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25-00ADD8.svg">
  <img alt="Wails v2" src="https://img.shields.io/badge/Wails-v2-F7DF1E.svg">
  <img alt="Vue 3" src="https://img.shields.io/badge/Vue-3-42B883.svg">
</p>

> A battery-included toolbox featuring port management, cryptography utilities, TOTP 2FA,
> JSON formatter, and an **icon generator with automatic rounded-corner detection**.

---

## ✨ Features

### 🔌 Port Manager

- Scan all listening ports in one click (`tcp4` / `tcp6`), with automatic identification of
  Java / Node / Nginx / Python processes
- Smart project detection: extracts the Java main class, JAR name, and Node script name so you
  can see at a glance which project holds a port
- Kill the selected process tree with a force-kill (`taskkill /F /T`)
- Or type a port number to kill the process bound to it directly
- Only prompts for administrator elevation when a kill actually fails due to insufficient
  permissions — no noisy warnings

### 🔐 Crypto & Encoding

- **Hashing:** MD5 / SHA1 / SHA256 / SHA512
- **HMAC:** HMAC-MD5 / HMAC-SHA256 / HMAC-SHA512
- **Symmetric encryption:** AES-CBC, DES-CBC (Base64 in/out, random IV)
- **Asymmetric encryption:** RSA 2048 key generation / encryption / decryption
- **Encoding:** Base64, Hex, URL encode/decode
- **Password hashing:** bcrypt generate & verify

### 🔢 2FA Authenticator

- Standard TOTP (RFC 6238), compatible with Google Authenticator
- Master-password protected: keys are AES-encrypted on disk (`totp.dat`), master password
  verified with bcrypt
- Click-to-copy codes, with a 30-second countdown progress ring

### 📋 JSON Formatter

- Format / minify / validate, with 2- or 4-space indentation
- One-click copy and one-click fill-output-back-into-input

### 🖼 IconForge — Icon Generator

- Convert PNG / JPG / BMP / GIF to multi-size ICO in one go (select any of 16 ~ 256 px)
- **Automatic rounded-corner detection:** detects and cuts away excess background around the
  corners of the source image (e.g. icons with a pale rounded outer frame) into transparency,
  with a slider to fine-tune the radius
- **Square cropping:** drag on the canvas to adjust the crop area, with live multi-size preview
- Rounded-corner cutting uses a signed-distance-field (SDF) anti-aliased mask for smooth
  edges, plus high-quality Lanczos scaling
- **ZIP export:** every selected size as its own `.ico` **plus** an all-in-one `icon.ico`,
  packaged together

### 🧰 Common Utilities

| Tool | Description |
|------|-------------|
| Timestamp converter | Live timestamp display (click to copy); auto-detect Unix sec/ms ↔ local time / weekday / ISO 8601 / UTC; date-string ↔ timestamp |
| JWT decoder | Fully local Header/Payload decode, live `exp` status (remaining time / expired), highlighted time claims, Bearer prefix support |
| Cron expression | Standard 5-field and Quartz 6~7-field (`?`, `L`, `#`, `MON-FRI`, year field), with Chinese semantic descriptions + next 5 run times + field breakdown |
| UUID generator | Batch v4 UUIDs (1–500), with un-dashed / uppercase options, click-to-copy |
| Regex tester | Live match highlighting, g / i / m / s / u flags, match details with capture groups & ranges |

## 🛠 Tech Stack

| Layer | Technology |
|-------|------------|
| Desktop framework | [Wails v2](https://wails.io) |
| Backend | Go ([gopsutil](https://github.com/shirou/gopsutil) for ports & processes, [imaging](https://github.com/disintegration/imaging) for image processing) |
| Frontend | Vue 3 + TypeScript + Vite |
| UI | Ant Design Vue 4 (Catppuccin Mocha dark theme) |
| Cron parsing | [cron-parser](https://github.com/pentestfunctions/cron-parser) + [cronstrue](https://github.com/bradymholt/cronstrue) |
| Icon algorithms | corner detection (diagonal/edge scan + least-squares arc fitting), SDF anti-aliased mask, Lanczos scaling, ICO encoding (BMP + PNG dual format), with unit tests |

## 🚀 Development & Build

Requirements: Go 1.21+, Node.js 18+, [Wails CLI](https://wails.io/docs/gettingstarted/installation) v2. **Windows only.**

```bash
# Install the Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Development mode (front-end hot reload)
wails dev

# Production build → build/bin/portmanager.exe
wails build
```

## ⬇️ Download

Grab the latest build from the **[Releases](https://github.com/1204mm/devkit-toolbox/releases)** page (all-in-one `portmanager.exe`, no installation required).

## 📄 Notes

- Everything runs locally — port scanning, file/process operations, JWT decode, and timestamp
  conversion are all handled on-device. **No data leaves your machine.**
- 2FA keys are encrypted with a key derived from your master password and stored in the user
  config directory (`%APPDATA%\DevKit\totp.dat`). Data stored next to the old exe is migrated
  automatically on startup. **If you forget the master password, your keys cannot be recovered.**

## 📄 License

Released under the [MIT License](./LICENSE). © 2026 1204mm.