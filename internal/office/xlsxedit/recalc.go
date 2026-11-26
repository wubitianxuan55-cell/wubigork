package xlsxedit

// recalc.go — LibreOffice 公式重算（2026-11-26 吸收重算脚本；宁丰反馈 #1）。
//
// 来历：重算链原先是「到工作区 .gaea/skills → ~/.codex/skills → ~/.gaea/skills 找
// recalc.py，再 python 调用」。脚本不是 gaea 的——各工作区各一份，改仓库那份修不到
// 别的工作区；用户机器上实际命中的是 Codex 技能目录里的副本。同批对照实验
// （28 次）另坐实一个比报错更糟的形态：soffice 走默认配置目录时，只要机器上另有
// 一个 LibreOffice 实例活着，本次调用就被转交给它并立刻返回，脚本读到没重算的
// 文件却回「成功」（6/6 复现）。
//
// 改后：脚本随二进制内嵌（scripts/recalc.py，自包含），每次调用复制一份独立的
// LibreOffice 配置目录去跑——不转交、并发互不踩踏，用完即删。首次调用让 LibreOffice
// 自己初始化一份配置（实测约 5 秒），跑成后留作模板；此后每次复制（约 0.1 秒）
// 加重算（约 1.4 秒），比原先走默认配置目录（约 1.9 秒）还快。

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/gaea/proc"
)

// recalcScript 随二进制内嵌的重算脚本：自包含，只依赖 python + openpyxl 与本机
// LibreOffice。它同时是配置模板的指纹来源（见 recalcTemplateDir）——改了脚本，
// 旧模板自然作废重建。
//
//go:embed scripts/recalc.py
var recalcScript string

// RecalcReport 是重算脚本的 JSON 输出。
type RecalcReport struct {
	Status        string                 `json:"status"`
	TotalErrors   int                    `json:"total_errors"`
	TotalFormulas int                    `json:"total_formulas"`
	ErrorSummary  map[string]interface{} `json:"error_summary"`
	// Error 非空即失败；Stage 标失败环节——recalc 是 LibreOffice 环节，read 是
	// 重算后回读工作簿（zip 解压类瞬时错误）。
	Error string `json:"error"`
	Stage string `json:"stage,omitempty"`
}

// 重算失败的三类，调用方用 errors.Is 判别：
//   - ErrRecalcUnavailable 环境不具备：缺 python / openpyxl / LibreOffice、宏没配上、
//     LibreOffice 超时或非零退出、重算被 GAEA_RECALC_DISABLE 关闭——原样重试没用；
//   - ErrRecalcRead 重算后回读工作簿失败——瞬时的，Recalc 已自动重试过一次仍失败；
//   - ErrRecalcNotWorkbook 文件不是能打开的 .xlsx（zip 结构就坏了）。
var (
	ErrRecalcUnavailable = errors.New("公式重算不可用")
	ErrRecalcRead        = errors.New("重算结果读取失败")
	ErrRecalcNotWorkbook = errors.New("不是能打开的 .xlsx")
)

const (
	// recalcDisableKey 非空即关闭重算：app 包测试进程置位，测试不拉起 LibreOffice。
	recalcDisableKey = "GAEA_RECALC_DISABLE"
	// recalcProfileKey 把本次调用的专属配置目录交给脚本。
	recalcProfileKey = "GAEA_RECALC_PROFILE"
	// recalcProfilePrefix 配置模板目录名前缀（后缀是脚本指纹）。
	recalcProfilePrefix = "lo-recalc-profile-"
	// recalcReadRetryDelay 回读失败后、重试前的等待（给文件系统收尾的时间）。
	recalcReadRetryDelay = 500 * time.Millisecond
)

// 测试注入口（零值 = 正常路径）。
var (
	recalcCacheRootOverride func() string
	recalcTempDir           = os.TempDir
	runRecalcScript         = execRecalcScript
	recalcSleep             = time.Sleep
)

// Recalc 用 LibreOffice 重算 path 里全部公式并写回缓存值（公式本身不动）。
// 失败分类见 RecalcReport 上方的三个 Err。
func Recalc(path string) (RecalcReport, error) {
	if os.Getenv(recalcDisableKey) != "" {
		return RecalcReport{}, fmt.Errorf("%w：已由 %s 关闭", ErrRecalcUnavailable, recalcDisableKey)
	}
	// zip 预检：打不开的文件交给 LibreOffice 会挂满 60 秒超时才回错（2026-11-26
	// 实测），不如在这里秒退。
	z, zerr := zip.OpenReader(path)
	if zerr != nil {
		return RecalcReport{}, fmt.Errorf("%w：%s（%v）", ErrRecalcNotWorkbook, filepath.Base(path), zerr)
	}
	_ = z.Close()
	rep, usedTemplate, err := recalcAttempt(path)
	if err == nil {
		return rep, nil
	}
	if !shouldRetryRecalc(err, usedTemplate) {
		return RecalcReport{}, err
	}
	if errors.Is(err, ErrRecalcRead) {
		recalcSleep(recalcReadRetryDelay)
	} else {
		_ = os.RemoveAll(recalcTemplateDir())
	}
	rep, _, err = recalcAttempt(path)
	if err != nil {
		return RecalcReport{}, fmt.Errorf("（已自动重试 1 次）%w", err)
	}
	return rep, nil
}

// shouldRetryRecalc 判定这次失败值不值得再来一次：回读错误是瞬时的；模板复制件
// 跑不起来多半是模板坏了（丢弃重建即可自愈）。全新配置都跑不起来 = 环境问题，不重试。
func shouldRetryRecalc(err error, usedTemplate bool) bool {
	return errors.Is(err, ErrRecalcRead) ||
		(usedTemplate && errors.Is(err, ErrRecalcUnavailable))
}

// recalcAttempt 跑一次重算：建一份本次专属的配置目录（有模板就复制，没有就交给
// 脚本从零初始化），调用脚本，用完删除。跑成的全新配置目录留作模板。
func recalcAttempt(path string) (rep RecalcReport, usedTemplate bool, err error) {
	prof, err := os.MkdirTemp(recalcTempDir(), "gaea-lo-recalc-")
	if err != nil {
		return RecalcReport{}, false, fmt.Errorf("%w：建临时配置目录失败 %v", ErrRecalcUnavailable, err)
	}
	defer func() { _ = os.RemoveAll(prof) }()
	usedTemplate = copyDir(recalcTemplateDir(), prof) == nil
	rep, err = runRecalcScript(path, prof)
	if err == nil && !usedTemplate {
		promoteRecalcTemplate(prof)
	}
	return rep, usedTemplate, err
}

// RecalcCacheRoot 重算配置模板的缓存根：%LOCALAPPDATA%\gaea\lo-recalc
// （LOCALAPPDATA 缺失回退系统临时目录——与 comhelper.CacheRoot 同降级）。
func RecalcCacheRoot() string {
	if recalcCacheRootOverride != nil {
		return recalcCacheRootOverride()
	}
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "gaea", "lo-recalc")
}

// recalcTemplateDir 配置模板目录：名带内嵌脚本指纹，脚本一改就换新模板。
func recalcTemplateDir() string {
	sum := sha256.Sum256([]byte(recalcScript))
	return filepath.Join(RecalcCacheRoot(), recalcProfilePrefix+hex.EncodeToString(sum[:])[:12])
}

// promoteRecalcTemplate 把这次跑成的全新配置目录留作模板。失败不报错：下次
// 调用再建，只是多花一次首次初始化的时间。
func promoteRecalcTemplate(prof string) {
	dst := recalcTemplateDir()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return
	}
	if err := fileutil.RenameWithRetry(prof, dst); err != nil {
		// 换不了名（跨卷，或并发的另一方先到）：复制到暂存名再换名落位——不让
		// 别的调用读到半份模板。
		staging := dst + ".staging"
		if copyDir(prof, staging) != nil || os.Rename(staging, dst) != nil {
			_ = os.RemoveAll(staging)
			return
		}
	}
	removeStaleRecalcTemplates(dst)
}

// removeStaleRecalcTemplates 清掉旧脚本指纹的模板（内嵌脚本升级后遗留的）。
func removeStaleRecalcTemplates(keep string) {
	entries, err := os.ReadDir(filepath.Dir(keep))
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), recalcProfilePrefix) || e.Name() == filepath.Base(keep) {
			continue
		}
		_ = os.RemoveAll(filepath.Join(filepath.Dir(keep), e.Name()))
	}
}

// copyDir 递归复制目录（文件与子目录；非常规文件跳过）。src 不是目录即报错，
// 调用方据此走「全新配置」。
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		return copyFile(p, target)
	})
}

// execRecalcScript 调内嵌脚本：脚本落成临时文件（每次调用一份，用完即删），
// 配置目录经环境变量交接。
func execRecalcScript(path, profile string) (RecalcReport, error) {
	script, err := os.CreateTemp("", "gaea-recalc-*.py")
	if err != nil {
		return RecalcReport{}, fmt.Errorf("%w：落临时脚本失败 %v", ErrRecalcUnavailable, err)
	}
	scriptPath := script.Name()
	defer func() { _ = os.Remove(scriptPath) }()
	if _, err := script.WriteString(recalcScript); err != nil {
		_ = script.Close()
		return RecalcReport{}, fmt.Errorf("%w：写临时脚本失败 %v", ErrRecalcUnavailable, err)
	}
	if err := script.Close(); err != nil {
		return RecalcReport{}, fmt.Errorf("%w：写临时脚本失败 %v", ErrRecalcUnavailable, err)
	}
	cmd := exec.Command("python", scriptPath, path, "60")
	cmd.Env = proc.CommandEnv(recalcProfileKey + "=" + profile)
	proc.HideWindow(cmd) // Windows: 防止弹出 cmd 黑框
	out, err := cmd.CombinedOutput()
	return parseRecalcOutput(out, err)
}

// parseRecalcOutput 把脚本输出折成报告或归类错误（纯函数，便于单测）。
func parseRecalcOutput(out []byte, runErr error) (RecalcReport, error) {
	if runErr != nil {
		return RecalcReport{}, fmt.Errorf("%w：重算脚本没能跑完（%v）%s",
			ErrRecalcUnavailable, runErr, truncate(string(out), 300))
	}
	rep, err := parseRecalcJSON(out)
	if err != nil {
		return RecalcReport{}, fmt.Errorf("%w：重算输出不是 JSON（%v）%s",
			ErrRecalcUnavailable, err, truncate(string(out), 300))
	}
	switch {
	case rep.Error == "":
		return rep, nil
	case rep.Stage == "read":
		return RecalcReport{}, fmt.Errorf("%w：%s", ErrRecalcRead, rep.Error)
	default:
		return RecalcReport{}, fmt.Errorf("%w：%s", ErrRecalcUnavailable, rep.Error)
	}
}

// parseRecalcJSON 解析脚本回执。stdout 与 stderr 是合并 captured 的，脚本若在
// JSON 之外还打了告警，就退而只读第一个 '{' 起的第一个 JSON 值（脚本自己只打
// 一个 JSON 对象，前后的告警行都不碍事）。
func parseRecalcJSON(out []byte) (RecalcReport, error) {
	var rep RecalcReport
	err := json.Unmarshal(out, &rep)
	if err == nil {
		return rep, nil
	}
	if i := bytes.IndexByte(out, '{'); i >= 0 {
		err = json.NewDecoder(bytes.NewReader(out[i:])).Decode(&rep)
	}
	return rep, err
}
