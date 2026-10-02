package app

import (
	"strings"

	"github.com/gaea/gaea/internal/app/board"
)

// GetBoardManifests 返回 canonical 板块 manifest 清单（§5.2）：
//   - 前端 MainLayout/ModuleLauncher 数据驱动（菜单/快捷键/页面映射/启动器
//     全部由清单生成，附 B 的 12 个硬编码点收敛）；
//   - 启动自检与审计 dump（与 GetBoardManifests 之外的「实际装配结果」对应，
//     见 09 报告 §6.3 的 DSH dump-config 借鉴）。
//
// Wails 绑定挂 CoreB（gen_bindings explicitOverrides：GetBoardManifests→core）。
func (a *App) GetBoardManifests() []board.Manifest {
	return board.BuiltinManifests()
}

// boardIDsFromManifests 把 manifest 清单拼成「板块 id 白名单串」（空格分隔，
// 顺序即 manifest 顺序 = 菜单顺序）。空清单/全空 id 返回 ""——调用方按
// 「不打白名单约束」降级（不 panic、不编造清单）。
func boardIDsFromManifests(ms []board.Manifest) string {
	ids := make([]string, 0, len(ms))
	for _, m := range ms {
		if id := strings.TrimSpace(m.ID); id != "" {
			ids = append(ids, id)
		}
	}
	return strings.Join(ids, " ")
}

// boardLabelsFromManifests 把 manifest 清单拼成「板块展示名串」（"/" 分隔，
// 供工具说明里给模型看的板块枚举）。空清单返回 ""。
func boardLabelsFromManifests(ms []board.Manifest) string {
	labels := make([]string, 0, len(ms))
	for _, m := range ms {
		if label := strings.TrimSpace(m.Label); label != "" {
			labels = append(labels, label)
		}
	}
	return strings.Join(labels, "/")
}

// boardIDList 返回当前 manifest 的板块 id 白名单串（提示词/工具 schema 的唯一
// 来源）。审计 AP8-08：此前 intent_llm.go 的 LLM 兜底提示词与 wx_agent.go 的
// 工具 schema/回执各手写一份 id 清单，含已删的 code、缺 schedule/sin/knowledge
// ——同一件事三套口径；现统一经本函数（GetBoardManifests 即 canonical 清单）。
func (a *App) boardIDList() string {
	return boardIDsFromManifests(a.GetBoardManifests())
}

// boardLabelList 返回当前 manifest 的板块展示名串（同 boardIDList 单一来源）。
func (a *App) boardLabelList() string {
	return boardLabelsFromManifests(a.GetBoardManifests())
}
