package main

// verify_strata_engine.go — 用 gaea 自己的 modelengine 代码路径验证
// engines.json 里的 custom-strata 条目：LoadState 采纳 → TestConnection
// 真连 127.0.0.1:8091 → BuildChatURL 装配。只读临时副本，不动真实文件。
import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gaea/gaea/internal/modelengine"
)

func main() {
	src := filepath.Join(os.Getenv("APPDATA"), "gaea", "whisper_data", "engines.json")
	tmp := filepath.Join(os.TempDir(), "engines-verify-strata.json")
	data, err := os.ReadFile(src)
	if err != nil {
		fmt.Println("READ_FAIL:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		fmt.Println("COPY_FAIL:", err)
		os.Exit(1)
	}

	m := modelengine.NewManager("", "")
	if err := m.LoadState(tmp); err != nil {
		fmt.Println("LOAD_FAIL:", err)
		os.Exit(1)
	}

	engines := m.GetEngines()
	var found *modelengine.EngineConfig
	for i := range engines {
		if engines[i].ID == "strata" {
			found = &engines[i]
		}
	}
	if found == nil {
		fmt.Println("ADOPT_FAIL: builtin strata engine missing from GetEngines")
		os.Exit(1)
	}
	fmt.Printf("SEED: label=%q base_url=%q enabled=%v is_local=%v isLocal=%v default=%q\n",
		found.Label, found.BaseURL, found.Enabled, found.IsLocal, found.Type.IsLocal(), found.DefaultModel)

	st, err := m.TestConnection(context.Background(), "strata")
	if err != nil {
		fmt.Println("TESTCONN_ERR:", err)
		os.Exit(1)
	}
	fmt.Printf("TESTCONN: connected=%v model_count=%d latency=%dms err=%q\n",
		st.Connected, st.ModelCount, st.LatencyMs, st.Error)

	chatURL, key, err := m.BuildChatURL("strata")
	if err != nil {
		fmt.Println("CHATURL_ERR:", err)
		os.Exit(1)
	}
	fmt.Printf("CHAT_URL: %s (key_empty=%v)\n", chatURL, key == "")

	if !st.Connected || st.ModelCount == 0 {
		fmt.Println("VERIFY_FAIL")
		os.Exit(1)
	}
	fmt.Println("VERIFY_OK")
}
