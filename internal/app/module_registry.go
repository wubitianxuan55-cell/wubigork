package app

import "fmt"

// Module 是一个可派发模块：Handle 接收 input map，返回结构化输出。
type Module struct {
	ID      string
	Name    string
	Intents []string
	Handle  func(input map[string]any) (map[string]any, error)
}

// ModuleRegistry 模块注册表（主脑可选编排用）。
type ModuleRegistry struct {
	modules map[string]Module
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
