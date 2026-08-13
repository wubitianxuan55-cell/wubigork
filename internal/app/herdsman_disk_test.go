package app

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ── 磁盘信息（E1-4） ───────────────────────────────────────────

func TestHerdsmanDiskInfo(t *testing.T) {
	orig := diskFreeFn
	defer func() { diskFreeFn = orig }()

	t.Run("正常", func(t *testing.T) {
		diskFreeFn = func(path string) (uint64, uint64, error) {
			if path == "" {
				t.Errorf("path 不应为空")
			}
			return 512 << 30, 128 << 30, nil // 512GB 总量 / 128GB 余量
		}
		total, free, err := herdsmanDiskInfo()
		if err != nil || total != 512<<30 || free != 128<<30 {
			t.Fatalf("herdsmanDiskInfo = (%d,%d,%v), want (512GB,128GB,nil)", total, free, err)
		}
	})

	t.Run("探测失败透出错误", func(t *testing.T) {
		diskFreeFn = func(path string) (uint64, uint64, error) {
			return 0, 0, errors.New("disk probe boom")
		}
		if _, _, err := herdsmanDiskInfo(); err == nil {
			t.Fatal("探测失败应返回错误")
		}
	})

	t.Run("超出 int64 报错", func(t *testing.T) {
		diskFreeFn = func(path string) (uint64, uint64, error) {
			return 1 << 63, 1, nil // total 超过 int64 上限
		}
		if _, _, err := herdsmanDiskInfo(); err == nil {
			t.Fatal("超出 int64 应报错")
		}
	})
}

// TestHerdsmanCatalogDiskSummary 验证模型库载荷包含已装占用与磁盘容量/余量。
func TestHerdsmanCatalogDiskSummary(t *testing.T) {
	origCLI := herdsmanCLI
	origDisk := diskFreeFn
	defer func() {
		herdsmanCLI = origCLI
		diskFreeFn = origDisk
	}()

	herdsmanCLI = func(args ...string) ([]byte, error) {
		return []byte(herdsmanCatalogFixture), nil
	}
	diskFreeFn = func(path string) (uint64, uint64, error) {
		return 1 << 40, 200 << 30, nil // 1TB / 200GB
	}

	catalog, err := (&App{}).HerdsmanModelCatalog()
	if err != nil {
		t.Fatalf("HerdsmanModelCatalog: %v", err)
	}
	// 夹具中已安装：bge-m3(437778496) + HauhauCS(23424536704) + zimage-turbo(20027974026)。
	want := int64(437778496 + 23424536704 + 20027974026)
	if catalog.InstalledBytes != want {
		t.Errorf("InstalledBytes = %d, want %d", catalog.InstalledBytes, want)
	}
	if catalog.DiskTotal != 1<<40 || catalog.DiskFree != 200<<30 {
		t.Errorf("DiskTotal/DiskFree = %d/%d, want 1TB/200GB", catalog.DiskTotal, catalog.DiskFree)
	}
	if catalog.DiskError != "" {
		t.Errorf("DiskError 应为空: %q", catalog.DiskError)
	}

	// JSON 序列化字段名（用带磁盘信息的首个 catalog）。
	b, _ := json.Marshal(catalog)
	for _, k := range []string{"installed_bytes", "disk_total", "disk_free"} {
		if !strings.Contains(string(b), k) {
			t.Errorf("catalog JSON 缺少字段 %q", k)
		}
	}

	// 磁盘探测失败：目录可用但 KPI 降级（DiskError 非空，不阻塞目录）。
	diskFreeFn = func(path string) (uint64, uint64, error) {
		return 0, 0, errors.New("probe fail")
	}
	catalog, err = (&App{}).HerdsmanModelCatalog()
	if err != nil {
		t.Fatalf("磁盘失败不应阻塞目录: %v", err)
	}
	if catalog.DiskError == "" {
		t.Error("磁盘探测失败应记录 DiskError")
	}
	if catalog.DiskTotal != 0 || catalog.DiskFree != 0 {
		t.Errorf("磁盘失败时应为零值: %d/%d", catalog.DiskTotal, catalog.DiskFree)
	}
}

// ── 生命周期操作串行化（E1-4） ─────────────────────────────────

// TestHerdsmanOpsSerialized 验证四个生命周期操作并发发起时被串行执行
// （对齐 herdsman local_concurrency=1；同时只有一个 CLI 调用在飞行）。
func TestHerdsmanOpsSerialized(t *testing.T) {
	orig := herdsmanCLIWithTimeout
	defer func() { herdsmanCLIWithTimeout = orig }()

	var inFlight, maxInFlight atomic.Int64
	var wg sync.WaitGroup
	herdsmanCLIWithTimeout = func(timeout time.Duration, args ...string) ([]byte, error) {
		cur := inFlight.Add(1)
		defer inFlight.Add(-1)
		for {
			old := maxInFlight.Load()
			if cur <= old || maxInFlight.CompareAndSwap(old, cur) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond) // 模拟昂贵 CLI
		return []byte(`{"ok":true,"result":{"status":"completed"}}`), nil
	}

	a := &App{}
	ops := []func() (HerdsmanOpResult, error){
		func() (HerdsmanOpResult, error) { return a.HerdsmanModelStart("m1") },
		func() (HerdsmanOpResult, error) { return a.HerdsmanModelStop("m2") },
		func() (HerdsmanOpResult, error) { return a.HerdsmanModelDownload("m3") },
		func() (HerdsmanOpResult, error) { return a.HerdsmanModelUninstall("m4") },
	}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, op := range ops {
				if _, err := op(); err != nil {
					t.Errorf("op 失败: %v", err)
				}
			}
		}()
	}
	wg.Wait()

	if max := maxInFlight.Load(); max != 1 {
		t.Fatalf("并发执行时最大在飞操作数 = %d, want 1（必须串行）", max)
	}
}
