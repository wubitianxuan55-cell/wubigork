package characterlib

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

// 台词语音文件（内容寻址缓存）：台词+音色+描述任一变化 → 键变 → 新文件；
// 旧文件自然成孤儿（无状态可腐坏，不清理也无副作用）。试听链路先探
// 键命中直接读文件（秒开），未命中合成一次落盘，后续点试听=纯播文件。

// voiceFileDir 台词语音目录（与 portraits 同层，跟随角色库 dataDir 生命周期）。
func voiceFileDir(dataDir string) string {
	return filepath.Join(dataDir, "voices")
}

// VoiceFileKey 台词语音文件的内容寻址键：sha1(台词|音色|描述) 前 16 位 hex。
func VoiceFileKey(line, voiceID, voiceDesc string) string {
	h := sha1.Sum([]byte(line + "|" + voiceID + "|" + voiceDesc))
	return hex.EncodeToString(h[:])[:16]
}

// voiceAudioExt 语音文件扩展名（按合成返回的 MIME 定）。
func voiceAudioExt(mimeType string) string {
	if strings.Contains(strings.ToLower(mimeType), "mpeg") ||
		strings.Contains(strings.ToLower(mimeType), "mp3") {
		return ".mp3"
	}
	return ".wav"
}

// VoiceLineAudioPath 探键命中的台词语音文件路径；未命中（或空文件）返回 ""。
// 合成侧 MIME 事先未知，wav/mp3 双探测。
func (s *Store) VoiceLineAudioPath(key string) string {
	if s == nil || s.dataDir == "" || strings.TrimSpace(key) == "" {
		return ""
	}
	dir := voiceFileDir(s.dataDir)
	for _, ext := range []string{".wav", ".mp3"} {
		p := filepath.Join(dir, key+ext)
		if info, err := os.Stat(p); err == nil && info.Size() > 0 {
			return p
		}
	}
	return ""
}

// SaveVoiceLineAudio 台词语音字节落盘（原子写），返回文件路径；失败返回空串。
func (s *Store) SaveVoiceLineAudio(key, mimeType string, data []byte) string {
	if s == nil || s.dataDir == "" || strings.TrimSpace(key) == "" || len(data) == 0 {
		return ""
	}
	dir := voiceFileDir(s.dataDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	path := filepath.Join(dir, key+voiceAudioExt(mimeType))
	// 纵深防御：键是 hex 摘要，路径必落 voices/ 目录内（防路径穿越惯例同 portraits）
	if !strings.HasPrefix(path, dir+string(filepath.Separator)) {
		return ""
	}
	if err := fileutil.AtomicWrite(path, data, 0o644); err != nil {
		return ""
	}
	return path
}

// ReadVoiceLineAudio 读回台词语音文件字节（缓存命中路径）。
func (s *Store) ReadVoiceLineAudio(path string) ([]byte, error) {
	if s == nil || s.dataDir == "" || path == "" {
		return nil, fmt.Errorf("角色库未初始化")
	}
	// 只允许读 voices/ 目录内的文件（调用方路径来自本包探键，纵深防御）
	if !strings.HasPrefix(path, voiceFileDir(s.dataDir)+string(filepath.Separator)) {
		return nil, fmt.Errorf("非法语音文件路径")
	}
	return os.ReadFile(path)
}
