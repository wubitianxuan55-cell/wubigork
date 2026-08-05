// Package proposal — 全局服务单例（供办公 agent 工具访问）
package proposal

import "sync"

var (
	globalSvcMu sync.Mutex
	globalSvc   *Service
)

// SetGlobalServiceForTest 注入服务（测试隔离）
func SetGlobalServiceForTest(svc *Service) {
	globalSvcMu.Lock()
	defer globalSvcMu.Unlock()
	globalSvc = svc
}

// ResetGlobalServiceForTest 清空全局服务
func ResetGlobalServiceForTest() {
	globalSvcMu.Lock()
	defer globalSvcMu.Unlock()
	globalSvc = nil
}

// GlobalService 返回全局服务（办公工具用）
func GlobalService() *Service {
	globalSvcMu.Lock()
	defer globalSvcMu.Unlock()
	return globalSvc
}
