package builtin

// ── bash 会话状态锚定（v4.378，Reasonix persistentshell 留池·取道不取器）────
//
// 上游 persistentshell/ 让 cd/env/函数跨调用存活，用单进程 PTY+marker 协议
// +分帧实现（Windows 需 conpty）。gaea 的 bash 全链路——每调用新进程、
// Windows Job Object 灭树、boundedOutput 运行中封顶、超时杀进程——深绑
// 「无状态进程」形态，PTY 化是全套工程且处处打架。本刀取道不取器：
// 进程不持久，状态持久。
//
//   cwd  命令跑完时 EXIT trap 用 cygpath 捕获 Windows 形路径（MSYS 的 $PWD
//        是 /c/... 形，exec.Cmd.Dir 只认 Windows 形），下次调用直接落
//        cmd.Dir；目录消失自动回落工作空间。
//   env  同一时点 export -p 全量转储，下次调用前置 eval 重放——模型上次
//        export 的变量/导出函数原样复现（转储即真相，不做差量解析）。
//
// 漂移治理（上游「盲缓存 cwd 前缀注入」的病根在推断，本设计在观测）：状态
// 只在 shell 真实退出（含命令里的显式 exit——EXIT trap 照跑）时从真 shell
// 采集；被杀（超时/取消是 taskkill /F 硬杀，trap 不跑）、trap 被覆盖、exec
// 换身等一切采不到的形态一律不采纳——保守=不更新，缓存永不超前于现实。
// 并行批次两调用各用独立探针文件，完成序采纳（与两个进程共用一个终端的
// 语义一致）；内存竞态由锁兜住。
//
// 豁免：PowerShell 壳（语法不通）、sandbox enforce/WSL（壳内 cwd/env 是另
// 一个世界，host 侧 cmd.Dir 无效）、run_in_background（任务生命周期与调用
// 脱钩，不回写状态；起始目录仍锚定）。会话切换（NewSession/Resume）经
// tool.SessionStateResetter 整体重置，防跨会话泄漏。

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

// envDumpMaxBytes 是采纳 env 转储的字节上限：超限弃 env 只留 cwd（退化防御；
// 正常进程环境远小于此）。
const envDumpMaxBytes = 32 << 10

// shellState 是一个工具注册表（≈一个会话）的 bash 状态锚。零值可用但无状态；
// 组装点经 newShellState 挂进 bash 实例。
type shellState struct {
	mu  sync.Mutex
	cwd string // 锚定工作目录（host 形绝对路径；空=未锚定）
	env string // export -p 转储（空=未锚定）
	dir string // 探针文件目录（惰性创建；Reset 整树删除）
	seq atomic.Uint64
}

func newShellState() *shellState { return &shellState{} }

// stateProbeFn 是 EXIT trap 采集函数模板：%s = pwdCaptureExpr()（Windows 上
// 经 cygpath 转 host 形，cygpath 缺失时输出空、采纳侧跳过 cwd；其余平台直接
// 用 $PWD；%%s 是 bash printf 的字面量动词，经 Sprintf 还原）。rc 在函数入口
// 固定，转储失败不影响退出码传播。
const stateProbeFnFmt = `__gaea_capture() { __gaea_rc=$?; { printf '%%s\0' "%s"; export -p; } >"$GAEA_STATE_OUT" 2>/dev/null; exit $__gaea_rc; }
`

// pwdCaptureExpr 返回探针里捕获工作目录的表达式（按 host GOOS 选形态）。
func pwdCaptureExpr() string {
	if runtime.GOOS == "windows" {
		return `$(cygpath -w "$PWD" 2>/dev/null)`
	}
	return `$PWD`
}

// workDir 返回本次调用的起始目录：锚定 cwd 仍存在则用之（消失即自愈回落
// fallback=工作空间/进程 cwd），否则 fallback。
func (s *shellState) workDir(fallback string) string {
	if s == nil {
		return fallback
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cwd != "" {
		if fi, err := os.Stat(s.cwd); err == nil && fi.IsDir() {
			return s.cwd
		}
		s.cwd = ""
	}
	return fallback
}

// begin 把 userCmd 包进状态采集前缀，返回包裹后的命令文本、需要追加的进程
// 环境变量（GAEA_STATE_OUT 探针路径 / GAEA_STATE_ENV 上次转储）与探针路径。
// 任何一步不成立都原样返回 userCmd（调用方按 probe=="" 走无状态路径）。
func (s *shellState) begin(userCmd string) (wrapped string, extraEnv []string, probe string) {
	if s == nil {
		return userCmd, nil, ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dir == "" {
		d, err := os.MkdirTemp("", "gaea-bash-state-")
		if err != nil {
			return userCmd, nil, "" // 目录建不出：退化为无状态（旧行为）
		}
		s.dir = d
	}
	probe = filepath.Join(s.dir, fmt.Sprintf("probe-%d.bin", s.seq.Add(1)))

	var b strings.Builder
	b.WriteString(fmt.Sprintf(stateProbeFnFmt, pwdCaptureExpr()))
	b.WriteString("trap __gaea_capture EXIT\n")
	if s.env != "" {
		b.WriteString("if [ -n \"${GAEA_STATE_ENV:-}\" ]; then eval \"$GAEA_STATE_ENV\"; fi\n")
		extraEnv = append(extraEnv, "GAEA_STATE_ENV="+s.env)
	}
	b.WriteString(userCmd)
	extraEnv = append(extraEnv, "GAEA_STATE_OUT="+probe)
	return b.String(), extraEnv, probe
}

// adopt 解析探针文件（pwd\0export-dump）并采纳。文件缺失/形状不对/目录已
// 消失/转储超限都静默跳过对应部分——保守不更新。读后即删探针文件。
func (s *shellState) adopt(probe string) {
	if s == nil || probe == "" {
		return
	}
	data, err := os.ReadFile(probe)
	_ = os.Remove(probe)
	if err != nil {
		return
	}
	i := bytes.IndexByte(data, 0)
	if i < 0 {
		return
	}
	pwd, dump := string(data[:i]), data[i+1:]
	s.mu.Lock()
	defer s.mu.Unlock()
	if pwd != "" && filepath.IsAbs(pwd) {
		if fi, err := os.Stat(pwd); err == nil && fi.IsDir() {
			s.cwd = pwd
		}
	}
	if len(dump) > 0 && len(dump) <= envDumpMaxBytes && bytes.Contains(dump, []byte("declare -x")) {
		s.env = string(dump)
	}
}

// reset 清空锚定状态并删除探针目录。会话切换时经 ResetSessionState 调用；
// 在途调用的晚到 trap 写向已删目录即静默失败（保守不采纳）。
func (s *shellState) reset() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cwd, s.env = "", ""
	if s.dir != "" {
		_ = os.RemoveAll(s.dir)
		s.dir = ""
	}
}
