package proc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

// 执行会话（codex 蒸馏 T13，unified_exec 同构）：长命令「边跑边看」的会话层。
// 双缓冲是「不重复输出」的关键——pending 消费式（取走即清，续取不重复），
// transcript 累积式（最终记录）。两缓冲在同一把锁里一起追加，否则取消时可能
// 只写进一个，出现「模型看到过但最终记录里没有」。双闸门：时间预算与输出量
// 预算谁先到谁生效；进程退出后有收尾上限（100ms，上游 POST_EXIT_CLOSE_WAIT_CAP
// =50ms 的保守版），防止后代持有管道导致永久等待。
//
// 开关：本层纯新增，不接线 = 零行为变化（现有「跑完取结果」路径原样）。

// 会话级常量。
const (
	// PostExitCloseWait 是进程退出后等待管道收尾的上限。
	PostExitCloseWait = 100 * time.Millisecond
	// MaxStdinApprovalBytes 是向交互式命令写 stdin 的审批阈值：超过即要求
	// 上层走审批，不直接写入（防把大段内容灌进 vim/python -i）。
	MaxStdinApprovalBytes = 8000
	// MinYieldMS 是时间预算下限：防止调用方传 0 导致空转。
	MinYieldMS = 10
	// sessionBufferBytes 是 pending/transcript 的容量（上游双缓冲都有上限）。
	sessionBufferBytes = 1 << 20
)

// ErrStdinApproval 是 stdin 超过审批阈值的哨兵错误（上层据此走审批路径）。
var ErrStdinApproval = errors.New("stdin exceeds approval threshold — requires user approval before writing")

// chunkID 生成 6 位十六进制随机串（日志关联用）。
func chunkID() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Session 是一个仍在运行/已退出的命令会话。
type Session struct {
	ChunkID string
	PID     int
	CallID  string
	EnvID   string

	mu         sync.Mutex
	pending    *HeadTail // 消费式：Drain 取走即清
	transcript *HeadTail // 累积式：最终记录
	stdin      io.WriteCloser
	exited     bool
	waitErr    error

	cmd *exec.Cmd
}

// StartSession 启动一个命令会话：进程跑起来、输出在后台持续落双缓冲。
// argv 原样交给 exec（不做 shell 解析）；ctx 取消即杀进程树。
func StartSession(ctx context.Context, callID, envID, cwd string, argv []string) (*Session, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = cwd
	cmd.Env = CommandEnv()
	HideWindow(cmd)
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = cmd.Stdout // 合并 stderr 进同一会话流
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	// ctx 取消即杀进程树（会话随宿主上下文消亡）。
	go func() {
		<-ctx.Done()
		KillTree(cmd)
	}()
	s := &Session{
		ChunkID:    chunkID(),
		PID:        cmd.Process.Pid,
		CallID:     callID,
		EnvID:      envID,
		pending:    NewHeadTail(sessionBufferBytes),
		transcript: NewHeadTail(sessionBufferBytes),
		stdin:      stdinPipe,
		cmd:        cmd,
	}
	// 后台持续收集：输出同锁追加进双缓冲；管道关闭（进程退出，或后代持有
	// 管道时迟迟不关）即标退出——exited 的真源在 read 侧，不在 Wait。
	go func() {
		buf := make([]byte, 16<<10)
		for {
			n, rerr := stdoutPipe.Read(buf)
			if n > 0 {
				s.append(buf[:n])
			}
			if rerr != nil {
				s.mu.Lock()
				s.exited = true
				s.mu.Unlock()
				return
			}
		}
	}()
	return s, nil
}

// append 同锁追加双缓冲（取消不会劈开一次更新）。
func (s *Session) append(p []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending.Write(p)
	s.transcript.Write(p)
}

// Drain 取走 pending 增量（消费式）：连续调用内容互不重叠，
// 拼接等于 Transcript 的出现序。maxBytes<=0 取全部；超预算的尾部留在
// pending，下次续取（不重复）。
func (s *Session) Drain(maxBytes int) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.pending.Transcript()
	if maxBytes > 0 && len(out) > maxBytes {
		s.pending = NewHeadTail(sessionBufferBytes)
		s.pending.Write(out[maxBytes:])
		return out[:maxBytes]
	}
	s.pending = NewHeadTail(sessionBufferBytes)
	return out
}

// Transcript 返回累积式完整记录（头尾保留 + 省略标记）。
func (s *Session) Transcript() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.transcript.Transcript()
}

// Exited 报告进程是否已退出（真源在管道 read 侧）。
func (s *Session) Exited() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exited
}

// Wait 等进程退出并收割（防僵尸），退出后最多再等 PostExitCloseWait 让读侧
// 落完管道余量——后代持有管道时不永久等待。
func (s *Session) Wait() error {
	werr := s.cmd.Wait()
	time.Sleep(PostExitCloseWait)
	s.mu.Lock()
	s.exited = true
	if s.waitErr == nil {
		s.waitErr = werr
	}
	s.mu.Unlock()
	return werr
}

// WriteStdin 向会话写 stdin 并收集一段增量（续取交互式命令）。
// 超 MaxStdinApprovalBytes 返回 ErrStdinApproval（审批路径由上层处理）；
// 会话已退出返回明确错误，不静默成功。
func (s *Session) WriteStdin(data []byte, yieldMS int) ([]byte, error) {
	if s.Exited() {
		return nil, fmt.Errorf("session %s already exited — cannot write stdin", s.ChunkID)
	}
	if len(data) > MaxStdinApprovalBytes {
		return nil, ErrStdinApproval
	}
	if yieldMS < MinYieldMS {
		yieldMS = MinYieldMS
	}
	if _, err := s.stdin.Write(data); err != nil {
		return nil, err
	}
	return s.Collect(yieldMS), nil
}

// Collect 在时间预算窗口内收集增量并取走（双闸门之一：时间先到先返回）。
// 有增量即小批凑批返回，不是干等满窗口。
func (s *Session) Collect(yieldMS int) []byte {
	if yieldMS < MinYieldMS {
		yieldMS = MinYieldMS
	}
	deadline := time.After(time.Duration(yieldMS) * time.Millisecond)
	for {
		select {
		case <-deadline:
			return s.Drain(0)
		default:
		}
		if s.pendingSize() > 0 {
			// 已有增量：给 20ms 凑批后返回。
			time.Sleep(20 * time.Millisecond)
			return s.Drain(0)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// pendingSize 返回未消费增量的字节数。
func (s *Session) pendingSize() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pending.Size()
}

// RunSession 是单次入口：启动会话、在双闸门窗口内收集、返回 (chunk, running)。
// 到点返回当前增量（不是结束进程）；进程仍在跑则 running=true，续取走
// WriteStdin / Collect。
func RunSession(ctx context.Context, callID, envID, cwd string, argv []string, yieldMS, maxOutputBytes int) (chunk []byte, running bool, sid string, err error) {
	s, err := StartSession(ctx, callID, envID, cwd, argv)
	if err != nil {
		return nil, false, "", err
	}
	if yieldMS < MinYieldMS {
		yieldMS = MinYieldMS
	}
	deadline := time.After(time.Duration(yieldMS) * time.Millisecond)
	for {
		select {
		case <-deadline:
			return s.Drain(maxOutputBytes), !s.Exited(), s.ChunkID, nil
		default:
		}
		// 输出量预算先到也返回（pending 实际字节数，不是累计总量）。
		if maxOutputBytes > 0 && s.pendingSize() >= maxOutputBytes {
			return s.Drain(maxOutputBytes), !s.Exited(), s.ChunkID, nil
		}
		time.Sleep(10 * time.Millisecond)
	}
}
