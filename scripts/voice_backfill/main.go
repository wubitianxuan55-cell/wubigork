// voice_backfill — 台词语音初始文件一次性补齐（真库真引擎，命令行直跑）。
//
// 为现有角色立即生成 voices/ 语音文件，不等下次启动的壳内批量：
//
//	go run ./scripts/voice_backfill
//
// 与壳内 bootstrapVoiceLineAudio 完全同一套内容寻址键
// （characterlib.VoiceFileKey：台词|音色|描述），产物互换、幂等跳过已有键。
// Breeze 未运行时按壳内同款配方拉起（bin/*/breeze-server.exe 最新版 +
// model/*.gguf q8 优先），等就绪后逐角色合成；单角色失败容忍继续。
package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/tts"
)

func resolveLine(name, ttsLine string, samples []string) string {
	line := strings.TrimSpace(ttsLine)
	if line != "" {
		return line
	}
	for _, s := range samples {
		if s = strings.TrimSpace(s); s != "" {
			return s
		}
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "…"
	}
	return fmt.Sprintf("你好，我是「%s」，很高兴认识你。", name)
}

func genderDesc(gender string) string {
	switch strings.ToLower(strings.TrimSpace(gender)) {
	case "male":
		return "沉稳清晰的青年男性声线"
	case "female":
		return "温柔清晰的青年女性声线"
	}
	return ""
}

func healthURL(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d/health", port)
}

func probe(port int) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(healthURL(port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// spawnBreeze 按壳内 breezeCmdFor 同款配方拉起 breeze-server。
func spawnBreeze(cfg *config.Config) (*exec.Cmd, error) {
	dir := config.DefaultBreezeDir
	if cfg != nil && cfg.BreezeDir != "" {
		dir = cfg.BreezeDir
	}
	exeMatches, err := filepath.Glob(filepath.Join(dir, "bin", "*", "breeze-server.exe"))
	if err != nil || len(exeMatches) == 0 {
		return nil, fmt.Errorf("未找到 breeze-server.exe（查找 %s\\bin\\*\\breeze-server.exe）", dir)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(exeMatches)))
	exe := exeMatches[0]

	matches, err := filepath.Glob(filepath.Join(dir, "model", "*.gguf"))
	if err != nil || len(matches) == 0 {
		return nil, fmt.Errorf("未找到 Breeze-TTS-2 模型 GGUF（查找 %s\\model\\*.gguf）", dir)
	}
	var q8, plain []string
	for _, m := range matches {
		l := strings.ToLower(filepath.Base(m))
		switch {
		case strings.Contains(l, "q8_0") && !strings.Contains(l, "-dd"):
			q8 = append(q8, m)
		case !strings.Contains(l, "-dd"):
			plain = append(plain, m)
		}
	}
	model := ""
	switch {
	case len(q8) > 0:
		sort.Strings(q8)
		model = q8[0]
	case len(plain) > 0:
		sort.Strings(plain)
		model = plain[0]
	default:
		return nil, fmt.Errorf("模型目录只有 -dd 变体，无可单文件加载的 GGUF")
	}

	port := config.DefaultBreezePort
	if cfg != nil && cfg.BreezePort > 0 {
		port = cfg.BreezePort
	}
	cmd := exec.Command(exe, model, "--host", "127.0.0.1", "--port", strconv.Itoa(port))
	cmd.Dir = filepath.Dir(exe)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动 breeze-server 失败: %w", err)
	}
	fmt.Printf("已拉起 breeze-server（模型 %s，端口 %d）\n", filepath.Base(model), port)
	return cmd, nil
}

func main() {
	cfg := config.Load()
	port := config.DefaultBreezePort
	if cfg != nil && cfg.BreezePort > 0 {
		port = cfg.BreezePort
	}
	dataDir := filepath.Join(config.DataRoot(), "whisper_data", "characterlib")
	s := characterlib.NewStore(dataDir)
	if s == nil {
		fmt.Fprintln(os.Stderr, "角色库打开失败：", dataDir)
		os.Exit(1)
	}
	defer func() { _ = s.Close() }()

	items, _, err := s.List("", "", false, 500, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "角色清单读取失败：", err)
		os.Exit(1)
	}
	fmt.Printf("角色库 %s：可见角色 %d 个\n", dataDir, len(items))

	if !probe(port) {
		cmd, err := spawnBreeze(cfg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Breeze 不可用：", err)
			os.Exit(2)
		}
		defer func() { _ = cmd.Process.Kill() }()
	}
	dead := false
	if !probe(port) {
		fmt.Println("等待 Breeze 就绪（模型加载中）…")
		for i := 0; i < 90 && !probe(port); i++ {
			time.Sleep(1 * time.Second)
		}
	}
	if !probe(port) {
		fmt.Fprintln(os.Stderr, "Breeze 90s 未就绪，放弃")
		os.Exit(3)
	}

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	gen, skip, fail := 0, 0, 0
	for i := range items {
		c := items[i]
		line := resolveLine(c.Name, c.TTSLine, c.DialogueSamples)
		voiceID := strings.TrimSpace(c.TTSVoiceID)
		desc := strings.TrimSpace(c.TTSDesc)
		if desc == "" {
			desc = genderDesc(c.Gender)
		}
		key := characterlib.VoiceFileKey(line, voiceID, desc)
		if s.VoiceLineAudioPath(key) != "" {
			skip++
			fmt.Printf("  跳过 %-12s 已有语音文件\n", c.Name)
			continue
		}
		p, err := tts.NewTTSProvider("breeze", tts.TTSConfig{
			BaseURL: base, Model: "breeze-tts-2",
			Voice: voiceID, VoiceDescription: desc,
		})
		if err != nil {
			fail++
			fmt.Printf("  失败 %-12s provider 构造: %v\n", c.Name, err)
			continue
		}
		audio, mime, err := p.SynthesizeWithMime(line)
		if err != nil || len(audio) == 0 {
			dead = true
			fail++
			fmt.Printf("  失败 %-12s 合成: %v\n", c.Name, err)
			continue
		}
		if path := s.SaveVoiceLineAudio(key, mime, audio); path != "" {
			gen++
			fmt.Printf("  生成 %-12s %s（%s）\n", c.Name, filepath.Base(path), line)
		} else {
			fail++
			fmt.Printf("  失败 %-12s 落盘\n", c.Name)
		}
	}
	fmt.Printf("完成：生成 %d / 已有 %d / 失败 %d（共 %d）\n", gen, skip, fail, len(items))
	if gen == 0 && skip == 0 && dead {
		os.Exit(4)
	}
}
