package app

// ── 图像域产物溯源登记（Asset Ledger v1）──
//
// 设计 docs/gaea-image-domain-t0-contract-design-2026-09.md §4：
//   - 按空间 append-only JSONL（play = <cwd>/.gaea/play/imagehub/assets.jsonl；
//     work = <cwd>/.gaea/imagehub/assets.jsonl）；
//   - 只存路径 + 元数据，不存 base64、不搬文件；
//   - 上限折叠不删文件（索引是辅助视图，不是文件真相）；
//   - 坏行跳过 + 计数，损坏不致命。

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gaea/gaea/internal/gaea/spaces"
)

// imageHubLedgerRecord JSONL 单行：溯源元数据 + 资产路径。
type imageHubLedgerRecord struct {
	Meta  imageHubAssetMeta `json:"meta"`
	Asset imageHubAsset     `json:"asset"`
}

// imageHubLedgerMaxLines 每空间索引上限（可调；折叠只丢索引不删文件）。
var imageHubLedgerMaxLines = 2000

// imageHubLedgerMu 进程内单写锁（JSONL append + 折叠都需互斥）。
var imageHubLedgerMu sync.Mutex

// imageHubLedger 以 cwd 为锚的登记簿（单测可注入临时目录）。
type imageHubLedger struct {
	cwd string
}

// normalizeImageHubSpace 空间归一：""/未知 → work（与 spaces.ExportsDir 语义一致）。
func normalizeImageHubSpace(space string) string {
	if spaces.Normalize(space) == spaces.SpacePlay {
		return spaces.SpacePlay
	}
	return spaces.SpaceWork
}

// newImageHubLedger 创建登记簿。
func newImageHubLedger(cwd string) *imageHubLedger {
	return &imageHubLedger{cwd: cwd}
}

// imageHubLedgerPath 返回空间索引文件路径（S4 目录分区镜像）。
func imageHubLedgerPath(cwd, space string) string {
	if normalizeImageHubSpace(space) == spaces.SpacePlay {
		return filepath.Join(cwd, ".gaea", "play", "imagehub", "assets.jsonl")
	}
	return filepath.Join(cwd, ".gaea", "imagehub", "assets.jsonl")
}

// record 追加一条登记（并发安全；写后触发折叠检查）。
func (l *imageHubLedger) record(space string, rec imageHubLedgerRecord) error {
	if strings.TrimSpace(rec.Asset.ID) == "" || strings.TrimSpace(rec.Asset.Path) == "" {
		return fmt.Errorf("登记缺资产 ID 或路径")
	}
	p := imageHubLedgerPath(l.cwd, space)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("创建登记目录失败: %w", err)
	}
	line, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("登记序列化失败: %w", err)
	}

	imageHubLedgerMu.Lock()
	defer imageHubLedgerMu.Unlock()

	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("打开登记文件失败: %w", err)
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		_ = f.Close()
		return fmt.Errorf("写入登记失败: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("关闭登记文件失败: %w", err)
	}
	return l.pruneIfNeeded(p)
}

// list 返回某空间最近的 N 条登记（坏行跳过；文件缺失 = 空表不报错）。
func (l *imageHubLedger) list(space string, limit int) []imageHubLedgerRecord {
	p := imageHubLedgerPath(l.cwd, space)
	raw, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		slog.Warn("读取图像域登记失败", "path", p, "error", err)
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	out := make([]imageHubLedgerRecord, 0, len(lines))
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		var rec imageHubLedgerRecord
		if err := json.Unmarshal([]byte(ln), &rec); err != nil {
			slog.Warn("图像域登记坏行已跳过", "path", p, "error", err)
			continue
		}
		out = append(out, rec)
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}

// pruneIfNeeded 行数超上限时保留最近 imageHubLedgerMaxLines 行（临时文件 + 替换）。
func (l *imageHubLedger) pruneIfNeeded(p string) error {
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) <= imageHubLedgerMaxLines {
		return nil
	}
	keep := lines[len(lines)-imageHubLedgerMaxLines:]
	tmp, err := os.CreateTemp(filepath.Dir(p), "assets-*.tmp")
	if err != nil {
		return fmt.Errorf("创建折叠临时文件失败: %w", err)
	}
	tmpName := tmp.Name()
	for _, ln := range keep {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		if _, err := tmp.WriteString(strings.TrimSpace(ln) + "\n"); err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
			return fmt.Errorf("折叠写入失败: %w", err)
		}
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("折叠关闭失败: %w", err)
	}
	// Windows 下 os.Rename 不能覆盖已存在文件：先移除旧文件再替换（记录折叠窗口极短）。
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(tmpName)
		return fmt.Errorf("折叠移除旧文件失败: %w", err)
	}
	if err := os.Rename(tmpName, p); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("折叠替换失败: %w", err)
	}
	return nil
}
