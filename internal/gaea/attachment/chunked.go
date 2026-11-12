package attachment

// ── 分块上传（超大附件过桥形态）──
//
// 单次绑定调用的 25 MiB 上限（MaxFileBytes）只约束「一次 base64 全量过桥」
// ——桥载荷与前端 JS 串内存都按整包计。拖拽/粘贴进 Composer 的超大文件改走
// Begin/Append/Finish 三段式：前端按 3 字节对齐切片（base64 每片独立可解码），
// Go 侧流式落盘，任一时刻驻留内存只有单片。落盘位置与命名和 SaveFileBytes
// 同规（.gaea/uploads 下 file-<token><ext>），发送侧照旧只认 @路径。
//
// 会话生命周期：Begin 注册 → Append 累加 → Finish 落名（或 Abort 清理）。
// 前端崩溃/半途放弃的会话由 idle TTL 清扫兜底（Begin 时顺手扫一遍），避免
// 永久泄漏打开的句柄与 .tmp 文件。

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// chunkIdleTTL 无活动会话的清扫时限：前端正常上传单片间隔秒级，30 分钟
// 远超任何合法上传的静默期；超过即视为放弃（进程退出/渲染层崩溃残留）。
const chunkIdleTTL = 30 * time.Minute

// chunkSession 一次分块上传的落盘状态。store 由会话独占持有（os.Root 打开
// 一次服务整个会话），Finish/Abort 时随会话一起释放。
type chunkSession struct {
	store *Store
	file  *os.File
	tmp   string // root 内相对路径（".<token>.tmp"）
	name  string // 原始文件名（提取扩展名用，与 SaveFileBytes 同口径）
	size  int64
	last  time.Time
}

var chunkReg = struct {
	sync.Mutex
	m map[string]*chunkSession
}{m: map[string]*chunkSession{}}

var chunkRand = rand.Read

// chunkedBudget AppendChunk 的总量预算（包级变量以便测试收缩；生产恒等于
// MaxChunkedFileBytes——const 不能被测试注入，而预算测试不得真写 1 GiB）。
var chunkedBudget = MaxChunkedFileBytes

// BeginChunk 在 rootPath 上传根里开一个分块上传会话，返回会话 token。
// 每个会话独立 Open 一个 Store（root 句柄），与既有「每次绑定调用独立
// Open」的形态一致。
func BeginChunk(rootPath, fileName string) (string, error) {
	store, err := Open(rootPath)
	if err != nil {
		return "", err
	}
	token, f, err := createChunkTemp(store)
	if err != nil {
		_ = store.Close()
		return "", err
	}
	now := time.Now()
	reg := &chunkReg
	reg.Lock()
	sweepIdleChunksLocked(now)
	reg.m[token] = &chunkSession{
		store: store,
		file:  f,
		tmp:   "." + token + ".tmp",
		name:  fileName,
		last:  now,
	}
	reg.Unlock()
	return token, nil
}

// createChunkTemp 在 root 内以 O_EXCL 创建 ".<token>.tmp"（与 writeOne 同形）。
// 返回 token 与已打开的句柄；调用方失败路径负责关闭 store。
func createChunkTemp(store *Store) (string, *os.File, error) {
	for range maxCreateAttempts {
		buf := make([]byte, 12)
		if _, err := chunkRand(buf); err != nil {
			return "", nil, err
		}
		token := hex.EncodeToString(buf)
		f, err := store.root.OpenFile("."+token+".tmp", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", nil, fmt.Errorf("create chunk temp: %w", err)
		}
		return token, f, nil
	}
	return "", nil, errors.New("create unique chunk temp")
}

// AppendChunk 解码并追加一片。encoded 必须是独立可解码的 base64（前端按
// 3 字节对齐切片保证）；总量超 MaxChunkedFileBytes 时拒收（免分配预算检查
// 在 DecodeString 之前，超限大串不进内存）。出错后会话保留，由调用方决定
// Abort 或继续纠正。整个操作持注册表锁：size 累加与文件写入串行化，
// 免去跨 goroutine 的读改写竞争（前端顺序上传无争用，锁开销可忽略）。
func AppendChunk(token, encoded string) error {
	reg := &chunkReg
	reg.Lock()
	defer reg.Unlock()
	s := reg.m[token]
	if s == nil {
		return fmt.Errorf("未知分块上传会话: %s", token)
	}
	if int64(base64.StdEncoding.DecodedLen(len(encoded))) > chunkedBudget-s.size {
		return ErrChunkedFileTooLarge
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("decode chunk: %w", err)
	}
	n, err := s.file.Write(raw)
	if err == nil && n != len(raw) {
		err = io.ErrShortWrite
	}
	if err != nil {
		return fmt.Errorf("write chunk: %w", err)
	}
	s.size += int64(n)
	s.last = time.Now()
	return nil
}

// FinishChunk 关闭并发布临时文件为正式附件名（file-<token><ext>，与
// writeOne 同规），返回绝对路径。空文件拒收（与 SaveFileBytes 同契约）。
func FinishChunk(token string) (string, error) {
	reg := &chunkReg
	reg.Lock()
	s := reg.m[token]
	delete(reg.m, token)
	reg.Unlock()
	if s == nil {
		return "", fmt.Errorf("未知分块上传会话: %s", token)
	}
	return s.finish()
}

func (s *chunkSession) finish() (string, error) {
	_ = s.file.Close() // Windows 下打开句柄阻塞 rename，先关再发
	defer func() { _ = s.store.Close() }()
	if s.size == 0 {
		_ = s.store.root.Remove(s.tmp)
		return "", errors.New("file attachment is empty")
	}
	final := "file-" + s.tmp[1:len(s.tmp)-len(".tmp")] + safeFileExt(s.name)
	if err := s.store.root.Rename(s.tmp, final); err != nil {
		_ = s.store.root.Remove(s.tmp)
		return "", fmt.Errorf("publish chunked attachment: %w", err)
	}
	return filepath.Join(s.store.rootPath, final), nil
}

// AbortChunk 关闭并删除临时文件，释放会话。幂等：未知 token 视为已清理。
func AbortChunk(token string) error {
	reg := &chunkReg
	reg.Lock()
	s := reg.m[token]
	delete(reg.m, token)
	reg.Unlock()
	if s == nil {
		return nil
	}
	_ = s.file.Close()
	_ = s.store.root.Remove(s.tmp)
	_ = s.store.Close()
	return nil
}

// sweepIdleChunksLocked 清扫超过 TTL 无活动的会话（Begin 时调用，持锁）。
func sweepIdleChunksLocked(now time.Time) {
	for t, s := range chunkReg.m {
		if now.Sub(s.last) < chunkIdleTTL {
			continue
		}
		delete(chunkReg.m, t)
		_ = s.file.Close()
		_ = s.store.root.Remove(s.tmp)
		_ = s.store.Close()
	}
}
