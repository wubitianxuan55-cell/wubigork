// voice_design — 专属台词批量设计与落库（真库直写）。
//
//	dump:  go run ./scripts/voice_design dump
//	    打印全量角色人设（JSON）供台词设计。
//	apply: go run ./scripts/voice_design apply -f lines.json
//	    lines.json = [{"id":"...","line":"..."}]。逐角色：Get→写 ttsLine→
//	    Upsert→按新键合成落盘 voices/（与壳内同一内容寻址键，产物互换）。
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
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

type designDump struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Kind            string   `json:"kind"`
	Gender          string   `json:"gender"`
	RoleType        string   `json:"roleType"`
	Personality     string   `json:"personality"`
	Background      string   `json:"background"`
	VoiceGuide      string   `json:"voiceGuide"`
	DialogueSamples []string `json:"dialogueSamples"`
	TTSLine         string   `json:"ttsLine"`
	TTSVoiceID      string   `json:"ttsVoiceId"`
	TTSDesc         string   `json:"ttsDesc"`
}

type linePatch struct {
	ID   string `json:"id"`
	Line string `json:"line"`
}

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
	case "male", "男":
		return "沉稳清晰的青年男性声线"
	case "female", "女":
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
		return nil, fmt.Errorf("未找到模型 GGUF")
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
	var model string
	if len(q8) > 0 {
		sort.Strings(q8)
		model = q8[0]
	} else if len(plain) > 0 {
		sort.Strings(plain)
		model = plain[0]
	} else {
		return nil, fmt.Errorf("模型目录只有 -dd 变体")
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
		return nil, err
	}
	fmt.Printf("已拉起 breeze-server（模型 %s，端口 %d）\n", filepath.Base(model), port)
	return cmd, nil
}

func ensureBreeze(cfg *config.Config) (port int, cleanup func(), err error) {
	port = config.DefaultBreezePort
	if cfg != nil && cfg.BreezePort > 0 {
		port = cfg.BreezePort
	}
	if !probe(port) {
		cmd, serr := spawnBreeze(cfg)
		if serr != nil {
			return port, nil, serr
		}
		cleanup = func() { _ = cmd.Process.Kill() }
	}
	if !probe(port) {
		fmt.Println("等待 Breeze 就绪（模型加载中）…")
		for i := 0; i < 90 && !probe(port); i++ {
			time.Sleep(1 * time.Second)
		}
	}
	if !probe(port) {
		return port, cleanup, fmt.Errorf("Breeze 90s 未就绪")
	}
	return port, cleanup, nil
}

func main() {
	sub := flag.String("m", "dump", "子命令：dump | apply")
	linesFile := flag.String("f", "lines.json", "apply 模式的台词清单 JSON")
	flag.Parse()

	cfg := config.Load()
	dataDir := filepath.Join(config.DataRoot(), "whisper_data", "characterlib")
	s := characterlib.NewStore(dataDir)
	if s == nil {
		fmt.Fprintln(os.Stderr, "角色库打开失败：", dataDir)
		os.Exit(1)
	}
	defer s.Close()

	items, _, err := s.List("", "", false, 500, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "角色清单读取失败：", err)
		os.Exit(1)
	}

	switch *sub {
	case "dump":
		out := make([]designDump, 0, len(items))
		for i := range items {
			c := items[i]
			out = append(out, designDump{
				ID: c.ID, Name: c.Name, Kind: c.Kind, Gender: c.Gender,
				RoleType: c.RoleType, Personality: c.Personality,
				Background: c.Background, VoiceGuide: c.VoiceGuide,
				DialogueSamples: c.DialogueSamples, TTSLine: c.TTSLine,
				TTSVoiceID: c.TTSVoiceID, TTSDesc: c.TTSDesc,
			})
		}
		enc, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(enc))

	case "apply":
		raw, err := os.ReadFile(*linesFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "读台词清单失败：", err)
			os.Exit(1)
		}
		var patches []linePatch
		if err := json.Unmarshal(raw, &patches); err != nil {
			fmt.Fprintln(os.Stderr, "台词清单解析失败：", err)
			os.Exit(1)
		}
		byID := make(map[string]characterlib.Character, len(items))
		for i := range items {
			byID[items[i].ID] = items[i]
		}

		port, cleanup, berr := ensureBreeze(cfg)
		if cleanup != nil {
			defer cleanup()
		}
		if berr != nil {
			fmt.Fprintln(os.Stderr, "Breeze 不可用：", berr)
			os.Exit(2)
		}
		base := fmt.Sprintf("http://127.0.0.1:%d", port)

		gen, skip, fail := 0, 0, 0
		for _, p := range patches {
			line := strings.TrimSpace(p.Line)
			if line == "" {
				skip++
				continue
			}
			c, ok := byID[p.ID]
			if !ok {
				fail++
				fmt.Printf("  失败 %-12s 库内无此 id：%s\n", p.ID, line)
				continue
			}
			c.TTSLine = line
			if err := s.Upsert(&c); err != nil {
				fail++
				fmt.Printf("  失败 %-12s 落库: %v\n", c.Name, err)
				continue
			}
			voiceID := strings.TrimSpace(c.TTSVoiceID)
			desc := strings.TrimSpace(c.TTSDesc)
			if desc == "" {
				desc = genderDesc(c.Gender)
			}
			key := characterlib.VoiceFileKey(line, voiceID, desc)
			if s.VoiceLineAudioPath(key) != "" {
				skip++
				fmt.Printf("  跳过 %-12s 语音文件已存在\n", c.Name)
				continue
			}
			prov, perr := tts.NewTTSProvider("breeze", tts.TTSConfig{
				BaseURL: base, Model: "breeze-tts-2",
				Voice: voiceID, VoiceDescription: desc,
			})
			if perr != nil {
				fail++
				fmt.Printf("  失败 %-12s provider: %v\n", c.Name, perr)
				continue
			}
			audio, mime, serr := prov.SynthesizeWithMime(line)
			if serr != nil || len(audio) == 0 {
				fail++
				fmt.Printf("  失败 %-12s 合成: %v\n", c.Name, serr)
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
		fmt.Printf("完成：写台词并生成 %d / 跳过 %d / 失败 %d（清单 %d）\n", gen, skip, fail, len(patches))

	// pin: 为每个角色固化克隆音色（音色不再漂）：设计模式合成其专属台词作
	// 参考音频 → 注册为 breeze 克隆音色（.breeze 持久）→ 写回角色卡
	// ttsVoiceId → 按钉死音色重生成语音文件。聊天/流播/试听从此全链同音色。
	// 幂等：已有 ttsVoiceId 的角色跳过。
	case "pin":
		port, cleanup, berr := ensureBreeze(cfg)
		if cleanup != nil {
			defer cleanup()
		}
		if berr != nil {
			fmt.Fprintln(os.Stderr, "Breeze 不可用：", berr)
			os.Exit(2)
		}
		base := fmt.Sprintf("http://127.0.0.1:%d", port)
		client := &http.Client{Timeout: 180 * time.Second}
		pinned, skip, fail := 0, 0, 0
		for i := range items {
			c := items[i]
			if strings.TrimSpace(c.TTSVoiceID) != "" {
				skip++
				fmt.Printf("  跳过 %-12s 已有克隆音色 %s\n", c.Name, c.TTSVoiceID)
				continue
			}
			line := resolveLine(c.Name, c.TTSLine, c.DialogueSamples)
			desc := strings.TrimSpace(c.TTSDesc)
			if desc == "" {
				desc = genderDesc(c.Gender)
			}
			voiceName := "char_" + strings.Map(func(r rune) rune {
				ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_'
				if ok {
					return r
				}
				return -1
			}, c.ID)
			if len(voiceName) > 64 {
				voiceName = voiceName[:64]
			}
			// ① 设计模式合成专属台词作参考样本（ref_text 须与音频逐字一致）
			prov, perr := tts.NewTTSProvider("breeze", tts.TTSConfig{
				BaseURL: base, Model: "breeze-tts-2",
				VoiceDescription: desc,
			})
			if perr != nil {
				fail++
				fmt.Printf("  失败 %-12s provider: %v\n", c.Name, perr)
				continue
			}
			refAudio, mime, serr := prov.SynthesizeWithMime(line)
			if serr != nil || len(refAudio) == 0 || !strings.Contains(strings.ToLower(mime), "wav") {
				fail++
				fmt.Printf("  失败 %-12s 设计采样: %v（mime=%s）\n", c.Name, serr, mime)
				continue
			}
			// ② 注册为克隆音色（服务端 .breeze 文件持久，跨重启即回）
			if verr := saveBreezeVoice(client, base, voiceName, desc, line, refAudio); verr != nil {
				fail++
				fmt.Printf("  失败 %-12s 注册克隆音色: %v\n", c.Name, verr)
				continue
			}
			// ③ 写回角色卡（聊天链 voicePersonaPreset ① 即取此 id）
			c.TTSVoiceID = voiceName
			if uerr := s.Upsert(&c); uerr != nil {
				fail++
				fmt.Printf("  失败 %-12s 落库: %v\n", c.Name, uerr)
				continue
			}
			// ④ 按钉死音色重生成台词语音文件（键含音色 id，新文件）
			key := characterlib.VoiceFileKey(line, voiceName, desc)
			if s.VoiceLineAudioPath(key) == "" {
				pinProv, p2 := tts.NewTTSProvider("breeze", tts.TTSConfig{
					BaseURL: base, Model: "breeze-tts-2",
					Voice: voiceName, VoiceDescription: desc,
				})
				if p2 == nil {
					audio, m2, aerr := pinProv.SynthesizeWithMime(line)
					if aerr == nil && len(audio) > 0 {
						s.SaveVoiceLineAudio(key, m2, audio)
					}
				}
			}
			pinned++
			fmt.Printf("  钉死 %-12s %s（%s）\n", c.Name, voiceName, line)
		}
		fmt.Printf("完成：钉死 %d / 跳过 %d / 失败 %d（共 %d）\n", pinned, skip, fail, len(items))
		if fail > 0 {
			os.Exit(4)
		}

	default:
		fmt.Fprintln(os.Stderr, "未知子命令：", *sub)
		os.Exit(1)
	}
}

// saveBreezeVoice POST /v1/voices multipart（与壳内 BreezeSaveVoice 同口径）。
func saveBreezeVoice(client *http.Client, base, name, description, refText string, refAudio []byte) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("ref_audio", "reference.wav")
	if err != nil {
		return err
	}
	if _, err := fw.Write(refAudio); err != nil {
		return err
	}
	for k, v := range map[string]string{"ref_text": refText, "name": name, "description": description} {
		if v == "" {
			continue
		}
		if err := w.WriteField(k, v); err != nil {
			return err
		}
	}
	if err := w.Close(); err != nil {
		return err
	}
	resp, err := client.Post(base+"/v1/voices", w.FormDataContentType(), &buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
