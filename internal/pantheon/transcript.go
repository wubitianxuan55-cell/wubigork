package pantheon

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

// transcript 把整场论证按贡献归档为独立文件，提示词只携带索引行与全文路径，
// 席位用文件读取工具按需自读（2026-11-03 实弹：6 席×3 轮在旧 24000 字符预算下
// 单场 22 次「提示词超限截断」，历史论证被 clamp 砍掉中段，严重影响论证）。
// session.json 仍是权威全文；这里的文件是分发辅助副本，缺失时从内存贡献惰性
// 补写，旧会话 Resume 与中断恢复不需要迁移步骤。
//
// 分发梯度（守 2026-10-31 圆桌裁决第一刀，只降更早轮次）：
//   同轮发言：全文（超限保完整尾部）——见 prompts.go formatContributionBody；
//   上一轮：头 800 字立场卡（=既有规则平移）；
//   更早轮次：索引行 + 头 400 字预览 + 全文路径。
type transcript struct {
	dir string // sessions/<id>
}

var safeContributionID = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func newTranscript(root, sessionID string) (transcript, error) {
	if !safeSessionID.MatchString(sessionID) {
		return transcript{}, fmt.Errorf("非法议事 ID %q", sessionID)
	}
	return transcript{dir: filepath.Join(root, "sessions", sessionID)}, nil
}

func (t transcript) contributionPath(id string) string {
	return filepath.Join(t.dir, "contrib", id+".md")
}

func (t transcript) evidenceDir() string {
	return filepath.Join(t.dir, "evidence")
}

// ensureContributionFile 幂等归档一条贡献：文件已在且非空即直接返回；缺失则
// 从内存全文补写（Resume/中断恢复的旧会话首次分发时自动补齐）。
func (t transcript) ensureContributionFile(item Contribution) (string, error) {
	if !safeContributionID.MatchString(item.ID) || strings.Contains(item.ID, "..") {
		return "", fmt.Errorf("非法贡献 ID %q", item.ID)
	}
	if strings.TrimSpace(item.Content) == "" {
		return "", fmt.Errorf("贡献 %s 无正文可归档", item.ID)
	}
	path := t.contributionPath(item.ID)
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("创建归档目录: %w", err)
	}
	if err := fileutil.AtomicWriteSync(path, []byte(item.Content), 0o644); err != nil {
		return "", fmt.Errorf("归档贡献 %s: %w", item.ID, err)
	}
	return path, nil
}

// evidenceFileName 用序号前缀防重名；用户提供的材料名只做字符净化，保留中文。
func evidenceFileName(index int, name string) string {
	return fmt.Sprintf("%02d-%s.md", index+1, sanitizeFileName(name))
}

func sanitizeFileName(name string) string {
	base := strings.TrimSpace(name)
	if base == "" {
		base = "材料"
	}
	var b strings.Builder
	for _, r := range base {
		switch {
		case r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|':
			b.WriteRune('-')
		case r < 0x20:
			// 控制字符直接丢弃
		default:
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(b.String())
	out = strings.Trim(out, ". ")
	if out == "" {
		out = "材料"
	}
	if len([]rune(out)) > 40 {
		out = string([]rune(out)[:40])
	}
	return out
}

// ensureEvidenceFile 幂等归档一条大块材料，返回全文路径。
func (t transcript) ensureEvidenceFile(index int, item EvidenceItem) (string, error) {
	if strings.TrimSpace(item.Content) == "" {
		return "", fmt.Errorf("材料 %q 无正文可归档", item.Name)
	}
	path := filepath.Join(t.evidenceDir(), evidenceFileName(index, item.Name))
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("创建材料归档目录: %w", err)
	}
	if err := fileutil.AtomicWriteSync(path, []byte(item.Content), 0o644); err != nil {
		return "", fmt.Errorf("归档材料 %q: %w", item.Name, err)
	}
	return path, nil
}

// archiveContributions 把全部贡献归档落盘，返回全文路径列表。单条失败跳过：
// 归档是分发辅助，失败的那条在提示词构建时降级为内联节选，不影响议事推进。
// 返回值供 grok 席 readPaths 白名单使用。
func (t transcript) archiveContributions(contributions []Contribution) []string {
	paths := make([]string, 0, len(contributions))
	for _, item := range contributions {
		if path, err := t.ensureContributionFile(item); err == nil {
			paths = append(paths, path)
		}
	}
	return paths
}
