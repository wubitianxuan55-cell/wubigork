// Package weixin — 微信 ClawBot iLink 通道（多助手架构）
// 每个助手实例独立：Token + 人格 + 长轮询
package weixin

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
	"github.com/gaea/gaea/internal/netclient"
)

// ─── 配置 ────────────────────────────────────────────────────

type Config struct {
	ILinkURL      string // iLink API 地址
	BotToken      string // ClawBot Token
	BotID         string // Bot 用户 ID
	AssistantID   string // 绑定的助手 ID
	PersonalityID string // 绑定的人格 ID
}

func DefaultConfig() Config {
	return Config{ILinkURL: "https://ilinkai.weixin.qq.com"}
}

// ─── 回调 ────────────────────────────────────────────────────

type ChatFunc func(userMsg string, fromUser string) (reply string, err error)

// ─── Server ──────────────────────────────────────────────────

type Server struct {
	cfg     Config
	chatFn  ChatFunc
	client  *http.Client
	running atomic.Bool
	stopCh  chan struct{}

	syncBuf   string
	syncBufMu sync.Mutex
	pollTO    time.Duration
	pollCount int64

	sessionExpired atomic.Bool
}

func New(cfg Config, chatFn ChatFunc) *Server {
	if cfg.ILinkURL == "" {
		cfg.ILinkURL = "https://ilinkai.weixin.qq.com"
	}
	return &Server{
		cfg:     cfg,
		chatFn:  chatFn,
		client:  netclient.NewSimpleClient(90 * time.Second),
		stopCh:  make(chan struct{}),
		pollTO:  30 * time.Second,
	}
}

// ─── 生命周期 ────────────────────────────────────────────────

func (s *Server) Start() error {
	if s.cfg.BotToken == "" {
		return fmt.Errorf("未配置 BotToken")
	}
	s.running.Store(true)
	s.notifyStart()
	go s.pollLoop()
	slog.Info("[weixin] 助手通道启动",
		"assistant", s.cfg.AssistantID,
		"personality", s.cfg.PersonalityID,
	)
	return nil
}

func (s *Server) Stop() {
	s.running.Store(false)
	close(s.stopCh)
	s.notifyStop()
	slog.Info("[weixin] 助手通道关闭", "assistant", s.cfg.AssistantID)
}

func (s *Server) IsRunning() bool  { return s.running.Load() }

// SessionExpired 会话是否过期（getUpdates 返回 sessExp -14，需重新绑定）
func (s *Server) SessionExpired() bool { return s.sessionExpired.Load() }
func (s *Server) HasILink() bool   { return s.cfg.BotToken != "" }
func (s *Server) AssistantID() string { return s.cfg.AssistantID }

// ─── 长轮询 ──────────────────────────────────────────────────

const (
	maxFail = 5
	backoff = 30 * time.Second
	retry   = 3 * time.Second
	sessExp = -14
)

type pollReq struct {
	GetUpdatesBuf string      `json:"get_updates_buf"`
	BaseInfo      interface{} `json:"base_info"`
}

type pollResp struct {
	Ret                  int          `json:"ret"`
	ErrCode              int          `json:"errcode"`
	ErrMsg               string       `json:"errmsg"`
	Msgs                 []inboundMsg `json:"msgs"`
	GetUpdatesBuf        string       `json:"get_updates_buf"`
	SyncBuf              string       `json:"sync_buf"`
	LongPollingTimeoutMs int          `json:"longpolling_timeout_ms"`
}

type inboundMsg struct {
	FromUserID   string `json:"from_user_id"`
	ToUserID     string `json:"to_user_id"`
	ClientID     string `json:"client_id"`
	ContextToken string `json:"context_token"`
	ItemList     []struct {
		ItemType int        `json:"item_type"`
		TextItem *textItem  `json:"text_item,omitempty"`
	} `json:"item_list"`
}

type textItem struct {
	Text string `json:"text"`
}

func (s *Server) pollLoop() {
	var fails int
	timeout := s.pollTO

	for s.running.Load() {
		req := pollReq{BaseInfo: s.baseInfo()}
		s.syncBufMu.Lock()
		req.GetUpdatesBuf = s.syncBuf
		s.syncBufMu.Unlock()

		resp, err := s.getUpdates(&req, timeout)
		if err != nil {
			fails++
			slog.Error("[weixin] getUpdates 失败", "assistant", s.cfg.AssistantID, "err", err)
			if fails >= maxFail {
				s.sleepOrStop(backoff)
				fails = 0
			} else {
				s.sleepOrStop(retry)
			}
			continue
		}

		if resp.LongPollingTimeoutMs > 0 {
			timeout = time.Duration(resp.LongPollingTimeoutMs) * time.Millisecond
		}

		ec := resp.ErrCode
		if ec == 0 { ec = resp.Ret }
		if ec != 0 {
			if ec == sessExp {
				s.sessionExpired.Store(true)
				slog.Warn("[weixin] 会话过期（token 无效或需重新绑定）", "assistant", s.cfg.AssistantID)
				s.sleepOrStop(5 * time.Minute)
				fails = 0
				continue
			}
			fails++
			s.sleepOrStop(retry)
			continue
		}

		s.sessionExpired.Store(false)
		fails = 0

		if buf := resp.GetUpdatesBuf; buf != "" {
			s.syncBufMu.Lock()
			s.syncBuf = buf
			s.syncBufMu.Unlock()
		} else if buf := resp.SyncBuf; buf != "" {
			s.syncBufMu.Lock()
			s.syncBuf = buf
			s.syncBufMu.Unlock()
		}

		// 活跃诊断：每 10 次成功轮询记录一次（确认长轮询在跑）
		s.pollCount++
		if s.pollCount%10 == 0 {
			s.syncBufMu.Lock()
			bufLen := len(s.syncBuf)
			s.syncBufMu.Unlock()
			slog.Info("[weixin] 轮询活跃", "assistant", s.cfg.AssistantID, "round", s.pollCount, "syncBufLen", bufLen)
		}

		for i := range resp.Msgs {
			s.handle(&resp.Msgs[i])
		}
	}
}

func (s *Server) handle(msg *inboundMsg) {
	text := ""
	for _, item := range msg.ItemList {
		if item.ItemType == 1 && item.TextItem != nil {
			text += item.TextItem.Text
		}
	}
	if text == "" || s.chatFn == nil {
		return
	}

	slog.Info("[weixin] 收到消息", "assistant", s.cfg.AssistantID, "from", msg.FromUserID, "len", len(text))
	reply, err := s.chatFn(text, msg.FromUserID)
	if err != nil {
		slog.Error("[weixin] AI回复失败", "err", err)
		reply = "思考中…请稍后再试"
	}
	if err := s.Send(msg.FromUserID, msg.ContextToken, reply); err != nil {
		slog.Error("[weixin] 回复失败", "err", err)
	}
}

// ─── 发送 ────────────────────────────────────────────────────

func (s *Server) Send(toUser, contextToken, text string) error {
	msg := map[string]interface{}{
		"from_user_id":  "",
		"to_user_id":    toUser,
		"client_id":     genUUID(),
		"message_type":  2,
		"message_state": 2,
		"item_list": []map[string]interface{}{
			{"item_type": 1, "text_item": map[string]string{"text": text}},
		},
	}
	if contextToken != "" {
		msg["context_token"] = contextToken
	}
	body, _ := json.Marshal(map[string]interface{}{
		"msg":       msg,
		"base_info": s.baseInfo(),
	})

	_, err := s.apiPost("/ilink/bot/sendmessage", body, 20*time.Second)
	return err
}

// ─── API ─────────────────────────────────────────────────────

func (s *Server) getUpdates(req *pollReq, timeout time.Duration) (*pollResp, error) {
	body, _ := json.Marshal(req)
	respBody, err := s.apiPost("/ilink/bot/getupdates", body, timeout)
	if err != nil {
		return nil, err
	}
	var resp pollResp
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	return &resp, nil
}

func (s *Server) notifyStart() {
	body, _ := json.Marshal(map[string]interface{}{"base_info": s.baseInfo()})
	s.apiPost("/ilink/bot/msg/notifystart", body, 10*time.Second)
}

func (s *Server) notifyStop() {
	body, _ := json.Marshal(map[string]interface{}{"base_info": s.baseInfo()})
	s.apiPost("/ilink/bot/msg/notifystop", body, 10*time.Second)
}

func (s *Server) apiPost(endpoint string, body []byte, timeout time.Duration) ([]byte, error) {
	req, _ := http.NewRequest("POST", s.cfg.ILinkURL+endpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("AuthorizationType", "ilink_bot_token")
	req.Header.Set("Authorization", "Bearer "+s.cfg.BotToken)
	req.Header.Set("X-WECHAT-UIN", randomUIN())
	req.Header.Set("iLink-App-Id", "bot")
	req.Header.Set("iLink-App-ClientVersion", "2.4.3")

	c := s.client
	if timeout < c.Timeout {
		c = netclient.NewSimpleClient(timeout)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return b, nil
}

func (s *Server) baseInfo() map[string]string {
	return map[string]string{"channel_version": "2.4.3", "bot_agent": "gaea-desktop/1.0.0"}
}

func (s *Server) sleepOrStop(d time.Duration) {
	select {
	case <-s.stopCh:
	case <-time.After(d):
	}
}

// ─── 工具 ────────────────────────────────────────────────────

func genUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return base64.RawURLEncoding.EncodeToString(b)
}

func randomUIN() string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%010d", time.Now().UnixNano()%10000000000)))
}
