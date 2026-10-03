package project

// project_files.go — 项目固定资产的文件读写（批 28 IN1-01 文件拆分，自
// project.go 原位搬移，零逻辑改动）：世界观（md/json 双形态）/角色/大纲/
// 伏笔 / 风格指纹·摘要·作者风格档案（StyleProfile，含 legacyBrandDir 同源
// 口径）/ Lorebook。生命周期与上下文构建见 project.go，章节读写见
// project_chapters.go，迁移与通用 JSON 辅助见 project_migrate.go，逐章
// 派生存储见 project_stores.go。

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/types"
)

// ── 文件读写辅助 ──────────────────────────────────────────────

// ReadWorldview 读世界观（优先 worldview.json，fallback worldview.md）
func (m *Manager) ReadWorldview() (string, error) {
	// 优先读结构化文件
	wf, err := m.ReadWorldviewFile()
	if err == nil && len(wf.Sections) > 0 {
		return wf.ToMarkdown(), nil
	}
	// fallback: 读取旧 worldview.md（不写入，迁移由 ReadWorldviewFile 负责）
	data, err := os.ReadFile(filepath.Join(m.Dir, "worldview.md"))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteWorldview 写世界观为 markdown（向后兼容，原子写）
func (m *Manager) WriteWorldview(content string) error {
	return fileutil.AtomicWrite(filepath.Join(m.Dir, "worldview.md"), []byte(content), 0o600)
}

// ReadWorldviewFile 读 worldview.json（不存在时从 worldview.md 自动迁移）
func (m *Manager) ReadWorldviewFile() (*types.WorldviewFile, error) {
	wf, err := loadJSON[types.WorldviewFile](filepath.Join(m.Dir, "worldview.json"))
	if err == nil {
		// 已有足够 section → 直接返回
		if len(wf.Sections) >= 2 {
			return wf, nil
		}
		// 只有 1 个 section 且是旧的 "main" section → 需要迁移
		if len(wf.Sections) == 1 && wf.Sections[0].ID == "main" {
			// fall through to migration
		} else if len(wf.Sections) >= 1 {
			// 已有至少 1 个真实 section（不是旧的 main）→ 保留不迁移
			return wf, nil
		}
	}

	// 需要迁移：读取旧 worldview.md
	oldContent := ""
	if err == nil && len(wf.Sections) == 1 && wf.Sections[0].Content != "" {
		oldContent = wf.Sections[0].Content
	} else {
		data, ferr := os.ReadFile(filepath.Join(m.Dir, "worldview.md"))
		if ferr == nil {
			oldContent = strings.TrimSpace(string(data))
		}
	}
	// 如果完全没有旧内容，创建空结构
	if oldContent == "" || oldContent == "# 世界观" || strings.HasPrefix(oldContent, "# 世界观") {
		sections := DefaultSections()
		if err := writeJSON(filepath.Join(m.Dir, "worldview.json"), &types.WorldviewFile{Sections: sections}); err != nil {
			return nil, err
		}
		return &types.WorldviewFile{Sections: sections}, nil
	}

	sections := DefaultSections()
	sections = append([]types.WorldviewSection{{
		ID:      "legacy",
		Title:   "📋 旧版世界观（请整理到下方各维度）",
		Content: oldContent,
		Order:   0,
	}}, sections...)

	if err := writeJSON(filepath.Join(m.Dir, "worldview.json"), &types.WorldviewFile{Sections: sections}); err != nil {
		return nil, err
	}
	return &types.WorldviewFile{Sections: sections}, nil
}

// WriteWorldviewFile 写 worldview.json
func (m *Manager) WriteWorldviewFile(wf *types.WorldviewFile) error {
	return writeJSON(filepath.Join(m.Dir, "worldview.json"), wf)
}

// ReadCharacters 读 characters.json
func (m *Manager) ReadCharacters() (*types.CharacterFile, error) {
	return loadJSON[types.CharacterFile](filepath.Join(m.Dir, "characters.json"))
}

// WriteCharacters 写 characters.json
func (m *Manager) WriteCharacters(cf *types.CharacterFile) error {
	return writeJSON(filepath.Join(m.Dir, "characters.json"), cf)
}

// ReadOutlines 读 outline.json
func (m *Manager) ReadOutlines() (*types.OutlineFile, error) {
	return loadJSON[types.OutlineFile](filepath.Join(m.Dir, "outline.json"))
}

// WriteOutlines 写 outline.json
func (m *Manager) WriteOutlines(of *types.OutlineFile) error {
	return writeJSON(filepath.Join(m.Dir, "outline.json"), of)
}

// ReadForeshadows 读 foreshadows.json
func (m *Manager) ReadForeshadows() (*types.ForeshadowFile, error) {
	return loadJSON[types.ForeshadowFile](filepath.Join(m.Dir, "foreshadows.json"))
}

// WriteForeshadows 写 foreshadows.json
func (m *Manager) WriteForeshadows(ff *types.ForeshadowFile) error {
	return writeJSON(filepath.Join(m.Dir, "foreshadows.json"), ff)
}

// StyleFingerprintPath 文风指纹参考档路径（fingerprint.json）。
// 边界声明（审计 IN1-05）：Fingerprint=评分口径，与 StyleProfile
// （.gaea/style-profile.json，生成注入口径）互为独立真相源，勿顺手合并。
func (m *Manager) StyleFingerprintPath() string {
	return filepath.Join(m.Dir, "fingerprint.json")
}

// ReadStyleFingerprint 读文风指纹参考档（不存在时返回 os 文件错误）
func (m *Manager) ReadStyleFingerprint() (*types.StyleFingerprintFile, error) {
	return loadJSON[types.StyleFingerprintFile](m.StyleFingerprintPath())
}

// WriteStyleFingerprint 写文风指纹参考档（原子写）
// ReadStyleDigest 读风格摘要档（长篇刀5）；文件缺失返回 nil（未构建态）。
func (m *Manager) ReadStyleDigest() (*types.StyleDigestFile, error) {
	df, err := loadJSON[types.StyleDigestFile](filepath.Join(m.Dir, "style_digest.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return df, nil
}

// WriteStyleDigest 写风格摘要档（原子写）。
func (m *Manager) WriteStyleDigest(df *types.StyleDigestFile) error {
	return writeJSON(filepath.Join(m.Dir, "style_digest.json"), df)
}

// ClearStyleDigest 删除风格摘要档（未构建时 no-op）。
func (m *Manager) ClearStyleDigest() error {
	err := os.Remove(filepath.Join(m.Dir, "style_digest.json"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (m *Manager) WriteStyleFingerprint(sf *types.StyleFingerprintFile) error {
	return writeJSON(m.StyleFingerprintPath(), sf)
}

// ── StyleProfile（作者风格档案，.gaea/style-profile.json）─────────────────
//
// 边界声明（审计 IN1-05）：本块与上方 StyleFingerprint/StyleDigest 是两套
// 互不相识的文风真相源——StyleProfile=生成注入口径（style.Profile，注入写作
// prompt），StyleFingerprint/StyleDigest=评分口径（fingerprint.json /
// style_digest.json，评审参考档）。刻意分层，互为独立真相源，勿顺手合并；
// 另见 internal/style 包 Profile 类型上的对偶声明。
//
// 为什么这里只有 []byte 读写而非 (*style.Profile, error)：style 包 import
// 本包（style→project），本包反向持有 style 类型即成环。故路径与品牌兼容
// 收口在本包，JSON 编解码留在 style 包薄壳。

// legacyBrandDir 旧品牌配置目录（.wubigork）。品牌兼容回退只允许出现在
// 本包（IsV4 / ReadStyleProfileFile 同源共用这一处目录名），其它包一律
// 经由本包读写，勿再各自手拼第二份兼容分支（审计 IN1-05）。
func legacyBrandDir(projectDir string) string {
	return filepath.Join(projectDir, ".wubigork")
}

// StyleProfilePath 作者风格档案路径（.gaea/style-profile.json）。
// 命名与 StyleFingerprintPath 同风格；落点单源，勿在他处手拼。
func (m *Manager) StyleProfilePath() string {
	return filepath.Join(m.Dir, ".gaea", "style-profile.json")
}

// ReadStyleProfileFile 读作者风格档案原始字节。
// 兼容旧品牌：优先 .gaea/，仅当缺失（os.IsNotExist）时回退旧品牌
// .wubigork/——档案级品牌回退全仓唯一收口点。错误原样透传，不包装。
func (m *Manager) ReadStyleProfileFile() ([]byte, error) {
	data, err := os.ReadFile(m.StyleProfilePath())
	if err != nil && os.IsNotExist(err) {
		data, err = os.ReadFile(filepath.Join(legacyBrandDir(m.Dir), "style-profile.json"))
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

// WriteStyleProfileFile 写作者风格档案原始字节（固定落 .gaea/，目录缺则建；
// 权限 0755/0644、非原子写，与历史 style.SaveProfile 行为逐字节一致——
// 勿顺手升级为原子写，评分口径的 writeJSON 才是原子语义）。
func (m *Manager) WriteStyleProfileFile(data []byte) error {
	if err := os.MkdirAll(filepath.Join(m.Dir, ".gaea"), 0755); err != nil {
		return err
	}
	return os.WriteFile(m.StyleProfilePath(), data, 0644)
}

// ── Lorebook ──────────────────────────────────────────────

// ReadLorebook 读取 lorebook.json（不存在时返回空）
func (m *Manager) ReadLorebook() (*types.LorebookFile, error) {
	path := filepath.Join(m.Dir, "lorebook.json")
	lf, err := loadJSON[types.LorebookFile](path)
	if err != nil {
		return &types.LorebookFile{}, nil
	}
	return lf, nil
}

// WriteLorebook 写入 lorebook.json
func (m *Manager) WriteLorebook(lf *types.LorebookFile) error {
	return writeJSON(filepath.Join(m.Dir, "lorebook.json"), lf)
}
