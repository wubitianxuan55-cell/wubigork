package app

// local_engine_proc.go — 外部本地引擎进程管理的共用底座（天璇 / Strata）。
// gaea 只登记自己拉起的启动器进程；停止时先结束该进程树，再按端口监听 PID
// 兜底，保证外部启动器起的实例也能停。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type localEngineProc struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	cancel context.CancelFunc
	// job kill-on-close 作业对象句柄（0=未纳管）：gaea 进程终止（正常退出/
	// 崩溃/被杀）时内核关闭句柄即终止整棵引擎进程树——外部引擎不再借 gaea
	// 的壳存活占机器。stop() 亦显式关闭（关闭即杀，兜住 taskkill 漏网层级）。
	job windows.Handle
}

func (p *localEngineProc) set(cmd *exec.Cmd, cancel context.CancelFunc) {
	p.mu.Lock()
	p.cmd = cmd
	p.cancel = cancel
	p.mu.Unlock()
}

// setWithJob 登记进程并纳管 kill-on-close 作业对象（gaea 退出/崩溃时内核
// 收割整棵引擎进程树）。纳管失败只记日志不阻断启动——引擎照常可用，仅失去
// 退出兜底（stop() 仍会 taskkill 收敛）。
func (p *localEngineProc) setWithJob(cmd *exec.Cmd, cancel context.CancelFunc) {
	job, err := attachKillOnCloseJob(cmd)
	if err != nil {
		slog.Warn("引擎进程纳管作业对象失败（gaea 退出将不自动终止该引擎）",
			"pid", cmd.Process.Pid, "error", err)
	}
	p.mu.Lock()
	p.cmd = cmd
	p.cancel = cancel
	p.job = job
	p.mu.Unlock()
}

// registered gaea 本会话是否登记了启动器进程（空闲自动退出只收 gaea 自己
// 拉起的实例——外部手动启动的引擎尊重其独立性，不代管）。
func (p *localEngineProc) registered() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cmd != nil && p.cmd.Process != nil
}

// attachKillOnCloseJob 创建 kill-on-close 作业对象并把已启动的进程纳入。
// 赋联在 Start 之后，存在微小竞窗（启动器若在纳管前已 spawn 子进程则该子
// 进程逃逸）——实际启动器（powershell serve.ps1）纳管前不会拉起服务进程，
// 可接受；子进程默认随父进程留在作业内（无 breakaway）。
func attachKillOnCloseJob(cmd *exec.Cmd) (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil || job == 0 {
		return 0, err
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	if ret, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil || ret == 0 {
		_ = windows.CloseHandle(job)
		return 0, err
	}
	// AssignProcessToJobObject 需要 PROCESS_SET_QUOTA | PROCESS_TERMINATE 权限，
	// 按 PID 重开句柄（os.Process 不暴露内核句柄）。
	ph, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		return 0, err
	}
	defer func() { _ = windows.CloseHandle(ph) }()
	if err := windows.AssignProcessToJobObject(job, ph); err != nil {
		_ = windows.CloseHandle(job)
		return 0, err
	}
	return job, nil
}

// closeJob 关闭作业对象（kill-on-close：句柄一关整树终止）。幂等。
func (p *localEngineProc) closeJob() {
	p.mu.Lock()
	job := p.job
	p.job = 0
	p.mu.Unlock()
	if job != 0 {
		_ = windows.CloseHandle(job)
	}
}

// clearIfCurrent 用 cmd 指针判身份，防止快停-慢收时晚到的 Wait goroutine
// 误清新一轮启动登记（与 comfyProcRefClearIfCurrent 同款 ABA 防护）。
// 作业句柄一并清零——句柄已由 stop() 关闭（或进程自亡后由 OS 回收），
// 留着会把新一轮启动的作业误关。
func (p *localEngineProc) clearIfCurrent(cmd *exec.Cmd) {
	p.mu.Lock()
	if p.cmd == cmd {
		p.cmd = nil
		p.cancel = nil
		p.job = 0
	}
	p.mu.Unlock()
}

// stop 结束 gaea 登记的启动器进程树。返回是否确有登记进程。
func (p *localEngineProc) stop() bool {
	p.mu.Lock()
	cmd := p.cmd
	cancel := p.cancel
	p.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return false
	}
	_, err := exec.Command("taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F").CombinedOutput()
	if err != nil {
		_ = cmd.Process.Kill()
	}
	// kill-on-close 兜底：句柄一关，作业内（含 taskkill /T 理论漏网的层级）
	// 全部终止；失败静默——进程已死的常见路径。
	p.closeJob()
	if cancel != nil {
		cancel()
	}
	p.clearIfCurrent(cmd)
	return true
}

// stopManagedEngine 是天璇 / Strata 共用的停止语义：登记进程树 → 端口监听 PID
// 兜底 → 轮询探活直到端口释放。probe 返回 true 表示引擎仍可达。
func stopManagedEngine(proc *localEngineProc, port int, wait, poll time.Duration, probe func() bool) map[string]interface{} {
	hadProcess := proc.stop()
	killErr := ""
	if pid := listeningPortPID(port); pid != 0 {
		hadProcess = true
		if out, err := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").CombinedOutput(); err != nil {
			// 进程可能在 netstat 与 taskkill 之间自己退出；不要据此提前判失败，
			// 继续以端口释放为最终判据（否则 UI 会误报停止失败并留下假状态）。
			killErr = strings.TrimSpace(string(out))
		}
	}
	if !hadProcess {
		if !probe() {
			return map[string]interface{}{"stopped": false, "reason": "not-running"}
		}
		// 端口探活仍可达但 netstat 没给出 PID（外部启动器/异常绑定）：
		// 不谎报已停止，继续等端口释放并如实返回失败原因。
	}
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		if !probe() {
			return map[string]interface{}{"stopped": true}
		}
		time.Sleep(poll)
	}
	if killErr != "" {
		return map[string]interface{}{"stopped": false, "reason": "taskkill-failed", "error": killErr}
	}
	return map[string]interface{}{"stopped": false, "reason": "port-still-busy"}
}

// probeOpenAIModelsAt 探测 OpenAI 兼容引擎的 /v1/models，返回模型 id 清单。
// 2xx 即可达；坏 JSON 仍视为可达（模型清单为空），避免状态点误报离线。
func probeOpenAIModelsAt(base string, timeout time.Duration) ([]string, bool) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(strings.TrimRight(base, "/") + "/models")
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false
	}
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return []string{}, true
	}
	ids := make([]string, 0, len(body.Data))
	for _, m := range body.Data {
		ids = append(ids, m.ID)
	}
	return ids, true
}

// listeningPortPID 找指定端口的监听进程 PID（netstat 口径）。
func listeningPortPID(port int) int {
	out, err := exec.Command("netstat", "-ano").Output()
	if err != nil {
		return 0
	}
	return parseListeningPortPID(string(out), port)
}

// parseListeningPortPID 从 netstat -ano 输出中找监听指定端口的 PID。匹配本地
// 地址后缀而非固定 127.0.0.1，兼容引擎绑定 0.0.0.0 / IPv6 的情况。
func parseListeningPortPID(out string, port int) int {
	wantSuffix := fmt.Sprintf(":%d", port)
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 5 && fields[0] == "TCP" && strings.HasSuffix(fields[1], wantSuffix) && fields[3] == "LISTENING" {
			if pid, err := strconv.Atoi(fields[4]); err == nil && pid > 0 {
				return pid
			}
		}
	}
	return 0
}

// readLogTail 取日志尾部用于启动早退的诚实报错（完整日志仍在文件里）。
// 只读最后 max 字节，避免大日志拖慢 UI。
func readLogTail(path string, max int) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return ""
	}
	offset := info.Size() - int64(max)
	if offset < 0 {
		offset = 0
	}
	if _, err := f.Seek(offset, 0); err != nil {
		return ""
	}
	data, err := io.ReadAll(io.LimitReader(f, int64(max)))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
