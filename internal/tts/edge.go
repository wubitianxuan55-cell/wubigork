package tts

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

// EdgeTTS 使用微软 Edge 在线 TTS（WebSocket 协议，免费，自然度极高）
type EdgeTTS struct {
	voice string
}

const (
	edgeHost             = "speech.platform.bing.com"
	edgePath             = "/consumer/speech/synthesize/readaloud/edge/v1"
	trustedClientToken   = "6A5AA1D4EAFF4E9FB37E23D68491D6F4"
	secMSGECVersion     = "1-143.0.3650.75"
	chromiumMajorVersion = "143"
	winEpoch             = 11644473600 // Windows file time epoch offset (1601→1970)
)

// NewEdgeTTS 创建 Edge TTS 客户端
func NewEdgeTTS() *EdgeTTS {
	return &EdgeTTS{
		voice: "zh-CN-XiaoxiaoNeural",
	}
}

// Synthesize 通过 WebSocket 合成语音，返回 MP3 字节。
func (e *EdgeTTS) Synthesize(text string) ([]byte, error) {
	if text == "" {
		return nil, fmt.Errorf("朗读文本为空")
	}

	conn, err := e.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// 1. 发送配置消息
	now := dateToEdgeString()
	configMsg := fmt.Sprintf(
		"X-Timestamp:%s\r\n"+
			"Content-Type:application/json; charset=utf-8\r\n"+
			"Path:speech.config\r\n\r\n"+
			`{"context":{"synthesis":{"audio":{"metadataoptions":{"sentenceBoundaryEnabled":true,"wordBoundaryEnabled":false},"outputFormat":"audio-24khz-48kbitrate-mono-mp3"}}}}`,
		now,
	)
	if err := wsSend(conn, []byte(configMsg), wsTextFrame); err != nil {
		return nil, fmt.Errorf("Edge TTS 配置发送失败: %w", err)
	}

	// 2. 发送 SSML
	requestID := randomHex(16)
	escaped := escapeSSML(text)
	ssml := fmt.Sprintf(
		`<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='zh-CN'>`+
			`<voice name='%s'><prosody pitch='+0Hz' rate='+0%%' volume='+0%%'>%s</prosody></voice>`+
			`</speak>`,
		e.voice, escaped,
	)
	ssmlMsg := fmt.Sprintf(
		"X-RequestId:%s\r\nContent-Type:application/ssml+xml\r\nX-Timestamp:%sZ\r\nPath:ssml\r\n\r\n%s",
		requestID, now, ssml,
	)

	slog.Info("EdgeTTS 合成", "text_len", len([]rune(text)), "voice", e.voice)

	if err := wsSend(conn, []byte(ssmlMsg), wsTextFrame); err != nil {
		return nil, fmt.Errorf("Edge TTS SSML 发送失败: %w", err)
	}

	// 3. 接收响应
	var audio []byte
	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	for {
		frameType, data, err := wsRecv(conn)
		if err != nil {
			if len(audio) > 0 {
				break
			}
			return nil, fmt.Errorf("Edge TTS 接收失败: %w", err)
		}
		switch frameType {
		case wsBinaryFrame:
			// 音频数据带 header（前2字节=header长度）
			if len(data) >= 2 {
				headerLen := int(binary.BigEndian.Uint16(data[:2]))
				if headerLen < len(data) {
					body := data[headerLen:]
					audio = append(audio, body...)
				}
			}
		case wsTextFrame:
			msg := string(data)
			if strings.Contains(msg, "Path:turn.end") {
				// 流正常结束
			}
		case wsCloseFrame:
			return audio, nil
		}
		if len(audio) > 0 && frameType == wsTextFrame && strings.Contains(string(data), "Path:turn.end") {
			break
		}
	}

	if len(audio) == 0 {
		return nil, fmt.Errorf("Edge TTS 未返回音频数据")
	}

	slog.Info("EdgeTTS 完成", "bytes", len(audio))
	return audio, nil
}

// Name 返回提供者 kind（seam 提供者自注册用）。
func (e *EdgeTTS) Name() string { return "edge" }

// SynthesizeWithMime 合成语音并返回音频与 MIME（Edge 返回 MP3）。
func (e *EdgeTTS) SynthesizeWithMime(text string) ([]byte, string, error) {
	audio, err := e.Synthesize(text)
	if err != nil {
		return nil, "", err
	}
	return audio, "audio/mp3", nil
}

func init() {
	RegisterTTSProvider("edge", func(cfg TTSConfig) (TTSProvider, error) {
		return NewEdgeTTS(), nil
	})
}

// ── WebSocket 连接 ──────────────────────────────────────────

func (e *EdgeTTS) dial() (*tls.Conn, error) {
	host := edgeHost + ":443"

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", host, &tls.Config{
		ServerName: edgeHost,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return nil, fmt.Errorf("Edge TTS 连接失败: %w", err)
	}

	// Sec-MS-GEC DRM token
	secGec := generateSecMSGec()

	url := fmt.Sprintf("GET %s?TrustedClientToken=%s&ConnectionId=%s&Sec-MS-GEC=%s&Sec-MS-GEC-Version=%s HTTP/1.1\r\n",
		edgePath, trustedClientToken, randomHex(16), secGec, secMSGECVersion)

	key := make([]byte, 16)
	rand.Read(key)
	secKey := base64.StdEncoding.EncodeToString(key)

	muid := strings.ToUpper(randomHex(16))

	req := url
	req += fmt.Sprintf("Host: %s\r\n", edgeHost)
	req += "Upgrade: websocket\r\n"
	req += "Connection: Upgrade\r\n"
	req += fmt.Sprintf("Sec-WebSocket-Key: %s\r\n", secKey)
	req += "Sec-WebSocket-Version: 13\r\n"
	req += fmt.Sprintf("Cookie: muid=%s;\r\n", muid)
	req += "Pragma: no-cache\r\n"
	req += "Cache-Control: no-cache\r\n"
	req += "Origin: chrome-extension://jdiccldimpdaibmpdkjnbmckianbfold\r\n"
	req += fmt.Sprintf("User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s.0.0.0 Safari/537.36 Edg/%s.0.0.0\r\n", chromiumMajorVersion, chromiumMajorVersion)
	req += "Accept-Encoding: gzip, deflate, br, zstd\r\n"
	req += "Accept-Language: en-US,en;q=0.9\r\n"
	req += "\r\n"

	if _, err := conn.Write([]byte(req)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("WebSocket 握手发送失败: %w", err)
	}

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("WebSocket 握手响应失败: %w", err)
	}
	if resp.StatusCode != 101 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		conn.Close()
		return nil, fmt.Errorf("WebSocket 握手失败: %d %s", resp.StatusCode, string(body))
	}

	expectedAccept := computeAcceptKey(secKey)
	if resp.Header.Get("Sec-WebSocket-Accept") != expectedAccept {
		conn.Close()
		return nil, fmt.Errorf("WebSocket Accept 校验失败")
	}

	return conn, nil
}

// ── Sec-MS-GEC DRM ──────────────────────────────────────────

func generateSecMSGec() string {
	now := time.Now().UTC().Unix()
	ticks := now + winEpoch       // Unix → Windows file time (seconds)
	ticks -= ticks % 300          // round down to nearest 5 min
	ticksNs := ticks * 10_000_000 // 100-nanosecond intervals

	strToHash := fmt.Sprintf("%d%s", ticksNs, trustedClientToken)
	hash := sha256.Sum256([]byte(strToHash))
	return strings.ToUpper(fmt.Sprintf("%x", hash))
}

// ── WebSocket 帧收发 ─────────────────────────────────────────

const (
	wsTextFrame   = 0x1
	wsBinaryFrame = 0x2
	wsCloseFrame  = 0x8
)

func wsSend(conn net.Conn, data []byte, opcode byte) error {
	var maskKey [4]byte
	rand.Read(maskKey[:])

	length := len(data)
	var header []byte

	header = append(header, 0x80|opcode)

	switch {
	case length <= 125:
		header = append(header, 0x80|byte(length))
	case length <= 65535:
		header = append(header, 0x80|126)
		ext := make([]byte, 2)
		binary.BigEndian.PutUint16(ext, uint16(length))
		header = append(header, ext...)
	default:
		header = append(header, 0x80|127)
		ext := make([]byte, 8)
		binary.BigEndian.PutUint64(ext, uint64(length))
		header = append(header, ext...)
	}

	header = append(header, maskKey[:]...)

	msg := append(header, data...)
	for i := range data {
		msg[len(header)+i] ^= maskKey[i%4]
	}

	_, err := conn.Write(msg)
	return err
}

func wsRecv(conn net.Conn) (opcode byte, data []byte, err error) {
	header := make([]byte, 2)
	if _, err = io.ReadFull(conn, header); err != nil {
		return 0, nil, err
	}

	opcode = header[0] & 0x0F
	masked := header[1]&0x80 != 0
	length := uint64(header[1] & 0x7F)

	switch {
	case length == 126:
		ext := make([]byte, 2)
		if _, err = io.ReadFull(conn, ext); err != nil {
			return 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(ext))
	case length == 127:
		ext := make([]byte, 8)
		if _, err = io.ReadFull(conn, ext); err != nil {
			return 0, nil, err
		}
		length = binary.BigEndian.Uint64(ext)
	}

	var maskKey [4]byte
	if masked {
		if _, err = io.ReadFull(conn, maskKey[:]); err != nil {
			return 0, nil, err
		}
	}

	data = make([]byte, length)
	if _, err = io.ReadFull(conn, data); err != nil {
		return 0, nil, err
	}

	if masked {
		for i := range data {
			data[i] ^= maskKey[i%4]
		}
	}

	return opcode, data, nil
}

func computeAcceptKey(key string) string {
	h := sha1.New()
	h.Write([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// ── 辅助 ──────────────────────────────────────────────────────

func dateToEdgeString() string {
	return time.Now().UTC().Format("Mon Jan 02 2006 15:04:05 GMT+0000 (Coordinated Universal Time)")
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func escapeSSML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// EdgeVoices 可用的中文神经语音
func EdgeVoices() []string {
	return []string{
		"zh-CN-XiaoxiaoNeural",
		"zh-CN-YunxiNeural",
		"zh-CN-XiaoyiNeural",
		"zh-CN-YunjianNeural",
		"zh-CN-XiaochenNeural",
	}
}

// ── 接口 ──────────────────────────────────────────────────────

var _ Synthesizer = (*EdgeTTS)(nil)

type Synthesizer interface {
	Synthesize(text string) ([]byte, error)
}

type SynthesizerChain struct {
	engines []Synthesizer
}

func NewSynthesizerChain(engines ...Synthesizer) *SynthesizerChain {
	return &SynthesizerChain{engines: engines}
}

func (c *SynthesizerChain) SynthesizeWithMeta(text string, metas []struct {
	Label  string
	Format string
}) ([]byte, string, string, error) {
	for i, eng := range c.engines {
		result, err := eng.Synthesize(text)
		if err == nil {
			format := "wav"
			label := "unknown"
			if i < len(metas) {
				format = metas[i].Format
				label = metas[i].Label
			}
			return result, format, label, nil
		}
		slog.Warn("TTS引擎失败，尝试下一个", "index", i, "error", err)
	}
	return nil, "", "", fmt.Errorf("所有 TTS 引擎均失败")
}
