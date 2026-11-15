package app

import "github.com/gaea/gaea/internal/types"

// character_inject_fields.go — 项目角色注入配置的共享原语（2026-11-15 小说板块
// 退役后从 character_handler.go / create_chapter_context.go 收敛而来；唯一消费方
// 是角色库「加入项目」导入链 characterlib_handler.go）。

// 注入模式四态（types.Character.InjectMode，项目级字段）：
//
//	"" / "auto"  检测注入——命中名字或别名才升全量档案行
//	"always"     恒注入——无条件升全量档案行（关键实体兜底）
//	"muted"      检测不注入——命中也不升档，仅名册备查
//	"never"      不渲染——全链不可见
const (
	injectModeAlways = "always"
	injectModeMuted  = "muted"
	injectModeNever  = "never"
)

// carryProjectInjectFields 把源角色的项目级注入配置（阶段八 8.1/8.2）回带到目标——
// 角色库投影/AI 补全重建角色卡时，注入模式/别名/排除词/状态演进只落本书，不随
// 投影丢失（v4.454 ProtagonistRelation 同款保护）。
func carryProjectInjectFields(dst *types.Character, src types.Character) {
	dst.InjectMode = src.InjectMode
	dst.Aliases = src.Aliases
	dst.ExcludeHits = src.ExcludeHits
	dst.Progressions = src.Progressions
}
