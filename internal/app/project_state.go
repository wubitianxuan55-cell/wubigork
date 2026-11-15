package app

import (
	"github.com/gaea/gaea/internal/project"
)

// project_state.go — 书架/项目管理共享态（2026-11-15 小说板块退役：原
// writing_state.go 收缩而来。getPM/setPM/closePM 三方法的消费方是书架
// shelf.go、项目开关 project_handler.go 与角色库「加入项目」导入链
// characterlib_handler.go，全部保留板块，故随 struct 一起留存）。

// getPM 以读锁获取当前项目（调用方用完即释放引用）
func (p *projectState) getPM() *project.Manager {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.pm
}

// setPM 以写锁设置当前项目
func (p *projectState) setPM(pm *project.Manager) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pm = pm
}

// closePM 以写锁关闭并清空当前项目
func (p *projectState) closePM() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pm == nil {
		return nil
	}
	err := p.pm.Close()
	p.pm = nil
	return err
}
