package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"hash"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	_ "golang.org/x/image/bmp"

	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sys/windows"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// PortInfo 端口信息结构体
type PortInfo struct {
	Port     uint32 `json:"port"`
	PID      int32  `json:"pid"`
	ProcName string `json:"procName"`
	Project  string `json:"project"`
}

// App struct
type App struct {
	ctx          context.Context
	totpPassword string // 内存中缓存的解锁密码
	totpUnlocked bool   // 是否已解锁
}

// NewApp 创建新的App实例
func NewApp() *App {
	return &App{}
}

// startup 应用启动时调用
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	totpMigrateLegacy()
}

// IsAdmin 检测是否管理员权限
func (a *App) IsAdmin() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)
	token := windows.Token(0)
	enabled, err := token.IsMember(sid)
	return err == nil && enabled
}

// ScanPorts 扫描所有监听端口，返回端口信息列表。在后台goroutine执行，不阻塞。
func (a *App) ScanPorts() []PortInfo {
	ch := make(chan []PortInfo, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				ch <- []PortInfo{}
			}
		}()
		result := a.scanPortsInternal()
		// Java进程排前面
		sort.Slice(result, func(i, j int) bool {
			ii := strings.Contains(strings.ToLower(result[i].ProcName), "java")
			jj := strings.Contains(strings.ToLower(result[j].ProcName), "java")
			if ii != jj {
				return ii
			}
			return result[i].Port < result[j].Port
		})
		ch <- result
	}()
	select {
	case result := <-ch:
		return result
	case <-time.After(15 * time.Second):
		return []PortInfo{}
	}
}

func (a *App) scanPortsInternal() []PortInfo {
	// 同时扫描 tcp4 和 tcp6，确保不遗漏
	var allConns []net.ConnectionStat
	if conns, err := net.Connections("tcp4"); err == nil {
		allConns = append(allConns, conns...)
	}
	if conns, err := net.Connections("tcp6"); err == nil {
		allConns = append(allConns, conns...)
	}

	portMap := make(map[int32][]uint32)
	seen := make(map[string]bool)
	for _, conn := range allConns {
		if conn.Status == "LISTEN" && conn.Pid > 0 {
			key := fmt.Sprintf("%d-%d", conn.Pid, conn.Laddr.Port)
			if seen[key] {
				continue
			}
			seen[key] = true
			portMap[conn.Pid] = append(portMap[conn.Pid], conn.Laddr.Port)
		}
	}

	var result []PortInfo
	for pid, ports := range portMap {
		p, err := process.NewProcess(pid)
		if err != nil {
			continue
		}
		name, _ := p.Name()
		if name == "" {
			continue
		}
		// 只保留 java / nginx / node / python 服务
		lowerName := strings.ToLower(name)
		if !strings.Contains(lowerName, "java") &&
			!strings.Contains(lowerName, "nginx") &&
			!strings.Contains(lowerName, "node") &&
			!strings.Contains(lowerName, "python") {
			continue
		}
		project := a.getProjectInfo(p, name)
		for _, port := range ports {
			result = append(result, PortInfo{
				Port:     port,
				PID:      pid,
				ProcName: name,
				Project:  project,
			})
		}
	}
	return result
}

func (a *App) getProjectInfo(p *process.Process, procName string) string {
	lower := strings.ToLower(procName)
	switch {
	case strings.Contains(lower, "java"):
		return a.extractJavaInfo(p)
	case strings.Contains(lower, "node"):
		return a.extractNodeInfo(p)
	case strings.Contains(lower, "nginx"):
		return "nginx"
	case strings.Contains(lower, "python"):
		return a.extractPythonInfo(p)
	}
	cmdline, err := p.Cmdline()
	if err != nil {
		return ""
	}
	lowerCmd := strings.ToLower(cmdline)
	if strings.Contains(lowerCmd, "nginx") {
		return "nginx"
	}
	if strings.Contains(lowerCmd, "java") || strings.Contains(lowerCmd, "-jar") {
		return a.extractJavaInfo(p)
	}
	if strings.Contains(lowerCmd, "node") {
		return a.extractNodeInfo(p)
	}
	if strings.Contains(lowerCmd, "python") {
		return a.extractPythonInfo(p)
	}
	return ""
}

func (a *App) extractJavaInfo(p *process.Process) string {
	cmdline, err := p.Cmdline()
	if err != nil {
		return "Java"
	}
	parts := strings.Fields(cmdline)
	for i, part := range parts {
		if part == "-jar" && i+1 < len(parts) {
			jarPath := parts[i+1]
			return fmt.Sprintf("JAR: %s", filepath.Base(jarPath))
		}
	}
	// 没有-jar，取第一个不以-开头、不以@开头的参数作为主类
	for _, part := range parts {
		lower := strings.ToLower(part)
		if !strings.HasPrefix(part, "-") &&
			!strings.HasPrefix(part, "@") &&
			!strings.EqualFold(part, "java") &&
			!strings.HasSuffix(lower, "javaw.exe") &&
			!strings.HasSuffix(lower, "java.exe") {
			return fmt.Sprintf("Main: %s", part)
		}
	}
	return "Java"
}

func (a *App) extractNodeInfo(p *process.Process) string {
	cmdline, err := p.Cmdline()
	if err != nil {
		return "Node.js"
	}
	parts := strings.Fields(cmdline)
	for _, part := range parts {
		if strings.HasSuffix(part, ".js") || strings.HasSuffix(part, ".mjs") {
			return fmt.Sprintf("Script: %s", filepath.Base(part))
		}
	}
	return "Node.js"
}

func (a *App) extractPythonInfo(p *process.Process) string {
	cmdline, err := p.Cmdline()
	if err != nil {
		return "Python"
	}
	parts := strings.Fields(cmdline)
	for _, part := range parts {
		// 跳过 python.exe 本身和 -m 等参数
		lower := strings.ToLower(part)
		if strings.HasSuffix(lower, "python.exe") ||
			strings.HasSuffix(lower, "pythonw.exe") ||
			strings.HasPrefix(part, "-") {
			continue
		}
		if strings.HasSuffix(part, ".py") || strings.HasSuffix(part, ".wsgi") {
			return fmt.Sprintf("Script: %s", filepath.Base(part))
		}
		// -m module 模式
	}
	// 检查 -m module 模式
	for i, part := range parts {
		if part == "-m" && i+1 < len(parts) {
			return fmt.Sprintf("Module: %s", parts[i+1])
		}
	}
	return "Python"
}

// KillPid 杀死指定PID进程
func (a *App) KillPid(pid int32) error {
	defer func() {
		recover()
	}()
	return killProcessTree(pid)
}

// KillPort 杀死占用指定端口的进程
func (a *App) KillPort(port uint32) error {
	defer func() {
		recover()
	}()
	if port < 1 || port > 65535 {
		return fmt.Errorf("端口号无效")
	}
	pids := findPidsByPort(port)
	if len(pids) == 0 {
		return fmt.Errorf("端口 %d 没有找到占用进程", port)
	}
	var errs []string
	for _, pid := range pids {
		if err := killProcessTree(pid); err != nil {
			errs = append(errs, fmt.Sprintf("PID %d: %v", pid, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "；"))
	}
	return nil
}

// findPidsByPort 查找占用指定端口的进程PID，优先返回监听中的，没有监听则返回其他占用的
func findPidsByPort(port uint32) []int32 {
	var listenPids, otherPids []int32
	seenListen := make(map[int32]bool)
	seenOther := make(map[int32]bool)
	for _, proto := range []string{"tcp4", "tcp6"} {
		conns, err := net.Connections(proto)
		if err != nil {
			continue
		}
		for _, conn := range conns {
			if conn.Pid <= 0 || conn.Laddr.Port != port {
				continue
			}
			if conn.Status == "LISTEN" {
				if !seenListen[conn.Pid] {
					seenListen[conn.Pid] = true
					listenPids = append(listenPids, conn.Pid)
				}
			} else if !seenOther[conn.Pid] {
				seenOther[conn.Pid] = true
				otherPids = append(otherPids, conn.Pid)
			}
		}
	}
	if len(listenPids) > 0 {
		return listenPids
	}
	return otherPids
}

// killProcessTree 用 taskkill /F /T 强制杀死进程树，隐藏CMD窗口
func killProcessTree(pid int32) error {
	cmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmd.CombinedOutput()
	if err != nil {
		outStr := string(output)
		// taskkill 对已退出的进程会返回 "进程不存在" 之类的错误，实际已经死了
		if strings.Contains(outStr, "not found") ||
			strings.Contains(outStr, "找不到") ||
			strings.Contains(outStr, "不存在") ||
			strings.Contains(outStr, "没有找到") {
			// 进程可能已经退出，不算错误
			return nil
		}
		// 权限不足（如系统进程），提示以管理员身份运行
		if strings.Contains(outStr, "拒绝访问") ||
			strings.Contains(outStr, "Access is denied") {
			return fmt.Errorf("权限不足，无法结束该进程，请以管理员身份运行本程序后重试")
		}
		return fmt.Errorf("%s: %v", strings.TrimSpace(outStr), err)
	}
	return nil
}

// ===================== 加密解密工具 =====================

// Hash 计算哈希值，支持 MD5/SHA1/SHA256/SHA512
func (a *App) Hash(algorithm string, text string) string {
	defer func() {
		recover()
	}()
	switch strings.ToUpper(algorithm) {
	case "MD5":
		h := md5.Sum([]byte(text))
		return hex.EncodeToString(h[:])
	case "SHA1":
		h := sha1.Sum([]byte(text))
		return hex.EncodeToString(h[:])
	case "SHA256":
		h := sha256.Sum256([]byte(text))
		return hex.EncodeToString(h[:])
	case "SHA512":
		h := sha512.Sum512([]byte(text))
		return hex.EncodeToString(h[:])
	}
	return ""
}

// AESEncrypt AES-CBC 加密，返回 Base64
func (a *App) AESEncrypt(plaintext string, key string) (string, error) {
	defer func() {
		recover()
	}()
	keyBytes := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(keyBytes[:])
	if err != nil {
		return "", err
	}
	padded := pkcs7Pad([]byte(plaintext), block.BlockSize())
	iv := make([]byte, block.BlockSize())
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	encrypted := make([]byte, len(padded))
	mode.CryptBlocks(encrypted, padded)
	// IV + 密文 拼接后 Base64
	result := append(iv, encrypted...)
	return base64.StdEncoding.EncodeToString(result), nil
}

// AESDecrypt AES-CBC 解密
func (a *App) AESDecrypt(ciphertextB64 string, key string) (string, error) {
	defer func() {
		recover()
	}()
	raw, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}
	keyBytes := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(keyBytes[:])
	if err != nil {
		return "", err
	}
	if len(raw) < block.BlockSize() {
		return "", fmt.Errorf("密文太短")
	}
	iv := raw[:block.BlockSize()]
	encrypted := raw[block.BlockSize():]
	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(encrypted))
	mode.CryptBlocks(decrypted, encrypted)
	unpadded, err := pkcs7Unpad(decrypted, block.BlockSize())
	if err != nil {
		return "", err
	}
	return string(unpadded), nil
}

// Base64Encode Base64 编码
func (a *App) Base64Encode(text string) string {
	return base64.StdEncoding.EncodeToString([]byte(text))
}

// Base64Decode Base64 解码
func (a *App) Base64Decode(text string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// HexEncode Hex 编码
func (a *App) HexEncode(text string) string {
	return hex.EncodeToString([]byte(text))
}

// HexDecode Hex 解码
func (a *App) HexDecode(text string) (string, error) {
	data, err := hex.DecodeString(text)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := strings.Repeat(string(rune(padding)), padding)
	return append(data, []byte(padText)...)
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("空数据")
	}
	padding := int(data[len(data)-1])
	if padding <= 0 || padding > blockSize {
		return nil, fmt.Errorf("无效填充")
	}
	return data[:len(data)-padding], nil
}

// Hmac 计算 HMAC，支持 HMAC-SHA256 / HMAC-SHA512 / HMAC-MD5
func (a *App) Hmac(algorithm string, text string, key string) string {
	defer func() {
		recover()
	}()
	var h func() hash.Hash
	switch strings.ToUpper(algorithm) {
	case "SHA256":
		h = sha256.New
	case "SHA512":
		h = sha512.New
	case "MD5":
		h = md5.New
	default:
		return ""
	}
	mac := hmac.New(h, []byte(key))
	mac.Write([]byte(text))
	return hex.EncodeToString(mac.Sum(nil))
}

// BcryptHash bcrypt 加密
func (a *App) BcryptHash(password string, cost int) (string, error) {
	defer func() {
		recover()
	}()
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// BcryptCompare bcrypt 校验，返回 true=匹配
func (a *App) BcryptCompare(password string, hashed string) bool {
	defer func() {
		recover()
	}()
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))
	return err == nil
}

// DesEncrypt DES-CBC 加密，返回 Base64
func (a *App) DesEncrypt(plaintext string, key string) (string, error) {
	defer func() {
		recover()
	}()
	// DES 密钥固定 8 字节，取前 8 位
	keyBytes := []byte(key)
	if len(keyBytes) < 8 {
		keyBytes = append(keyBytes, strings.Repeat("0", 8-len(keyBytes))...)
	}
	keyBytes = keyBytes[:8]
	block, err := des.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}
	padded := pkcs7Pad([]byte(plaintext), block.BlockSize())
	iv := make([]byte, block.BlockSize())
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	encrypted := make([]byte, len(padded))
	mode.CryptBlocks(encrypted, padded)
	result := append(iv, encrypted...)
	return base64.StdEncoding.EncodeToString(result), nil
}

// DesDecrypt DES-CBC 解密
func (a *App) DesDecrypt(ciphertextB64 string, key string) (string, error) {
	defer func() {
		recover()
	}()
	raw, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}
	keyBytes := []byte(key)
	if len(keyBytes) < 8 {
		keyBytes = append(keyBytes, strings.Repeat("0", 8-len(keyBytes))...)
	}
	keyBytes = keyBytes[:8]
	block, err := des.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}
	if len(raw) < block.BlockSize() {
		return "", fmt.Errorf("密文太短")
	}
	iv := raw[:block.BlockSize()]
	encrypted := raw[block.BlockSize():]
	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(encrypted))
	mode.CryptBlocks(decrypted, encrypted)
	unpadded, err := pkcs7Unpad(decrypted, block.BlockSize())
	if err != nil {
		return "", err
	}
	return string(unpadded), nil
}

// RSAGenerateKey 生成 RSA 密钥对（2048位），返回 PEM 格式
func (a *App) RSAGenerateKey() (string, error) {
	defer func() {
		recover()
	}()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", err
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})
	return string(privatePEM) + "\n" + string(publicPEM), nil
}

// RSAEncrypt RSA 公钥加密，返回 Base64
func (a *App) RSAEncrypt(plaintext string, publicKeyPEM string) (string, error) {
	defer func() {
		recover()
	}()
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return "", fmt.Errorf("无效的公钥PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}
	pubKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("不是RSA公钥")
	}
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, []byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// RSADecrypt RSA 私钥解密
func (a *App) RSADecrypt(ciphertextB64 string, privateKeyPEM string) (string, error) {
	defer func() {
		recover()
	}()
	raw, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", fmt.Errorf("无效的私钥PEM")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, priv, raw)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}

// URLEncode URL 编码
func (a *App) URLEncode(text string) string {
	return url.QueryEscape(text)
}

// URLDecode URL 解码
func (a *App) URLDecode(text string) (string, error) {
	return url.QueryUnescape(text)
}

// ===================== 2FA TOTP =====================

// TOTPSecret TOTP 密钥项
type TOTPSecret struct {
	Name   string `json:"name"`
	Secret string `json:"secret"` // base32 密钥（明文，运行时解密后）
	Issuer string `json:"issuer"`
}

// TOTPCode TOTP 验证码结果
type TOTPCode struct {
	Name   string `json:"name"`
	Issuer string `json:"issuer"`
	Code   string `json:"code"`
	Remain int    `json:"remain"` // 当前周期剩余秒数
}

// totpExeDir 获取 exe 同级目录（旧版数据位置，用于迁移）
func totpExeDir() string {
	exe, err := os.Executable()
	if err != nil {
		dir, _ := os.Getwd()
		return dir
	}
	return filepath.Dir(exe)
}

// totpDataDir TOTP 数据目录：用户配置目录（%APPDATA%\DevKit）
func totpDataDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return totpExeDir() // 获取失败时退回 exe 目录
	}
	return filepath.Join(base, "DevKit")
}

// totpStoragePath TOTP 密钥加密存储文件
func totpStoragePath() string {
	return filepath.Join(totpDataDir(), "totp.dat")
}

// totpKeyPath 密码校验文件（bcrypt 哈希）
func totpKeyPath() string {
	return filepath.Join(totpDataDir(), "totp.key")
}

// totpMigrateLegacy 旧版把 TOTP 数据存在 exe 同目录，迁移到用户配置目录（每次启动检查，幂等）
func totpMigrateLegacy() {
	newDir := totpDataDir()
	oldDir := totpExeDir()
	if newDir == oldDir {
		return
	}
	if err := os.MkdirAll(newDir, 0700); err != nil {
		return
	}
	for _, name := range []string{"totp.dat", "totp.key"} {
		newPath := filepath.Join(newDir, name)
		if _, err := os.Stat(newPath); err == nil {
			continue // 新位置已有数据，不覆盖
		}
		data, err := os.ReadFile(filepath.Join(oldDir, name))
		if err != nil {
			continue
		}
		_ = os.WriteFile(newPath, data, 0600)
	}
}

// totpDeriveKey 从用户密码派生 AES 密钥
func totpDeriveKey(password string) []byte {
	h := sha256.Sum256([]byte("DevKit-TOTP-" + password))
	return h[:]
}

// TOTPIsPasswordSet 检查是否已设置密码
func (a *App) TOTPIsPasswordSet() bool {
	defer func() {
		recover()
	}()
	_, err := os.Stat(totpKeyPath())
	return err == nil
}

// TOTPSetupPassword 首次设置密码
func (a *App) TOTPSetupPassword(password string) error {
	defer func() {
		recover()
	}()
	if len(password) < 4 {
		return fmt.Errorf("密码至少4位")
	}
	if a.TOTPIsPasswordSet() {
		return fmt.Errorf("密码已设置，不可重复设置")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := os.WriteFile(totpKeyPath(), hashed, 0600); err != nil {
		return err
	}
	a.totpPassword = password
	a.totpUnlocked = true
	return nil
}

// TOTPUnlock 验证密码解锁
func (a *App) TOTPUnlock(password string) bool {
	defer func() {
		recover()
	}()
	hashed, err := os.ReadFile(totpKeyPath())
	if err != nil {
		return false
	}
	if bcrypt.CompareHashAndPassword(hashed, []byte(password)) != nil {
		return false
	}
	a.totpPassword = password
	a.totpUnlocked = true
	return true
}

// TOTPLock 锁定（退出 2FA 页面时调用）
func (a *App) TOTPLock() {
	a.totpPassword = ""
	a.totpUnlocked = false
}

// TOTPIsUnlocked 检查是否已解锁
func (a *App) TOTPIsUnlocked() bool {
	return a.totpUnlocked
}

// TOTPAddSecret 添加 TOTP 密钥（加密存储）
func (a *App) TOTPAddSecret(name string, secret string, issuer string) error {
	defer func() {
		recover()
	}()
	if !a.totpUnlocked {
		return fmt.Errorf("请先输入密码解锁")
	}
	secret = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(secret), " ", ""))
	if name == "" || secret == "" {
		return fmt.Errorf("名称和密钥不能为空")
	}
	// 验证 base32
	if _, err := base32.StdEncoding.DecodeString(secret); err != nil {
		return fmt.Errorf("密钥不是有效的Base32格式: %v", err)
	}

	storage := a.totpLoadStorage()
	// 去重
	for i, s := range storage {
		if s.Name == name {
			storage[i].Secret = secret
			storage[i].Issuer = issuer
			return a.totpSaveStorage(storage)
		}
	}
	storage = append(storage, TOTPSecret{Name: name, Secret: secret, Issuer: issuer})
	return a.totpSaveStorage(storage)
}

// TOTPListSecrets 列出所有 TOTP 密钥
func (a *App) TOTPListSecrets() []TOTPSecret {
	defer func() {
		recover()
	}()
	if !a.totpUnlocked {
		return []TOTPSecret{}
	}
	return a.totpLoadStorage()
}

// TOTPDeleteSecret 删除 TOTP 密钥
func (a *App) TOTPDeleteSecret(name string) error {
	defer func() {
		recover()
	}()
	if !a.totpUnlocked {
		return fmt.Errorf("请先输入密码解锁")
	}
	storage := a.totpLoadStorage()
	for i, s := range storage {
		if s.Name == name {
			storage = append(storage[:i], storage[i+1:]...)
			return a.totpSaveStorage(storage)
		}
	}
	return fmt.Errorf("未找到名称为 %s 的密钥", name)
}

// TOTPGenerateAll 生成所有 TOTP 验证码
func (a *App) TOTPGenerateAll() []TOTPCode {
	defer func() {
		recover()
	}()
	if !a.totpUnlocked {
		return []TOTPCode{}
	}
	storage := a.totpLoadStorage()
	var codes []TOTPCode
	now := time.Now().Unix()
	remain := 30 - int(now%30)
	for _, s := range storage {
		code := totpCompute(s.Secret, now)
		codes = append(codes, TOTPCode{
			Name:   s.Name,
			Issuer: s.Issuer,
			Code:   code,
			Remain: remain,
		})
	}
	return codes
}

// totpCompute 计算 TOTP（RFC 6238）
func totpCompute(base32Secret string, timestamp int64) string {
	key, err := base32.StdEncoding.DecodeString(base32Secret)
	if err != nil {
		return "------"
	}
	counter := timestamp / 30
	buf := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		buf[i] = byte(counter & 0xFF)
		counter >>= 8
	}
	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	hash := mac.Sum(nil)
	offset := int(hash[len(hash)-1] & 0x0F)
	bin := ((int(hash[offset]) & 0x7F) << 24) |
		((int(hash[offset+1]) & 0xFF) << 16) |
		((int(hash[offset+2]) & 0xFF) << 8) |
		(int(hash[offset+3]) & 0xFF)
	code := bin % 1000000
	return fmt.Sprintf("%06d", code)
}

// totpLoadStorage 加载并解密存储
func (a *App) totpLoadStorage() []TOTPSecret {
	path := totpStoragePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return []TOTPSecret{}
	}
	key := totpDeriveKey(a.totpPassword)
	block, err := aes.NewCipher(key)
	if err != nil {
		return []TOTPSecret{}
	}
	if len(data) < block.BlockSize() {
		return []TOTPSecret{}
	}
	iv := data[:block.BlockSize()]
	encrypted := data[block.BlockSize():]
	if len(encrypted)%block.BlockSize() != 0 {
		return []TOTPSecret{}
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(encrypted))
	mode.CryptBlocks(decrypted, encrypted)
	unpadded, err := pkcs7Unpad(decrypted, block.BlockSize())
	if err != nil {
		return []TOTPSecret{}
	}
	var storage []TOTPSecret
	if err := json.Unmarshal(unpadded, &storage); err != nil {
		return []TOTPSecret{}
	}
	return storage
}

// totpSaveStorage 加密并保存存储
func (a *App) totpSaveStorage(storage []TOTPSecret) error {
	path := totpStoragePath()
	data, err := json.Marshal(storage)
	if err != nil {
		return err
	}
	key := totpDeriveKey(a.totpPassword)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	padded := pkcs7Pad(data, block.BlockSize())
	iv := make([]byte, block.BlockSize())
	if _, err := rand.Read(iv); err != nil {
		return err
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	encrypted := make([]byte, len(padded))
	mode.CryptBlocks(encrypted, padded)
	result := append(iv, encrypted...)
	return os.WriteFile(path, result, 0600)
}

// ===================== IconForge 图标生成 =====================

// ImageInfo 前端拿到的图片信息
type ImageInfo struct {
	Name           string `json:"name"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	DataURL        string `json:"dataUrl"`
	DetectedRadius int    `json:"detectedRadius"` // 自动识别的圆角半径
	CornerDetected bool   `json:"cornerDetected"` // 是否识别到圆角外背景
}

// ExportParams 导出 ICO 的参数
type ExportParams struct {
	ImageData    string `json:"imageData"`    // 原图 dataURL
	CropX        int    `json:"cropX"`        // 裁剪框（原图坐标）
	CropY        int    `json:"cropY"`
	CropSize     int    `json:"cropSize"`
	CornerRadius int    `json:"cornerRadius"` // 圆角半径，0=不切
	Sizes        []int  `json:"sizes"`        // ICO 内包含的尺寸
}

// loadImageByPath 读取并解析图片文件
func loadImageByPath(path string) (*ImageInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("不支持的图片格式（支持 PNG / JPG / BMP / GIF）: %v", err)
	}
	bounds := img.Bounds()
	mime := "image/" + format
	if format == "jpg" {
		mime = "image/jpeg"
	}

	// 自动识别圆角
	radius, detected := DetectCornerRadius(img)

	return &ImageInfo{
		Name:           filepath.Base(path),
		Width:          bounds.Dx(),
		Height:         bounds.Dy(),
		DataURL:        "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data),
		DetectedRadius: radius,
		CornerDetected: detected,
	}, nil
}

// OpenImageDialog 弹出文件选择框选择图片，用户取消时返回 null
func (a *App) OpenImageDialog() (*ImageInfo, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择图片",
		Filters: []runtime.FileFilter{
			{DisplayName: "图片文件 (*.png;*.jpg;*.jpeg;*.bmp;*.gif)", Pattern: "*.png;*.jpg;*.jpeg;*.bmp;*.gif"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}
	return loadImageByPath(path)
}

// LoadImageByPath 按路径加载图片（拖拽场景）
func (a *App) LoadImageByPath(path string) (*ImageInfo, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".bmp", ".gif":
	default:
		return nil, fmt.Errorf("不支持的文件类型: %s", ext)
	}
	return loadImageByPath(path)
}

// ExportIcons 导出图标 ZIP：每个尺寸一个独立 ICO + 一个多尺寸合一的 icon.ico，打包为 zip
// 返回保存路径（用户取消返回空）
func (a *App) ExportIcons(p ExportParams) (string, error) {
	if len(p.Sizes) == 0 {
		return "", fmt.Errorf("请至少选择一个 ICO 尺寸")
	}

	// 解码原图
	idx := strings.Index(p.ImageData, "base64,")
	if idx < 0 {
		return "", fmt.Errorf("图片数据无效")
	}
	raw, err := base64.StdEncoding.DecodeString(p.ImageData[idx+7:])
	if err != nil {
		return "", fmt.Errorf("图片数据解码失败: %v", err)
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("图片解码失败: %v", err)
	}

	// 尺寸从小到大排序，保证 zip 内文件顺序稳定
	sizes := append([]int{}, p.Sizes...)
	sort.Ints(sizes)

	// 打包：icon.ico（多尺寸合一，Windows 直接可用） + icon_<size>.ico（各尺寸独立文件）
	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)

	addEntry := func(name string, data []byte) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}

	// 多尺寸合一
	combined, err := BuildIconPipeline(src, p.CropX, p.CropY, p.CropSize, p.CornerRadius, sizes)
	if err != nil {
		return "", fmt.Errorf("生成 ICO 失败: %v", err)
	}
	if err := addEntry("icon.ico", combined); err != nil {
		return "", fmt.Errorf("写入 ZIP 失败: %v", err)
	}

	// 各尺寸独立文件
	for _, s := range sizes {
		one, err := BuildIconPipeline(src, p.CropX, p.CropY, p.CropSize, p.CornerRadius, []int{s})
		if err != nil {
			return "", fmt.Errorf("生成 %dpx ICO 失败: %v", s, err)
		}
		if err := addEntry(fmt.Sprintf("icon_%d.ico", s), one); err != nil {
			return "", fmt.Errorf("写入 ZIP 失败: %v", err)
		}
	}

	if err := zw.Close(); err != nil {
		return "", fmt.Errorf("打包 ZIP 失败: %v", err)
	}

	// 保存对话框
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "保存图标压缩包",
		DefaultFilename: "icons.zip",
		Filters: []runtime.FileFilter{
			{DisplayName: "ZIP 压缩包 (*.zip)", Pattern: "*.zip"},
		},
	})
	if err != nil {
		return "", err
	}
	if savePath == "" {
		return "", nil // 用户取消
	}
	if !strings.EqualFold(filepath.Ext(savePath), ".zip") {
		savePath += ".zip"
	}

	if err := os.WriteFile(savePath, zipBuf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %v", err)
	}
	return savePath, nil
}
