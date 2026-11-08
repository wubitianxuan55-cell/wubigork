package tool

// 节点写作用域策略测试（2026-11-08 众神殿裁决刀4「逐调用路径包含校验」）：
// 声明内放行/声明外拒绝/父工作区边界仍先行/读透传父策略/声明基准与执行基准
// 同走 RealPath 归一（Windows 大小写不敏感）。

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteScopePolicyContainment(t *testing.T) {
	ws := t.TempDir()
	declared := filepath.Join(ws, "out")
	if err := os.MkdirAll(declared, 0o755); err != nil {
		t.Fatal(err)
	}
	parent := NewPathAccessPolicy(ws, nil, nil, nil)
	pol := NewWriteScopePolicy(parent, []string{declared})
	ctx := context.Background()

	// 声明内新建文件 → 放行。
	if err := pol.CheckWrite(ctx, filepath.Join(declared, "a.txt")); err != nil {
		t.Errorf("声明内写应放行: %v", err)
	}
	// 声明目录本身（目录写=建目录/改目录元数据） → 放行。
	if err := pol.CheckWrite(ctx, declared); err != nil {
		t.Errorf("声明根自身应放行: %v", err)
	}
	// 工作区内声明外 → 拒。
	if err := pol.CheckWrite(ctx, filepath.Join(ws, "b.txt")); err == nil {
		t.Error("声明外写应拒绝")
	}
	// 工作区外 → 父边界先行拒绝（不是节点作用域的错误文案）。
	if err := pol.CheckWrite(ctx, filepath.Join(filepath.Dir(ws), "c.txt")); err == nil {
		t.Error("工作区外写应被父边界拒绝")
	}
	// 读透传父策略不收窄：声明外文件可读。
	if err := pol.CheckRead(ctx, filepath.Join(ws, "b.txt")); err != nil {
		t.Errorf("读面应透传父策略: %v", err)
	}
	// 目录前缀包含：声明根的子目录写 → 放行。
	sub := filepath.Join(declared, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := pol.CheckWrite(ctx, filepath.Join(sub, "c.txt")); err != nil {
		t.Errorf("声明子目录写应放行: %v", err)
	}
}

// TestWriteScopePolicyCaseInsensitive 路径基准统一（裁决风险条 2c）：声明根
// 与执行目标两侧同走 RealPath 归一——既有文件的大小写变体写应命中同一根。
func TestWriteScopePolicyCaseInsensitive(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("大小写不敏感是 Windows 文件系统语义")
	}
	ws := t.TempDir()
	declared := filepath.Join(ws, "out")
	if err := os.MkdirAll(declared, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(declared, "a.txt")
	if err := os.WriteFile(existing, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	pol := NewWriteScopePolicy(NewPathAccessPolicy(ws, nil, nil, nil), []string{declared})
	upper := filepath.Join(filepath.Dir(declared), "OUT", "A.TXT") // 全大写变体
	if err := pol.CheckWrite(context.Background(), upper); err != nil {
		t.Errorf("大小写变体应命中同一声明根（RealPath 归一）: %v", err)
	}
}

// TestWriteScopePolicyUnresolvableRootDropped 解析失败的声明根丢弃——空根集
// = 全拒（宁缺勿放宽，与 NewPathAccessPolicy 同纪律），绝不静默降级为不设限。
func TestWriteScopePolicyUnresolvableRootDropped(t *testing.T) {
	parent := NewPathAccessPolicy(t.TempDir(), nil, nil, nil)
	pol := NewWriteScopePolicy(parent, []string{`Z:\nonexistent\drive\root`})
	err := pol.CheckWrite(context.Background(), filepath.Join(parent.writeRoots[0], "x.txt"))
	if err == nil {
		t.Fatal("声明根全部解析失败时应拒绝写（不得静默不设限）")
	}
}
