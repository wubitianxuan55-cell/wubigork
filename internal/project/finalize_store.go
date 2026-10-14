package project

// ── 章节定稿档 + 阶段合账缓存（阶段八 8.4 定稿回写）──────────
//
// 沿用本包三态语义（与 plans.json 同律）：
//   - 缺失 = 正常态（未定稿 / 无缓存，返回 nil+false+nil 或空表）；
//   - 损坏 = error 且绝不覆盖（定稿档可能是作者唯一留痕；合账缓存损坏由消费方
//     降级即时合成，不阻断生成）；
//   - 写入 = 原子替换（writeJSON → fileutil.AtomicWrite）。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gaea/gaea/internal/types"
)

// ChapterFinalizationPath 定稿档路径 chapters/NNN{,a,b,c}-final.json
// （与章节正文/摘要文件同目录同命名口径）。
func (m *Manager) ChapterFinalizationPath(num int, branch string) string {
	name := fmt.Sprintf("%03d-final.json", num)
	if branch != "" {
		name = fmt.Sprintf("%03d%s-final.json", num, branch)
	}
	return filepath.Join(m.Dir, "chapters", name)
}

// ReadChapterFinalization 读定稿档。未定稿 = (nil, false, nil) 正常态；
// 损坏 = error 且原文件原封不动。
func (m *Manager) ReadChapterFinalization(num int, branch string) (*types.ChapterFinalization, bool, error) {
	path := m.ChapterFinalizationPath(num, branch)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("读取定稿档失败 (%s): %w", path, err)
	}
	var cf types.ChapterFinalization
	if err := json.Unmarshal(raw, &cf); err != nil {
		return nil, false, fmt.Errorf("解析定稿档失败 (%s): %w", path, err)
	}
	return &cf, true, nil
}

// WriteChapterFinalization 原子写定稿档（目录不存在自动创建）。
// 章号 <1 直接报错，不写出无主档。
func (m *Manager) WriteChapterFinalization(cf *types.ChapterFinalization) error {
	if cf == nil {
		return fmt.Errorf("定稿档为空")
	}
	if cf.ChapterNum < 1 {
		return fmt.Errorf("定稿档章号非法: %d", cf.ChapterNum)
	}
	if cf.Version <= 0 {
		cf.Version = types.ChapterFinalizationVersion
	}
	path := m.ChapterFinalizationPath(cf.ChapterNum, cf.Branch)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建 chapters 目录失败: %w", err)
	}
	return writeJSON(path, cf)
}

// StageRecapCachePath 阶段合账缓存路径 chapters/stage-recaps.json。
func (m *Manager) StageRecapCachePath() string {
	return filepath.Join(m.Dir, "chapters", "stage-recaps.json")
}

// ReadStageRecapCache 读合账缓存。缺失 = 空表正常态；损坏 = error（消费方降级
// 即时合成，不覆盖坏档——坏档可能是作者手改）。
func (m *Manager) ReadStageRecapCache() (*types.StageRecapCacheFile, error) {
	f, err := loadJSON[types.StageRecapCacheFile](m.StageRecapCachePath())
	if err != nil {
		if os.IsNotExist(err) {
			return &types.StageRecapCacheFile{Version: 1, Recaps: map[string]types.StageRecapEntry{}}, nil
		}
		return nil, err
	}
	if f == nil {
		return &types.StageRecapCacheFile{Version: 1, Recaps: map[string]types.StageRecapEntry{}}, nil
	}
	if f.Recaps == nil {
		f.Recaps = map[string]types.StageRecapEntry{}
	}
	if f.Version <= 0 {
		f.Version = 1
	}
	return f, nil
}

// UpsertStageRecap 合账缓存按阶段号 upsert（读-改-写，key=阶段号十进制）。
func (m *Manager) UpsertStageRecap(stage int, entry types.StageRecapEntry) error {
	if stage < 1 {
		return fmt.Errorf("阶段号非法: %d", stage)
	}
	f, err := m.ReadStageRecapCache()
	if err != nil {
		return err
	}
	f.Recaps[strconv.Itoa(stage)] = entry
	path := m.StageRecapCachePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建 chapters 目录失败: %w", err)
	}
	return writeJSON(path, f)
}

// ReadStageRecap 读单阶段合账（存在与否用 bool 表达）。
func (m *Manager) ReadStageRecap(stage int) (*types.StageRecapEntry, bool, error) {
	f, err := m.ReadStageRecapCache()
	if err != nil {
		return nil, false, err
	}
	e, ok := f.Recaps[strconv.Itoa(stage)]
	if !ok {
		return nil, false, nil
	}
	return &e, true, nil
}
