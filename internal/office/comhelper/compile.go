package comhelper

// compile.go — 运行时编译 helper：Windows 自带的 .NET Framework csc.exe，
// 产物按源码 sha256 前 12 位缓存（与上游 helper-client.ts 同策略）。源码
// 无 BOM 且含中文字面量，编译固定加 /codepage:65001 强制 UTF-8 读入
//（doc.go 偏离 1）。编译一次约 1 秒，之后直接命中缓存。

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/proc"
)

// helperSource 由 go:embed 逐字节内嵌（与上游 cmp 一致，见 UPSTREAM.md）。
//
//go:embed OfficeHelper.cs
var helperSource []byte

// helperReferences 上游 REFERENCES 原样（csc 引用清单；JavaScriptSerializer 在
// System.Web.Extensions，卡片 UI 在 System.Windows.Forms）。
var helperReferences = []string{
	"System.dll", "System.Core.dll", "Microsoft.CSharp.dll", "System.Drawing.dll",
	"System.Windows.Forms.dll", "System.Web.Extensions.dll",
}

// SourceHash 内嵌源码的 sha256 前 12 位（缓存目录键；改源码即自然重编）。
func SourceHash() string {
	sum := sha256.Sum256(helperSource)
	return hex.EncodeToString(sum[:])[:12]
}

// cacheRootOverride 测试注入口（nil=正常路径）。
var cacheRootOverride func() string

// CacheRoot 编译缓存根：%LOCALAPPDATA%\gaea\office-helper（LOCALAPPDATA 缺失
// 回退系统临时目录——与上游同降级）。
func CacheRoot() string {
	if cacheRootOverride != nil {
		return cacheRootOverride()
	}
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "gaea", "office-helper")
}

// FindCsc 定位 .NET Framework 4 自带 C# 编译器（Framework64 优先，缺了退
// Framework）；两者都没有时给出上游同款报错指引。
func FindCsc() (string, error) {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = os.Getenv("windir")
	}
	if root == "" {
		root = `C:\Windows`
	}
	for _, arch := range []string{"Framework64", "Framework"} {
		dir := filepath.Join(root, "Microsoft.NET", arch, "v4.0.30319")
		csc := filepath.Join(dir, "csc.exe")
		if _, err := os.Stat(csc); err == nil {
			return csc, nil
		}
	}
	return "", fmt.Errorf("找不到 .NET Framework 4 自带的 C# 编译器（csc.exe）。Windows 10/11 默认自带；" +
		"请在「启用或关闭 Windows 功能」里勾选 .NET Framework 4.8 高级服务后重试")
}

var (
	buildMu         sync.Mutex
	builtExe        string // 已确认存在的产物路径（SourceHash 变化后失效）
	builtForHash    string
	buildErr        error
	buildInProgress bool
	buildDone       = make(chan struct{})
)

// EnsureHelperExe 返回编译好的 helper exe 路径：缓存命中直接返回；未命中
// 用 csc 编译（singleflight——并发调用只编一次，失败不缓存重试）。编译走
// 临时文件+rename，另一进程先赢时其产物字节相同，直接复用（上游同语义）。
func EnsureHelperExe() (string, error) {
	buildMu.Lock()
	if builtForHash == SourceHash() && buildErr == nil && builtExe != "" {
		buildMu.Unlock()
		return builtExe, nil
	}
	if buildInProgress {
		buildMu.Unlock()
		<-buildDone // 等在途编译收尾，再走一遍快路径取结果
		buildMu.Lock()
		defer buildMu.Unlock()
		if buildErr == nil && builtExe != "" {
			return builtExe, nil
		}
		return "", buildErr
	}
	buildInProgress = true
	buildMu.Unlock()

	exe, err := buildHelper()

	buildMu.Lock()
	buildInProgress = false
	builtExe, builtForHash, buildErr = exe, SourceHash(), err
	// 失败不缓存：下一次调用重试（首次编译失败常见于功能未启用，启用后即好）。
	if err != nil {
		builtForHash = ""
	}
	close(buildDone)
	buildDone = make(chan struct{})
	buildMu.Unlock()
	if err != nil {
		return "", err
	}
	return exe, nil
}

func buildHelper() (string, error) {
	csc, err := FindCsc()
	if err != nil {
		return "", err
	}
	hash := SourceHash()
	dir := filepath.Join(CacheRoot(), "helper-"+hash)
	exe := filepath.Join(dir, "office-helper.exe")
	if _, err := os.Stat(exe); err == nil {
		return exe, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建 helper 编译缓存目录失败: %w", err)
	}
	temp := filepath.Join(dir, fmt.Sprintf("build-%d-%d.exe", os.Getpid(), time.Now().UnixNano()))
	args := []string{
		"-nologo", "-optimize+", "-target:exe", "-out:" + temp,
		// doc.go 偏离 1：源文件无 BOM 且含中文字面量，强制按 UTF-8 读入。
		"-codepage:65001",
	}
	for _, ref := range helperReferences {
		args = append(args, "-r:"+ref)
	}
	srcPath := filepath.Join(dir, "OfficeHelper.cs")
	if err := os.WriteFile(srcPath, helperSource, 0o644); err != nil {
		return "", fmt.Errorf("写出 helper 源码失败: %w", err)
	}
	args = append(args, srcPath)

	cmd := exec.Command(csc, args...)
	proc.HideWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		detail := string(out)
		if detail == "" {
			detail = err.Error()
		}
		_ = os.Remove(temp)
		_ = os.Remove(srcPath)
		return "", fmt.Errorf("编译 office helper 失败: %s", detail)
	}
	// 源码副本在编译后即无用（缓存键是哈希目录，下次直接命中 exe）。
	_ = os.Remove(srcPath)
	if err := os.Rename(temp, exe); err != nil {
		// 另一进程赢了 rename 竞态：其产物与本次字节相同。
		_ = os.Remove(temp)
		if _, statErr := os.Stat(exe); statErr != nil {
			return "", fmt.Errorf("helper 编译产物落位失败: %w", err)
		}
	}
	return exe, nil
}
