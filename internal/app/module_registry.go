package app

import (
	"fmt"
	"log/slog"
	"sort"

	"github.com/gaea/gaea/internal/app/board"
)

// Module 是一个可派发模块：Handle 接收 input map，返回结构化输出。
type Module struct {
	ID      string
	Name    string
	Intents []string
	Handle  func(input map[string]any) (map[string]any, error)
}

// ModuleRegistry 模块注册表（主脑可选编排用）。3.0 §5.2 起由板块 manifest
// 填充（FillFromManifests），Register 保留为测试/扩展的直接注册入口。
type ModuleRegistry struct {
	modules map[string]Module
	fillErr error
}

func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{modules: map[string]Module{}}
}

func (r *ModuleRegistry) Register(m Module) error {
	if m.ID == "" || m.Handle == nil {
		return fmt.Errorf("module needs id and handle")
	}
	if _, dup := r.modules[m.ID]; dup {
		return fmt.Errorf("duplicate module %q", m.ID)
	}
	r.modules[m.ID] = m
	return nil
}

func (r *ModuleRegistry) Dispatch(moduleID, intent string, input map[string]any) (map[string]any, error) {
	m, ok := r.modules[moduleID]
	if !ok {
		slog.Error("主脑: 派发未知模块", "module", moduleID, "intent", intent)
		return nil, fmt.Errorf("unknown module %q", moduleID)
	}
	for _, i := range m.Intents {
		if i == intent {
			return m.Handle(input)
		}
	}
	return nil, fmt.Errorf("unknown intent %q for module %q", intent, moduleID)
}

func (r *ModuleRegistry) Has(moduleID string) bool {
	_, ok := r.modules[moduleID]
	return ok
}

// Err 返回最近一次 FillFromManifests 的完整性错误（启动自检用；nil = 装配成功）。
// 缺陷 2 的机器保证：intent 无 handler 时装配失败，错误可被启动路径读取，
// 不静默跳过。
func (r *ModuleRegistry) Err() error { return r.fillErr }

// Handler 意图处理器（App 方法闭包）。
type Handler func(input map[string]any) (map[string]any, error)

// IntentResolver 解析 (板块ID, 意图) → 主脑模块 + 处理器。
// 由 app 侧提供（module_bindings.go 的 resolveIntent：moduleOfIntent 表 +
// intentHandlers 闭包表）；返回 ok=false 表示该 (板块, 意图) 无处理器。
type IntentResolver func(boardID, intent string) (moduleID string, handler Handler, ok bool)

// FillFromManifests 由板块 manifest 填充注册表（§5.2 Go 侧）：
//
//   - 遍历板块清单，收集每个板块 manifest 声明的意图（Intents）；
//   - 每个意图经 resolver 解析为 (模块, 处理器)；解析失败（无 handler /
//     未知意图）→ 立即返回 error 并记录到 r.fillErr —— 启动即报错，杜绝
//     「intent 无 handler 静默跳过」的缺陷 2 复发；
//   - 解析成功的意图按模块聚合注册（Module.ID = 主脑模块 id，与
//     classifyMainBrainIntent 的返回空间一致）。
//
// 限制：Module.Handle 不含意图参数，当前仅支持单意图模块；多意图模块会
// 返回完整性错误（现状 canonical 板块全部为单意图）。
func (r *ModuleRegistry) FillFromManifests(boards []board.Board, resolve IntentResolver) error {
	r.fillErr = nil
	type modDecl struct {
		moduleID string
		name     string
		intents  []string
		handle   Handler
	}
	mods := map[string]*modDecl{}
	for _, b := range boards {
		if b == nil {
			return r.fillFail("板块清单含 nil 板块")
		}
		intents := b.Intents()
		if len(intents) == 0 {
			continue // 无意图的板块不产生模块（记忆中枢/模型中心等）
		}
		for _, intent := range intents {
			moduleID, handler, ok := resolve(b.ID(), intent)
			if !ok || handler == nil {
				return r.fillFail(fmt.Sprintf(
					"板块 %q 意图 %q 无 handler（manifest 完整性断言失败；缺陷 2 防复发：intent 必须注册处理器）",
					b.ID(), intent))
			}
			d := mods[moduleID]
			if d == nil {
				d = &modDecl{moduleID: moduleID, name: b.Name()}
				mods[moduleID] = d
			}
			d.intents = append(d.intents, intent)
			d.handle = handler
		}
	}
	ids := make([]string, 0, len(mods))
	for id := range mods {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		d := mods[id]
		if len(d.intents) != 1 {
			return r.fillFail(fmt.Sprintf(
				"模块 %q 声明 %d 个意图（Module.Handle 不含意图参数，当前仅支持单意图模块）",
				id, len(d.intents)))
		}
		if err := r.Register(Module{ID: d.moduleID, Name: d.name, Intents: d.intents, Handle: d.handle}); err != nil {
			return r.fillFail(err.Error())
		}
	}
	return nil
}

// fillFail 记录完整性错误并返回（启动自检读取 r.Err() 可拿到同一错误）。
func (r *ModuleRegistry) fillFail(msg string) error {
	r.fillErr = fmt.Errorf("板块 manifest 装配失败: %s", msg)
	return r.fillErr
}
