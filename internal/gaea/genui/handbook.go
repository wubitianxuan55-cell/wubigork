// Package genui — GenUI 蒸馏的 Go 侧共享资产：模型词汇手册、提示词规则、
// 结构校验。上限常量与 frontend/src/genui/spec.ts 的 GENUI_LIMITS 同源
// （改动必须两边同步，并在注释互相引用）。
package genui

import _ "embed"

// Handbook 是办公引擎内置 inline 技能 genui 的正文（run_skill 按需折入）。
//
//go:embed handbook.md
var Handbook string

// ChatRule 是聊天板块（plain 与人格模式）注入的精简结构化呈现规则。
//
//go:embed chatrule.md
var ChatRule string

// OfficePointer 是办公系统提示词里的短指针：完整词汇在 run_skill genui。
//
//go:embed officepointer.md
var OfficePointer string
