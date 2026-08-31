<p align="center">
  <img src="icon-1024-rounded.png" width="128" alt="DevKit" />
</p>

<h1 align="center">DevKit - 程序员工具箱</h1>

<p align="center">
  基于 <a href="https://wails.io">Wails v2</a> 构建的 Windows 桌面开发工具箱（Go + Vue 3）
</p>

<p align="center">
  <a href="./README.en.md"><img alt="English" src="https://img.shields.io/badge/English-Documentation-blue"></a>
  <a href="./LICENSE"><img alt="License: MIT" src="https://img.shields.io/badge/License-MIT-yellow.svg"></a>
  <img alt="Platform: Windows" src="https://img.shields.io/badge/Platform-Windows%2010%2F11-0078D6.svg">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25-00ADD8.svg">
  <img alt="Wails v2" src="https://img.shields.io/badge/Wails-v2-F7DF1E.svg">
  <img alt="Vue 3" src="https://img.shields.io/badge/Vue-3-42B883.svg">
</p>

> 一站式开发工具箱：端口管理、加密解密、TOTP 二次验证、JSON 格式化，以及**带圆角自动识别的图标生成器**。

## ✨ 功能

### 🔌 端口管理

- 一键扫描本机所有监听端口（tcp4 / tcp6），自动识别 Java / Node / Nginx / Python 进程
- 智能项目识别：自动提取 Java 主类、JAR 包名、Node 脚本名，一眼看出端口被哪个项目占用
- 选中后强制杀死进程（`taskkill /F /T` 结束整个进程树）
- 手动输入端口号，直接杀掉占用该端口的进程
- 杀进程失败（权限不足）时才提示以管理员身份运行，不弹多余警告

### 🔐 加密解密

- 哈希：MD5 / SHA1 / SHA256 / SHA512
- HMAC：HMAC-MD5 / HMAC-SHA256 / HMAC-SHA512
- 对称加密：AES-CBC、DES-CBC（Base64 输入输出，IV 随机生成）
- 非对称加密：RSA 2048 密钥生成 / 加密 / 解密
- 编码转换：Base64、Hex、URL 编解码
- 密码学哈希：bcrypt 生成与校验

### 🔢 2FA 验证码

- 标准 TOTP（RFC 6238），兼容 Google Authenticator
- 主密码保护：密钥使用 AES 加密存储在本地（`totp.dat`），主密码经 bcrypt 校验
- 验证码点击即复制，30 秒周期倒计时进度条

### 📋 JSON 格式化

- 格式化 / 压缩 / 校验，支持 2 / 4 空格缩进
- 一键复制、输出一键回填输入

### 🖼 IconForge — 图标生成

- PNG / JPG / BMP / GIF 一键转多尺寸 ICO（16 ~ 256px 自由勾选）
- **圆角自动识别**：自动检测图片四角的多余背景（如带浅色圆角外框的图标图），一键切除为透明，滑块微调半径
- 正方形裁剪：拖拽画布调整裁剪区域，实时多尺寸预览
- 圆角切割基于带符号距离场（SDF）抗锯齿蒙版，边缘平滑无锯齿；Lanczos 高质量缩放
- 导出 ZIP：各尺寸独立 ICO + 多尺寸合一 `icon.ico` 一次打包

### 🧰 常用工具

| 工具 | 说明 |
|------|------|
| 时间戳转换 | 当前时间戳实时显示（点击复制）；Unix 秒/毫秒自动识别 ↔ 本地时间 / 星期 / ISO 8601 / UTC；日期字符串 ↔ 时间戳 |
| JWT 解码 | 纯本地解码 Header / Payload，`exp` 过期状态实时提示（有效剩余时长 / 已过期），时间声明字段高亮，支持 Bearer 前缀 |
| Cron 表达式 | 支持标准 5 段与 Quartz 6~7 段（`?`、`L`、`#`、`MON-FRI`、年份字段），中文语义描述 + 最近 5 次执行时间 + 字段拆解 |
| UUID 生成 | 批量生成 v4 UUID（1~500 个），支持去连字符 / 大写，单条点击复制 |
| 正则测试 | 实时高亮匹配，g / i / m / s / u 标志，匹配详情含捕获组与位置区间 |

## 🛠 技术栈

| 层 | 技术 |
|----|------|
| 桌面框架 | [Wails v2](https://wails.io) |
| 后端 | Go（[gopsutil](https://github.com/shirou/gopsutil) 读取端口与进程，[imaging](https://github.com/disintegration/imaging) 图像处理） |
| 前端 | Vue 3 + TypeScript + Vite |
| UI | Ant Design Vue 4（Catppuccin Mocha 深色主题定制） |
| Cron 解析 | [cron-parser](https://github.com/pentestfunctions/cron-parser) + [cronstrue](https://github.com/bradymholt/cronstrue)（中文描述） |
| 图标算法 | 圆角识别（对角线/边缘扫描 + 最小二乘圆拟合）、SDF 抗锯齿蒙版、Lanczos 缩放、ICO 编码（BMP+PNG 双格式），含单元测试 |

## 🚀 开发与构建

环境要求：Go 1.21+、Node.js 18+、[Wails CLI](https://wails.io/docs/gettingstarted/installation) v2，仅支持 Windows。

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 开发模式（前端热重载）
wails dev

# 生产构建，输出 build/bin/portmanager.exe
wails build
```

## ⬇️ 下载

前往 **[Releases](https://github.com/1204mm/devkit-toolbox/releases)** 页面获取最新版本（单文件 `portmanager.exe`，免安装）。

## 📄 说明

- 本工具仅操作本机进程与本地文件，JWT 解码、时间戳转换等均在纯前端完成，不发送任何数据
- 2FA 密钥经主密码派生密钥加密后存储在用户配置目录（`%APPDATA%\DevKit\totp.dat`），旧版本存放在 exe 同目录的数据会在启动时自动迁移，忘记主密码无法恢复

## 📄 开源协议

基于 [MIT License](./LICENSE) 发布。© 2026 1204mm。