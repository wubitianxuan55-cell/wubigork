package attachment

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// chunkTestRoot 分块上传测试的独立上传根（与 openTestStore 同款 EvalSymlinks
// 预处理：产品护栏「根不得 symlink/junction」保持全强度）。注册表复位挂在
// 这里注册（TempDir 之后）：t.Cleanup LIFO——先清注册表关句柄，TempDir 的
// RemoveAll 才不会被仍打开的 root 句柄顶住（顺序反了就是 unlinkat 冲突）。
func chunkTestRoot(t *testing.T) string {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		chunkReg.Lock()
		for tok, s := range chunkReg.m {
			delete(chunkReg.m, tok)
			_ = s.file.Close()
			_ = s.store.root.Remove(s.tmp)
			_ = s.store.Close()
		}
		chunkReg.Unlock()
	})
	return filepath.Join(base, "uploads")
}

// resetChunkRegistry 测试间的注册表隔离：清空会话（连带关句柄删 .tmp），
// 防 TTL/清扫状态跨测试漂移。仅用于不建上传根的测试。
func resetChunkRegistry(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		chunkReg.Lock()
		for tok, s := range chunkReg.m {
			delete(chunkReg.m, tok)
			_ = s.file.Close()
			_ = s.store.root.Remove(s.tmp)
			_ = s.store.Close()
		}
		chunkReg.Unlock()
	})
}

// threeAligned 基准分片数据：3 字节对齐（每片独立 base64 可解码的前提）。
func threeAligned(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte('A' + i%26)
	}
	return b
}

func TestChunkedUploadRoundTrip(t *testing.T) {
	root := chunkTestRoot(t)
	// 5MB，两片（3MiB 对齐 + 2MiB 尾片），覆盖对齐/尾片两形态。
	data := threeAligned(5 << 20)
	const first = 3 << 20

	token, err := BeginChunk(root, "report.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if err := AppendChunk(token, base64.StdEncoding.EncodeToString(data[:first])); err != nil {
		t.Fatal(err)
	}
	if err := AppendChunk(token, base64.StdEncoding.EncodeToString(data[first:])); err != nil {
		t.Fatal(err)
	}
	path, err := FinishChunk(token)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(filepath.Base(path), "file-") || !strings.HasSuffix(path, ".pdf") {
		t.Fatalf("chunked path = %q", path)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Fatalf("round-trip mismatch: got %d bytes, want %d", len(got), len(data))
	}
	// Finish 后 .tmp 不残留，会话已注销。
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("temp file leaked: %s", e.Name())
		}
	}
	if err := AppendChunk(token, "aGk="); err == nil {
		t.Fatal("append after finish should fail")
	}
}

func TestChunkedFinishRejectsEmpty(t *testing.T) {
	root := chunkTestRoot(t)
	token, err := BeginChunk(root, "empty.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := FinishChunk(token); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty finish err = %v, want empty rejection", err)
	}
	// 空文件拒收路径也必须清掉 .tmp。
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("root not clean after empty finish: %d entries", len(entries))
	}
}

func TestChunkedAppendRejectsOverBudget(t *testing.T) {
	// 预算换成包级变量后测试收缩到 1MiB：预算检查在 DecodeString 之前，
	// 超限大串不允许先进内存再报错（生产预算 MaxChunkedFileBytes=1GiB
	// 不可测注入，真写 1GiB 的测试不成立）。
	old := chunkedBudget
	chunkedBudget = 1 << 20
	t.Cleanup(func() { chunkedBudget = old })
	root := chunkTestRoot(t)
	token, err := BeginChunk(root, "big.bin")
	if err != nil {
		t.Fatal(err)
	}
	if err := AppendChunk(token, strings.Repeat("A", (1<<20)/3*4+8)); !errors.Is(err, ErrChunkedFileTooLarge) {
		t.Fatalf("over-budget append = %v, want ErrChunkedFileTooLarge", err)
	}
	// 预算内分片照常收（错误后会话保留的契约）。
	if err := AppendChunk(token, base64.StdEncoding.EncodeToString(threeAligned(3000))); err != nil {
		t.Fatalf("in-budget append after rejection = %v", err)
	}
	if err := AbortChunk(token); err != nil {
		t.Fatal(err)
	}
}

func TestChunkedAbortCleansTemp(t *testing.T) {
	root := chunkTestRoot(t)
	token, err := BeginChunk(root, "gone.zip")
	if err != nil {
		t.Fatal(err)
	}
	if err := AppendChunk(token, base64.StdEncoding.EncodeToString([]byte("hello"))); err != nil {
		t.Fatal(err)
	}
	// 幂等：未知 token 静默成功。
	if err := AbortChunk("no-such-token"); err != nil {
		t.Fatalf("abort unknown token = %v, want nil", err)
	}
	if err := AbortChunk(token); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("abort left %d entries behind", len(entries))
	}
}

func TestChunkedUnknownTokenAppend(t *testing.T) {
	resetChunkRegistry(t)
	if err := AppendChunk("ghost", "aGk="); err == nil {
		t.Fatal("append to unknown session should fail")
	}
	if _, err := FinishChunk("ghost"); err == nil {
		t.Fatal("finish unknown session should fail")
	}
}

func TestChunkedSweepReapsIdleSessions(t *testing.T) {
	root := chunkTestRoot(t)
	token, err := BeginChunk(root, "stale.bin")
	if err != nil {
		t.Fatal(err)
	}
	// 把会话拨旧到超过 TTL，再 Begin（内含清扫）应把它收走。
	chunkReg.Lock()
	chunkReg.m[token].last = time.Now().Add(-2 * chunkIdleTTL)
	chunkReg.Unlock()
	if _, err := BeginChunk(root, "fresh.bin"); err != nil {
		t.Fatal(err)
	}
	chunkReg.Lock()
	_, alive := chunkReg.m[token]
	chunkReg.Unlock()
	if alive {
		t.Fatal("idle session survived sweep")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	// 只剩新会话的 .tmp（旧会话临时文件随清扫删除）；token 是随机 hex，
	// 靠条数断言而非名字匹配。
	if len(entries) != 1 {
		t.Fatalf("root entries after sweep = %d, want 1 (fresh session temp only)", len(entries))
	}
}
