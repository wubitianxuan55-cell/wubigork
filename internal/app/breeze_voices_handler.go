package app

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"time"

	"github.com/gaea/gaea/internal/config"
)

// Breeze 克隆音色管理（breeze-server /v1/voices 系）。
// 音色文件由服务端写入 <voices-dir>/<name>.breeze，跨重启持久；合成时经
// OpenAI JSON 路径的 voice 字段引用（voice_model_handler → BreezeTTS）。

// breezeVoiceNameRe 服务端名称规则镜像：字母/数字/减号/下划线，≤64（名称即文件名）。
var breezeVoiceNameRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// breezeBaseURL 由配置推导 breeze-server 根地址。
func (c *core) breezeBaseURL() string {
	port := config.DefaultBreezePort
	if c.cfg != nil && c.cfg.BreezePort > 0 {
		port = c.cfg.BreezePort
	}
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}

// breezeWaitReady 服务未就绪时异步拉起并同步等待（克隆/列表都需要服务在线；
// 模型加载+预热约 6~10s，上限 20s）。
func (a *mediaState) breezeWaitReady() bool {
	if a.ttsReady("breeze") {
		return true
	}
	a.ensureLocalTTSService("breeze")
	for i := 0; i < 10 && !a.ttsReady("breeze"); i++ {
		time.Sleep(2 * time.Second)
	}
	return a.ttsReady("breeze")
}

// BreezeListVoices 列出已保存/缓存的克隆音色（GET /v1/voices，不加载模型但
// 服务必须在线）。服务未运行时返回 started=false，前端提示先启动 Breeze。
func (a *mediaState) BreezeListVoices() map[string]interface{} {
	if !a.ttsReady("breeze") {
		return map[string]interface{}{"started": false, "voices": []interface{}{}}
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(a.breezeBaseURL() + "/v1/voices")
	if err != nil {
		return map[string]interface{}{"started": true, "error": err.Error(), "voices": []interface{}{}}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return map[string]interface{}{"started": true, "error": fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)), "voices": []interface{}{}}
	}
	var voices []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&voices); err != nil {
		return map[string]interface{}{"started": true, "error": "解析音色列表失败: " + err.Error(), "voices": []interface{}{}}
	}
	return map[string]interface{}{"started": true, "voices": voices}
}

// BreezeSaveVoice 保存克隆音色（POST /v1/voices multipart）：参考音频 WAV
// 字节（base64，前端选文件/录制后转来）+ 逐字精确转写 + 名称（即文件名，
// 字母数字-_ ≤64）+ 可选描述。保存后可直接作合成 voice 使用。
func (a *mediaState) BreezeSaveVoice(name, description, refText, refAudioB64 string) map[string]interface{} {
	fail := func(err string) map[string]interface{} { return map[string]interface{}{"ok": false, "error": err} }
	name = trimBreezeVoiceName(name)
	if !breezeVoiceNameRe.MatchString(name) {
		return fail("音色名称只允许字母/数字/减号/下划线，长度 1-64")
	}
	if refText == "" {
		return fail("参考音频的转写文本不能为空（须与音频逐字一致，否则克隆劣化）")
	}
	audio, err := base64.StdEncoding.DecodeString(refAudioB64)
	if err != nil || len(audio) == 0 {
		return fail("参考音频解码失败（须为 WAV 的 base64）")
	}
	if len(audio) > 32<<20 {
		return fail("参考音频过大（上限 32MB）")
	}
	if !a.breezeWaitReady() {
		return fail("Breeze 服务未就绪（未安装或启动超时），请先在模型中心启动")
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, err := w.CreateFormFile("ref_audio", "reference.wav")
	if err != nil {
		return fail(err.Error())
	}
	if _, err := fw.Write(audio); err != nil {
		return fail(err.Error())
	}
	for k, v := range map[string]string{"ref_text": refText, "name": name, "description": description} {
		if v == "" {
			continue
		}
		if err := w.WriteField(k, v); err != nil {
			return fail(err.Error())
		}
	}
	if err := w.Close(); err != nil {
		return fail(err.Error())
	}

	client := &http.Client{Timeout: 120 * time.Second} // 首次编码含模型加载，放宽
	resp, err := client.Post(a.breezeBaseURL()+"/v1/voices", w.FormDataContentType(), &body)
	if err != nil {
		return fail("请求失败: " + err.Error())
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusOK {
		return fail(fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)))
	}
	var voice map[string]interface{}
	if err := json.Unmarshal(respBody, &voice); err != nil {
		return map[string]interface{}{"ok": true, "id": name}
	}
	return map[string]interface{}{"ok": true, "id": name, "voice": voice}
}

// BreezeDeleteVoice 从服务内存移除音色（DELETE /v1/voices/<id>）。服务端刻意
// 保留 .breeze 文件（重启即回来）——彻底删除须自行清理 voices 目录。
func (a *mediaState) BreezeDeleteVoice(id string) map[string]interface{} {
	if !breezeVoiceNameRe.MatchString(trimBreezeVoiceName(id)) {
		return map[string]interface{}{"ok": false, "error": "非法音色 id"}
	}
	if !a.ttsReady("breeze") {
		return map[string]interface{}{"ok": false, "error": "Breeze 服务未运行"}
	}
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodDelete, a.breezeBaseURL()+"/v1/voices/"+id, nil)
	if err != nil {
		return map[string]interface{}{"ok": false, "error": err.Error()}
	}
	resp, err := client.Do(req)
	if err != nil {
		return map[string]interface{}{"ok": false, "error": err.Error()}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return map[string]interface{}{"ok": resp.StatusCode == http.StatusOK, "status": resp.StatusCode, "body": string(body)}
}

// trimBreezeVoiceName 去首尾空白（名称规则校验前调用）。
func trimBreezeVoiceName(s string) string {
	b := []byte(s)
	start, end := 0, len(b)
	for start < end && (b[start] == ' ' || b[start] == '\t') {
		start++
	}
	for end > start && (b[end-1] == ' ' || b[end-1] == '\t') {
		end--
	}
	return string(b[start:end])
}
