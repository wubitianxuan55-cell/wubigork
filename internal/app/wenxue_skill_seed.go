package app

// wenxue_skill_seed.go — 图文板块小说域技能播种分发链（2026-11-17 审查批5 刀2）。
//
// 背景：novel-agents/novel-deslop/novel-review 三件在图文 allowlist 在册
// （wenxue_skills.go），但 store 发现链扫的是 书工作空间 .gaea/skills
// （skill.Store.roots 首根）→ 自定义 paths → home 约定目录；用户书目录没有
// 这三件、gaea.toml 也无 [skills] paths → 索引不列、run_skill 不报 unknown
// 的「自洽不可达」态。
//
// 为什么播种而不是 go:embed 直挂 SkillExtras：novel-agents 的派发契约要求
// 子代理 read_file 角色卡 `.gaea/skills/novel-agents/agents/<role>.md`（相对
// 工作区根）、novel-deslop 的词表覆盖点 words.json、novel-review 的平台
// rubrics——都是**工作区相对路径的伴随文件**。只有把完整目录落进书工作空间
// （store 首根），索引可见性、run_skill 派发、read_file 伴随件三条链才同时
// 成立；SkillExtras 单 Body 挂载会留下断链的派发契约。
//
// 纪律：只补缺失文件（用户改动/词表覆盖先到先得），永不覆盖已有文件；幂等
// （二次播种 count=0）；失败告警不阻断建书/切书。办公引擎零触碰。
//
// 资产双源防漂移：本目录是仓内 .gaea/skills 的快照拷贝，漂移守卫见
// wenxue_skill_seed_test.go（嵌入 FS 与仓源头文件逐字节比对，全档位件与
// 无限制档专属件同守卫）。

import (
	"embed"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/wspath"
)

//go:embed all:wenxue_skill_seed
var wenxueSkillSeedFS embed.FS

// wenxueSeedSkillNames 播种技能清单（全档位；与 wenxueAllowedSkills 对齐；
// story-long-write/story-deslop/wenxue-workbench 是内嵌 Body 三件不经此链）。
var wenxueSeedSkillNames = []string{"novel-agents", "novel-deslop", "novel-review"}

// wenxueSeedSkillNamesUnrestricted 无限制档专属播种清单（2026-11-23 用户拍板）：
// 只在 深度成人（无限制） 书落盘——文件在 store 首根才可被发现链索引；
// 非无限制书不落盘即五面不可见。降档不删文件（先到先得纪律），残留可见性由
// wenxueSkillFilter 的动态档位闸兜住（无限制时才加载，五面同源成立）。
var wenxueSeedSkillNamesUnrestricted = []string{"novel-chapter-branches"}

// wenxueSeedBookSkills 把小说域技能播种进书工作空间 .gaea/skills/。
// unrestricted=true 时额外播种无限制档专属件。返回新落盘文件数；
// bookRoot 越界/空根时诚实拒绝（不写盘）。
func wenxueSeedBookSkills(bookRoot string, unrestricted bool) int {
	if strings.TrimSpace(bookRoot) == "" {
		return 0
	}
	names := wenxueSeedSkillNames
	if unrestricted {
		names = append(append([]string{}, wenxueSeedSkillNames...), wenxueSeedSkillNamesUnrestricted...)
	}
	count := 0
	for _, name := range names {
		srcRoot := "wenxue_skill_seed/" + name
		err := fs.WalkDir(wenxueSkillSeedFS, srcRoot, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(srcRoot, p)
			if err != nil {
				return err
			}
			dst := filepath.Join(bookRoot, ".gaea", "skills", name, filepath.FromSlash(rel))
			// 越界拒绝（书根必须包含目标；防御性——bookRoot 来自内部流程）。
			if _, err := wspath.ResolveRelWithin(bookRoot, filepath.ToSlash(filepath.Join(".gaea", "skills", name, rel))); err != nil {
				return nil
			}
			if _, err := os.Stat(dst); err == nil {
				return nil // 已存在=用户改动或上次播种，先到先得
			}
			body, err := fs.ReadFile(wenxueSkillSeedFS, p)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(dst, body, 0o644); err != nil {
				return err
			}
			count++
			return nil
		})
		if err != nil {
			slog.Warn("图文技能播种失败（不阻断）", "skill", name, "error", err)
		}
	}
	return count
}
