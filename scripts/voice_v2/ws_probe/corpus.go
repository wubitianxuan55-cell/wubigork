// corpus.go — M3 中文 EOT 评测：语料构建（breeze 合成+插停顿）+ corpus 场景（逐案例新连接跑线缆面）。
//
// 边界语义：segment 间的 pause_ms_after ≥ 800ms（> speculative reopen grace）才会
// 触发真实 soft-end；期望 = 内部边界 hold（incomplete）、末边界 commit（complete）、
// ≥2000ms 间隙 = 允许成新轮。切句对错以线缆面为准：transcription.completed 的数量与内容。
// 逐边界 Smart Turn 判决（complete/incomplete + p 值 + 推理耗时）在服务端日志，
// 由 analyze 侧按「Client connected」连接切片提取。
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

type corpusSegment struct {
	Text         string `json:"text"`
	PauseMsAfter int    `json:"pause_ms_after"`
}

type corpusCase struct {
	Name     string          `json:"name"`
	Desc     string          `json:"desc"`
	Segments []corpusSegment `json:"segments"`
}

type corpusBoundary struct {
	AfterSegment int    `json:"after_segment"` // 0 基
	PauseMs      int    `json:"pause_ms"`
	Expected     string `json:"expected"` // incomplete=续听 / complete=提交
}

type corpusBuiltCase struct {
	Name          string           `json:"name"`
	Desc          string           `json:"desc"`
	Wav           string           `json:"wav"`
	ExpectedTexts []string         `json:"expected_texts"` // 规范化后的期望提交轮文本
	ExpectedTurns int              `json:"expected_turns"`
	Boundaries    []corpusBoundary `json:"boundaries"`
	Observe       bool             `json:"observe"` // 观察项不判对错
}

type corpusBuilt struct {
	Source  string            `json:"source"`
	Voice   string            `json:"voice"`
	GraceMs int               `json:"reopen_grace_ms"`
	Cases   []corpusBuiltCase `json:"cases"`
}

// breeze 合成一段文本 → 16k PCM16
func synthSegment16k(baseURL, text, voice string) ([]byte, error) {
	body, _ := json.Marshal(map[string]any{
		"model": "breeze-tts-2", "input": text, "voice": voice, "response_format": "wav",
	})
	resp, err := http.Post(baseURL+"/audio/speech", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("breeze %d: %s", resp.StatusCode, truncate(string(raw), 160))
	}
	pcm, rate, err := parseWAV(raw)
	if err != nil {
		return nil, err
	}
	if rate != pcmRate {
		pcm = resampleLinear(pcm, rate, pcmRate)
	}
	return pcm, nil
}

func buildCorpusWAVs(manifestPath, dir, baseURL, voice string) error {
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var cases []corpusCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	built := corpusBuilt{Source: manifestPath, Voice: voice, GraceMs: 800}
	for _, c := range cases {
		var full []byte
		var boundaries []corpusBoundary
		for i, seg := range c.Segments {
			pcm, err := synthSegment16k(baseURL, seg.Text, voice)
			if err != nil {
				return fmt.Errorf("%s 段 %d: %w", c.Name, i, err)
			}
			full = append(full, pcm...)
			last := i == len(c.Segments)-1
			if last {
				boundaries = append(boundaries, corpusBoundary{AfterSegment: i, PauseMs: 0, Expected: "complete"})
			} else if seg.PauseMsAfter > 0 {
				exp := "incomplete"
				if seg.PauseMsAfter >= 2000 {
					exp = "complete" // 长间隙允许成新轮
				}
				boundaries = append(boundaries, corpusBoundary{AfterSegment: i, PauseMs: seg.PauseMsAfter, Expected: exp})
				full = append(full, make([]byte, pcmRate*2*seg.PauseMsAfter/1000)...)
			}
		}
		wavPath := filepath.Join(dir, c.Name+".wav")
		if err := writeWAV(wavPath, full, pcmRate); err != nil {
			return err
		}
		// 期望提交轮：按「允许成新轮的边界」切段
		expTexts := []string{}
		cur := ""
		for i, seg := range c.Segments {
			cur += seg.Text
			if i == len(c.Segments)-1 || seg.PauseMsAfter >= 2000 {
				expTexts = append(expTexts, normalizeZh(cur))
				cur = ""
			}
		}
		observe := c.Name == "trailing_incomplete" || c.Name == "backchannel_um"
		built.Cases = append(built.Cases, corpusBuiltCase{
			Name: c.Name, Desc: c.Desc, Wav: c.Name + ".wav",
			ExpectedTexts: expTexts, ExpectedTurns: len(expTexts),
			Boundaries: boundaries, Observe: observe,
		})
		fmt.Printf("built %s（%.1fs，%d 边界）\n", c.Name, float64(len(full)/2)/pcmRate, len(boundaries))
	}
	out, _ := json.MarshalIndent(built, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "corpus_built.json"), out, 0o644); err != nil {
		return err
	}
	fmt.Printf("语料就绪：%s（%d 案例）\n", dir, len(built.Cases))
	return nil
}

type corpusRunCase struct {
	Name          string   `json:"name"`
	ExpectedTurns int      `json:"expected_turns"`
	ActualTurns   int      `json:"actual_turns"`
	Transcripts   []string `json:"transcripts"`
	MatchTexts    bool     `json:"match_texts"`
	TurnsOnly     bool     `json:"turns_only"` // 轮数对但 ASR 文本有误（EOT 语义正确）
	Observe       bool     `json:"observe"`
	ElapsedMs     int      `json:"elapsed_ms"`
	Healed        int      `json:"healed"` // 自愈次数（重连/重启）
}

func runCorpus(dir, server, outDir string, tailMs int) int {
	raw, err := os.ReadFile(filepath.Join(dir, "corpus_built.json"))
	if err != nil {
		fmt.Println("读 corpus_built.json 失败（先 -build-corpus）:", err)
		return 2
	}
	var built corpusBuilt
	if err := json.Unmarshal(raw, &built); err != nil {
		fmt.Println("corpus_built.json 解析失败:", err)
		return 2
	}
	var results []corpusRunCase
	for _, c := range built.Cases {
		pcm, _, err := loadWAVPCM(filepath.Join(dir, c.Wav))
		if err != nil {
			fmt.Println(c.Name, "读取失败:", err)
			return 2
		}
		start := time.Now()
		p, healed, runErr := runCaseWithHeal(server, outDir, tailMs, pcm, c.ExpectedTurns)
		if runErr != nil {
			fmt.Printf("CASE %s ERROR: %v（已含自愈）\n", c.Name, runErr)
		}
		actual := make([]string, 0, len(p.transFinal))
		for _, a := range p.transFinal {
			actual = append(actual, normalizeZh(a.text))
		}
		turnsOnly := len(actual) == c.ExpectedTurns && !equalSlices(actual, c.ExpectedTexts) &&
			fuzzyAllContained(c.ExpectedTexts, actual)
		match := len(actual) == c.ExpectedTurns && equalSlices(actual, c.ExpectedTexts)
		results = append(results, corpusRunCase{
			Name: c.Name, ExpectedTurns: c.ExpectedTurns, ActualTurns: len(actual),
			Transcripts: actual, MatchTexts: match, TurnsOnly: turnsOnly,
			Observe: c.Observe, ElapsedMs: int(time.Since(start).Milliseconds()),
			Healed: healed,
		})
		mark := "PASS"
		if c.Observe {
			mark = "OBS "
		} else if turnsOnly {
			mark = "PASS*" // EOT 语义对，ASR 字错
		} else if !match {
			mark = "FAIL"
		}
		fmt.Printf("CASE %-22s %s turns=%d/%d texts=%v\n", c.Name, mark, len(actual), c.ExpectedTurns, actual)
		time.Sleep(500 * time.Millisecond)
	}
	out, _ := json.MarshalIndent(results, "", "  ")
	_ = os.WriteFile(filepath.Join(outDir, "corpus_run.json"), out, 0o644)
	fail := 0
	for _, r := range results {
		if !r.Observe && !r.MatchTexts && !r.TurnsOnly {
			fail++
		}
	}
	fmt.Printf("语料跑完：%d 案例，判错 %d（详：%s）\n", len(results), fail, filepath.Join(outDir, "corpus_run.json"))
	if fail > 0 {
		return 1
	}
	return 0
}

// runCaseWithHeal：session.created 饿死（M3 实测间歇性，连接被认领但零事件上缆）
// 的自愈阶梯：直连 → 2s 后重连 → 重启 sidecar 再连。
func runCaseWithHeal(server, outDir string, tailMs int, pcm []byte, expectTurns int) (*probe, int, error) {
	healed := 0
	tryOnce := func() (*probe, error) {
		p := &probe{
			server: server, scenario: "corpus",
			speech: pcm, tailMs: tailMs, pace: 1.0, bargeAfter: 600,
			wait: 70 * time.Second, outDir: outDir,
			audio:       map[string]*bytes.Buffer{},
			expectTurns: expectTurns,
		}
		err := p.runForCorpus()
		return p, err
	}
	if p, err := tryOnce(); err == nil {
		return p, 0, nil
	} else {
		fmt.Printf("  [heal] %s 直连失败（%v，收到事件 %d 条），2s 后重试\n", time.Now().Format("15:04:05"), err, 0)
		_ = p
	}
	healed++
	time.Sleep(2 * time.Second)
	if p, err := tryOnce(); err == nil {
		return p, healed, nil
	} else {
		fmt.Printf("  [heal] %s 重试仍失败（%v），重启 sidecar\n", time.Now().Format("15:04:05"), err)
		_ = p
	}
	healed++
	if err := restartSidecar(server); err != nil {
		return nil, healed, fmt.Errorf("重启 sidecar 失败: %w", err)
	}
	p, err := tryOnce()
	if err != nil {
		return p, healed, fmt.Errorf("重启后仍失败: %w", err)
	}
	return p, healed, nil
}

// restartSidecar：按端口找 PID 杀掉，再拉起 start_spike.bat（本工具源在仓库
// scripts/voice_v2/ 下，bat 在其上一级），轮询 WS 就绪。
func restartSidecar(server string) error {
	host, port, err := splitHostPort(server)
	if err != nil {
		return err
	}
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		fmt.Sprintf("(Get-NetTCPConnection -LocalPort %s -State Listen | Select-Object -First 1).OwningProcess", port)).Output()
	if err == nil && strings.TrimSpace(string(out)) != "" {
		pid := strings.TrimSpace(string(out))
		if err := exec.Command("taskkill", "/PID", pid, "/F").Run(); err != nil {
			return fmt.Errorf("taskkill %s: %w", pid, err)
		}
		fmt.Printf("  [heal] 已杀 sidecar PID %s\n", pid)
	}
	time.Sleep(2 * time.Second)
	bat := filepath.Join("scripts", "voice_v2", "start_spike.bat")
	cmd := exec.Command("cmd", "/C", bat)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008} // DETACHED_PROCESS，父进程退出不连坐
	cmd.Stdout, _ = os.OpenFile(filepath.Join(".tmp", "voice_v2", "server_heal.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("拉起失败: %w", err)
	}
	wsURL := "ws://" + net.JoinHostPort(host, port) + "/v1/realtime"
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		c, _, err := (&websocket.Dialer{HandshakeTimeout: 2 * time.Second}).Dial(wsURL, nil)
		if err == nil {
			_ = c.Close()
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("60s 内 sidecar 未就绪")
}

func splitHostPort(server string) (string, string, error) {
	u := strings.TrimPrefix(strings.TrimPrefix(server, "ws://"), "wss://")
	i := strings.LastIndex(u, ":")
	if i < 0 {
		return "", "", fmt.Errorf("端口解析失败: %s", server)
	}
	return u[:i], u[i+1:], nil
}

// runForCorpus：corpus 变体——完成条件改为「完成轮数达标」，不做单场景断言/写报告。
func (p *probe) runForCorpus() error {
	p.respCreated = map[string]anchor{}
	p.firstText = map[string]anchor{}
	p.firstAudio = map[string]anchor{}
	p.respDone = map[string]anchor{}
	p.audioAfter = map[string]int{}
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.Dial(p.server, nil)
	if err != nil {
		return fmt.Errorf("连接 %s: %w", p.server, err)
	}
	p.conn = conn
	defer func() { _ = conn.Close() }()
	p.start = time.Now()
	errCh := make(chan error, 1)
	go p.readLoop(errCh)
	if err := p.expectSession(); err != nil {
		fmt.Printf("  [diag] 饿死连接：共收到 %d 条事件\n", len(p.report.Events))
		return err
	}
	if err := p.sendSessionUpdate(); err != nil {
		return err
	}
	p.streamSpeech(p.speech, &p.speechFirstSent, &p.speechLastSent)
	p.streamSilence(p.tailMs)
	deadline := time.Now().Add(p.wait)
	for time.Now().Before(deadline) && p.completedCount() < p.expectTurns {
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(graceMs * time.Millisecond)
	select { // 连接层若已报错（如读循环异常）带出
	case e := <-errCh:
		if e != nil {
			return e
		}
	default:
	}
	return nil
}

func (p *probe) completedCount() int {
	n := 0
	for _, a := range p.respDone {
		if doneStatusOfText(a.text) == "completed" {
			n++
		}
	}
	return n
}

func doneStatusOfText(t string) string {
	for i := 0; i < len(t); i++ {
		if t[i] == '/' {
			return t[:i]
		}
	}
	return t
}

// normalizeZh：去空白与中英标点（ASR 无标点输出，FireRed 输入也去标点，同口径）
func normalizeZh(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '，', '。', '？', '！', '、', '；', '：', ',', '.', '?', '!', ';', ':', '…', '“', '”', '"', '\'':
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// fuzzyAllContained：轮数一致且各轮期望文本被实际转写近似包含（ASR 字错容忍）。
// 近似 = 期望文本与实际文本的 Levenshtein 相似度 ≥0.8（同轮比对，顺序对应）。
func fuzzyAllContained(expected, actual []string) bool {
	if len(expected) == 0 || len(expected) != len(actual) {
		return false
	}
	for i, exp := range expected {
		if similarity(exp, actual[i]) < 0.8 {
			return false
		}
	}
	return true
}

func similarity(a, b string) float64 {
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 && len(rb) == 0 {
		return 1
	}
	d := make([][]int, len(ra)+1)
	for i := range d {
		d[i] = make([]int, len(rb)+1)
		d[i][0] = i
	}
	for j := 0; j <= len(rb); j++ {
		d[0][j] = j
	}
	for i := 1; i <= len(ra); i++ {
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			d[i][j] = minInt(d[i-1][j]+1, minInt(d[i][j-1]+1, d[i-1][j-1]+cost))
		}
	}
	dist := d[len(ra)][len(rb)]
	maxLen := maxInt(len(ra), len(rb))
	return float64(maxLen-dist) / float64(maxLen)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
