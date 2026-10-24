// soffice 单源（F2 蒸馏刀搬家）：LibreOffice 定位/转换/进程原语从
// internal/app/gaea_pdf.go 移入——只读渲染工具（doc_render）与 app 导出
// 共用同一份实现，app 侧保留同名薄委托。行为逐字节等价（搬家不改语义）。
package docmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/proc"
)

// FindSoffice 定位 soffice：PATH 优先，再补常见安装位置（Windows 下 PATH
// 通常没有 soffice）。
func FindSoffice() string {
	if p, err := exec.LookPath("soffice"); err == nil {
		return p
	}
	var cands []string
	switch runtime.GOOS {
	case "windows":
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			cands = append(cands, filepath.Join(local, "Programs", "LibreOffice", "program", "soffice.exe"))
		}
		for _, pf := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")} {
			if pf != "" {
				cands = append(cands, filepath.Join(pf, "LibreOffice", "program", "soffice.exe"))
			}
		}
	case "darwin":
		cands = append(cands, "/Applications/LibreOffice.app/Contents/MacOS/soffice")
	default:
		cands = append(cands, "/usr/lib/libreoffice/program/soffice", "/opt/libreoffice/program/soffice")
	}
	for _, c := range cands {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}
	return ""
}

// ConvertToPdfFile 用 soffice 把单个文档转成 outPath 指定的 PDF。
// soffice 以源文件名命名输出，所以先转进临时目录再搬到 outPath。
func ConvertToPdfFile(src, outPath string) error {
	soffice := FindSoffice()
	if soffice == "" {
		return fmt.Errorf("未找到 LibreOffice（soffice），请安装 LibreOffice 后重试")
	}
	tmpOut, err := os.MkdirTemp("", "gaea-pdf-out-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmpOut) }()
	profile, err := os.MkdirTemp("", "gaea-soffice-profile-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(profile) }()
	// 独立 profile：与 recalc/用户已开的 LibreOffice 互不抢锁
	profileURL := "file:///" + strings.ReplaceAll(filepath.ToSlash(profile), " ", "%20")

	args := []string{
		"-env:UserInstallation=" + profileURL,
		"--headless", "--norestore", "--nolockcheck", "--nologo",
		"--convert-to", "pdf", "--outdir", tmpOut, src,
	}
	if err := RunProcess(soffice, args, 180); err != nil {
		return fmt.Errorf("PDF 转换失败: %w", err)
	}
	produced := filepath.Join(tmpOut, strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))+".pdf")
	if _, err := os.Stat(produced); err != nil {
		return fmt.Errorf("PDF 未生成（源文件可能无法被 LibreOffice 打开）")
	}
	return MoveOrCopyFile(produced, outPath)
}

// MoveOrCopyFile 搬运文件：同盘 rename，跨盘回退复制（临时目录可能在
// 系统盘而工作区在数据盘）。
func MoveOrCopyFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// RunProcess 带超时地运行外部命令（stderr 合并进错误信息）。
func RunProcess(exe string, args []string, timeoutSec int) error {
	cmd := exec.Command(exe, args...)
	proc.HideWindow(cmd) // Windows: 防止弹出 cmd 黑框
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				return fmt.Errorf("%v（%s）", err, truncateFor(stderr.String(), 1500))
			}
			return err
		}
		return nil
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		_ = cmd.Process.Kill()
		<-done
		return fmt.Errorf("超时（%d 秒）", timeoutSec)
	}
}

func truncateFor(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
