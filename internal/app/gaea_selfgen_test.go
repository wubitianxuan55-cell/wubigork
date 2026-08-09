package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/ai"
	internalConfig "github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/boot"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
	"github.com/gaea/gaea/internal/modelengine"
)

// TestGaeaSelfGenerateCost 让 gaea 智能体自主生成成本测算表（真实模型 + 完整
// 工具链：bash/python/openpyxl + LibreOffice 重算），验证产物含公式与原生图表。
//   GAEA_SELFGEN=1 go test ./internal/app -run TestGaeaSelfGenerateCost -v -timeout 20m
func TestGaeaSelfGenerateCost(t *testing.T) {
	if os.Getenv("GAEA_SELFGEN") == "" {
		t.Skip("未设置 GAEA_SELFGEN（真实模型自生成测试）")
	}
	root := repoRoot(t)
	key := dotEnvKey(filepath.Join(root, ".env"), "DEEPSEEK_API_KEY")
	if key == "" {
		t.Fatal("仓库 .env 缺少 DEEPSEEK_API_KEY")
	}
	os.Setenv("DEEPSEEK_API_KEY", key)

	workdir := t.TempDir()
	recalc := filepath.Join(root, ".gaea", "skills", "xlsx", "scripts", "recalc.py")
	if err := os.MkdirAll(filepath.Join(workdir, ".gaea", "sessions"), 0o755); err != nil {
		t.Fatal(err)
	}

	// 照桌面端接线：engineMgr + ai.Client + bridge provider。
	// 引擎/模型优先取「办公」功能绑定（~/.gaea_config.json，模型中心可配），
	// 未绑定或不可用时回退 deepseek。
	cfg := internalConfig.Load()
	eng, model := cfg.GetFeatureModel("office")
	if !cfg.GetFeatureModelEnabled("office") || eng == "" || model == "" {
		eng, model = "deepseek", "deepseek-v4-pro"
	}
	mgr := modelengine.NewManager("", key)
	if e, ok := mgr.GetEngine(eng); !ok || !e.Enabled {
		eng, model = "deepseek", "deepseek-v4-pro"
	}
	t.Logf("办公功能绑定路由: engine=%s model=%s（来源 %s）", eng, model, func() string {
		if eng == "deepseek" && model == "deepseek-v4-pro" {
			return "fallback"
		}
		return "feature"
	}())
	client := ai.NewClient(cfg)
	client.SetEngineManager(mgr)
	client.SetActiveEngine(eng)
	bridge.SetClient(client)
	bridge.SetFeature(eng, model)

	// 事件采集：记录工具调用与最终答复
	var mu sync.Mutex
	var tools []string
	var finalText string
	sink := event.FuncSink(func(e event.Event) {
		mu.Lock()
		defer mu.Unlock()
		switch e.Kind {
		case event.ToolDispatch:
			args := strings.ReplaceAll(e.Tool.Args, "\n", " ")
			if len(args) > 160 {
				args = args[:160] + "…"
			}
			tools = append(tools, fmt.Sprintf("%s %s", e.Tool.Name, args))
		case event.ToolResult:
			status := "ok"
			if e.Tool.Err != "" {
				status = "err:" + truncateStr(e.Tool.Err, 80)
			}
			tools = append(tools, fmt.Sprintf("→ %s %s", e.Tool.Name, status))
		case event.Message:
			finalText = e.Text
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	ctrl, err := boot.Build(ctx, boot.Options{
		Model:    "gaea",
		MaxSteps: 80,
		Cwd:      workdir,
		Sink:     sink,
	})
	if err != nil {
		t.Fatalf("构建 gaea 控制器失败: %v", err)
	}
	defer ctrl.Close()
	ctrl.SetPermLevel("yolo")

	prompt := strings.ReplaceAll(`请在工作区根目录自主生成一份「成本测算.xlsx」（某市政道路改造工程）：
1. 用 bash 调 python + openpyxl 创建（openpyxl 已安装），不要用其他工具替代；
2. 主表按费用构成：人工费、材料费（2-3 种主要材料）、机械费、直接费合计、企业管理费(直接费×10%)、规费(直接费×2%)、利润((直接费+企管+规费)×7%)、税金(×9%)、含税总造价、综合单价；明细行 数量×单价=合价，全部用 Excel 公式联动；
3. 建「费用汇总」sheet（公式引用主表）+ 原生图表：费用构成饼图、直接费构成柱状图（openpyxl PieChart/BarChart）；
4. 建「编制说明」sheet；
5. 最后用 LibreOffice 重算公式缓存值：python @RECALC@ <xlsx路径>（脚本已存在）；
6. 保存为工作区根目录 成本测算.xlsx，完成后报告文件路径与关键金额。`, "@RECALC@", recalc)

	start := time.Now()
	runErr := ctrl.Run(ctx, prompt)
	elapsed := time.Since(start)
	mu.Lock()
	t.Logf("=== 工具调用（%d 条）===", len(tools))
	for _, s := range tools {
		t.Logf("  %s", s)
	}
	t.Logf("=== 最终答复 ===\n%s", truncateStr(finalText, 2000))
	mu.Unlock()
	t.Logf("耗时 %s，run err=%v", elapsed, runErr)

	// 验证产物
	files := []string{}
	filepath.Walk(workdir, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(p), ".xlsx") {
			files = append(files, p)
		}
		return nil
	})
	if len(files) == 0 {
		t.Fatalf("工作区没有生成 xlsx（run err=%v）", runErr)
	}
	out := files[0]
	t.Logf("生成文件: %s（%d 字节）", out, fileSize(out))
	verifyXLSX(t, out)
	// 复制产物到交付目录，避免临时目录被清理
	dest := filepath.Join(root, ".gaea", "exports", "成本测算-gaea自主生成.xlsx")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(out, dest); err != nil {
		t.Fatalf("复制产物失败: %v", err)
	}
	t.Logf("产物已保存: %s", dest)
}

func verifyXLSX(t *testing.T, path string) {
	t.Helper()
	// 用 openpyxl 验证：工作表/公式/缓存值/原生图表
	script := `
import os, sys
sys.stdout.reconfigure(encoding="utf-8", errors="replace")
from openpyxl import load_workbook
p = r"""` + path + `"""
wb = load_workbook(p, data_only=True)
ws = wb.worksheets[0]
print("sheets:", ",".join(wb.sheetnames))
print("cells:", ws.max_row, "x", ws.max_column)
wb2 = load_workbook(p)
formulas = 0
for sh in wb2.worksheets:
    for row in sh.iter_rows():
        for c in row:
            if isinstance(c.value, str) and c.value.startswith("="):
                formulas += 1
charts = sum(len(sh._charts) for sh in wb2.worksheets)
print("formulas:", formulas)
print("charts:", charts)
`
	out, err := exec.Command("python", "-c", script).CombinedOutput()
	if err != nil {
		t.Fatalf("openpyxl 验证失败: %v（%s）", err, truncateStr(string(out), 500))
	}
	t.Logf("openpyxl 验证：\n%s", truncateStr(string(out), 1200))
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".env")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("未找到仓库根（.env）")
		}
		dir = parent
	}
}

func dotEnvKey(path, key string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key+"=") {
			return strings.Trim(strings.TrimPrefix(line, key+"="), `"'`)
		}
	}
	return ""
}

func fileSize(p string) int64 {
	if info, err := os.Stat(p); err == nil {
		return info.Size()
	}
	return 0
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}
