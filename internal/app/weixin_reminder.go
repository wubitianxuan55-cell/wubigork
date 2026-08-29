package app

// weixin_reminder.go — 微信「离线代办」提醒（v4.4 触点第一档）
//
// 微信文本「提醒我 …」→ 中文时间解析 → 本地持久化（JSON）→ 到点经微信回推。
// 这是路线图 §8 的「离线代办」差异化主打：官方元宝做不了桌面端的定时回推，
// gaea 桌面常驻 + 微信遥控器组合可以。回推目标 = 助手 Server 最近活跃会话
// （Push），个人小号单用户场景；重启后提醒不丢（JSON 落 whisperDataRoot）。

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ─── 数据模型 ─────────────────────────────────────────────────

// wxReminder 单条微信提醒。Status: pending / done / failed。
type wxReminder struct {
	ID          string     `json:"id"`
	Text        string     `json:"text"`        // 提醒事项（剥离触发词与时间表达后的正文）
	FireAt      time.Time  `json:"fireAt"`      // 触发时间
	AssistantID string     `json:"assistantId"` // 经哪个微信助手回推
	Source      string     `json:"source"`      // weixin（微信文本创建）/ manual（前端手建）
	Status      string     `json:"status"`
	FailCount   int        `json:"failCount"`
	CreatedAt   time.Time  `json:"createdAt"`
	SentAt      *time.Time `json:"sentAt,omitempty"`
}

const (
	wxReminderStatusPending = "pending"
	wxReminderStatusDone    = "done"
	wxReminderStatusFailed  = "failed"
	// wxReminderMaxFails 回推连续失败上限：超过标记 failed（停止重试）。
	wxReminderMaxFails = 5
)

// weixinTaskCfg 微信任务化配置（内存态默认 + JSON 持久化，重启保留）。
type weixinTaskCfg struct {
	RemindersEnabled bool `json:"remindersEnabled"`
}

func defaultWeixinTaskCfg() weixinTaskCfg { return weixinTaskCfg{RemindersEnabled: true} }

// weixinTaskCfgPath 任务化配置持久化文件（重启保留，与提醒文件同目录）。
func (a *whisperState) weixinTaskCfgPath() string {
	return filepath.Join(a.whisperDataRoot, "weixin_task.json")
}

// getWeixinTaskCfg 返回配置副本（惰性加载文件，缺省默认值）。
func (a *whisperState) getWeixinTaskCfg() weixinTaskCfg {
	a.weixinTaskCfgMu.Lock()
	defer a.weixinTaskCfgMu.Unlock()
	if !a.weixinTaskCfgInit {
		cfg := defaultWeixinTaskCfg()
		if b, err := os.ReadFile(a.weixinTaskCfgPath()); err == nil {
			_ = json.Unmarshal(b, &cfg)
		}
		a.weixinTaskCfg = cfg
		a.weixinTaskCfgInit = true
	}
	return a.weixinTaskCfg
}

// setWeixinTaskCfg 部分更新配置并落盘（cfgJSON 支持缺省保持原值）。
func (a *whisperState) setWeixinTaskCfg(cfgJSON string) error {
	var raw struct {
		RemindersEnabled *bool `json:"remindersEnabled"`
	}
	if cfgJSON != "" {
		if err := json.Unmarshal([]byte(cfgJSON), &raw); err != nil {
			return fmt.Errorf("解析微信任务配置失败: %w", err)
		}
	}
	a.weixinTaskCfgMu.Lock()
	// 不可重入：此处内联惰性加载（getWeixinTaskCfg 会再次取锁，Mutex 不可重入）。
	if !a.weixinTaskCfgInit {
		cfg := defaultWeixinTaskCfg()
		if b, err := os.ReadFile(a.weixinTaskCfgPath()); err == nil {
			_ = json.Unmarshal(b, &cfg)
		}
		a.weixinTaskCfg = cfg
	}
	if raw.RemindersEnabled != nil {
		a.weixinTaskCfg.RemindersEnabled = *raw.RemindersEnabled
	}
	next := a.weixinTaskCfg
	a.weixinTaskCfgInit = true
	a.weixinTaskCfgMu.Unlock()
	b, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.weixinTaskCfgPath(), b, 0o644)
}

// ─── 中文时间解析 ─────────────────────────────────────────────

var cnDigits = map[rune]int{
	'零': 0, '一': 1, '二': 2, '两': 2, '三': 3, '四': 4, '五': 5,
	'六': 6, '七': 7, '八': 8, '九': 9,
}

// cnNumToInt 解析 1–99 的中文数字（"三"、"十"、"十五"、"二十"、"二十三"）；
// 非中文数字表达返回 0。仅供时刻/时长字段兜底。
func cnNumToInt(s string) int {
	if s == "" {
		return 0
	}
	// 纯阿拉伯数字由正则直接捕获，这里只处理含中文数字的串。
	hasCN := false
	for _, r := range s {
		if _, ok := cnDigits[r]; ok || r == '十' {
			hasCN = true
			break
		}
	}
	if !hasCN {
		return 0
	}
	// "X十Y" / "X十" / "十Y" / "十"（'十' 为 3 字节 UTF-8）
	tenIdx := strings.IndexRune(s, '十')
	if tenIdx < 0 {
		n := 0
		for _, r := range s {
			d, ok := cnDigits[r]
			if !ok {
				return 0
			}
			n = n*10 + d
		}
		return n
	}
	tens := 0
	if tenIdx > 0 {
		if tenIdx != 3 { // 十位只允许单个数字字符（'二'十 = 20）
			return 0
		}
		d, ok := cnDigits[rune(s[0])]
		if !ok {
			return 0
		}
		tens = d * 10
	} else {
		tens = 10 // 裸「十」= 10
	}
	return tens + cnNumToInt(s[tenIdx+3:])
}

// cnNumToken 捕获「中文数字或阿拉伯数字」片段：(\d+|[零一二两三四五六七八九十]+)
var cnNumToken = `(\d+|[零一二两三四五六七八九十]+)`

// cnToInt 把捕获组统一转 int（先试阿拉伯，再试中文）。
func cnToInt(tok string) int {
	if n, err := strconv.Atoi(tok); err == nil {
		return n
	}
	return cnNumToInt(tok)
}

var (
	reRelDuration = regexp.MustCompile(
		cnNumToken + `\s*(分钟|分|个小时|小时|钟头|天)\s*(?:之后|以后|后)`)
	reDatePrefix = regexp.MustCompile(`(今天|今日|明天|明日|后天|明早|明晚)`)
	reClock      = regexp.MustCompile(
		`(凌晨|早上|上午|中午|下午|傍晚|晚上|夜里)?\s*` + cnNumToken + `\s*[点:：]\s*(半|一刻|三刻|` + cnNumToken + `\s*分?)?`)
)

// parseReminderWhen 从自然语言里解析触发时间。返回 (time, 是否绝对时间已过, 是否成立)。
// 支持三类表达（按优先级）：
//  1. 相对时长：「30分钟后」「2小时后」「1天后」
//  2. 日期前缀 + 时刻：「明天早上9点」「后天 14:30」「明晚8点半」
//  3. 裸时刻：「18:30」「9点半」「下午3点」（无段词按字面理解；已过 → 顺延明天；
//     stale=true 表示显式「今天」但时刻已过，由调用方决定拒绝或提示）
//
// 歧义约定：「明天2点半」按字面 = 凌晨2:30（无段词不猜下午）——确认回复会带
// 完整时间供用户纠正，不做玄学启发式。
func parseReminderWhen(text string, now time.Time) (time.Time, bool, bool) {
	// 「明早/明晚」拆成「明天早上/明天晚上」，让日期前缀与时刻段词同时命中
	// （否则段词被日期前缀吞掉，「明晚8点」会解析成 08:00）。
	text = strings.NewReplacer("明早", "明天早上", "明晚", "明天晚上", "今晚", "今天晚上").Replace(text)

	// ① 相对时长
	if m := reRelDuration.FindStringSubmatch(text); m != nil {
		n := cnToInt(m[1])
		if n <= 0 {
			return time.Time{}, false, false
		}
		switch m[2] {
		case "分钟", "分":
			return now.Add(time.Duration(n) * time.Minute), false, true
		case "个小时", "小时", "钟头":
			return now.Add(time.Duration(n) * time.Hour), false, true
		case "天":
			return now.AddDate(0, 0, n), false, true
		}
	}

	// 日期前缀（可与时刻分离：「明天的九点」）
	dayOffset := -1 // -1 = 无前缀
	dp := reDatePrefix.FindStringSubmatch(text)
	if dp != nil {
		switch dp[1] {
		case "今天", "今日":
			dayOffset = 0
		case "明天", "明日":
			dayOffset = 1
		case "后天":
			dayOffset = 2
		}
	}

	// ② ③ 时刻
	if m := reClock.FindStringSubmatch(text); m != nil {
		hour := cnToInt(m[2])
		if hour < 0 || hour > 23 {
			return time.Time{}, false, false
		}
		minute := 0
		switch {
		case m[3] == "半":
			minute = 30
		case m[3] == "一刻":
			minute = 15
		case m[3] == "三刻":
			minute = 45
		case m[3] != "":
			minute = cnToInt(regexp.MustCompile(`\d+|[零一二两三四五六七八九十]+`).FindString(m[3]))
		}
		if minute < 0 || minute > 59 {
			return time.Time{}, false, false
		}
		// 段词换算 12 小时制（裸数字 ≤12 时才应用；13点 不因「下午」变 25）。
		switch m[1] {
		case "凌晨":
			if hour == 12 {
				hour = 0
			}
		case "早上", "上午":
			if hour == 12 {
				hour = 0 // 「上午12点」视作 0 点（宽容处理）
			}
		case "中午":
			if hour < 12 {
				hour += 0 // 「中午12点」= 12；「中午1点」= 13
			}
		case "下午", "傍晚", "晚上", "夜里":
			if hour < 12 {
				hour += 12
			}
		}

		fire := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
		if dayOffset > 0 {
			fire = fire.AddDate(0, 0, dayOffset)
		} else if !fire.After(now) {
			if dayOffset == 0 {
				// 显式「今天」但已过 → 交给调用方提示
				return fire, true, true
			}
			fire = fire.AddDate(0, 0, 1) // 裸时刻已过 → 顺延明天
		}
		return fire, false, true
	}

	// 有日期前缀但无时刻：「明天再说」——不成立提醒
	return time.Time{}, false, false
}

// stripReminderText 剥离触发词与时间表达，得到提醒事项正文。
func stripReminderText(text string) string {
	s := text
	for _, re := range []*regexp.Regexp{reDatePrefix, reRelDuration, reClock} {
		s = re.ReplaceAllString(s, " ")
	}
	// 触发词与口头语
	for _, w := range []string{"提醒我", "提醒", "记得", "麻烦", "请", "帮我", "到时", "到时候", "叫我", "喊我"} {
		s = strings.ReplaceAll(s, w, " ")
	}
	// 分隔符残渣
	s = strings.NewReplacer("：", " ", ":", " ", "，", " ", "。", " ", "、", " ").Replace(s)
	f := strings.Fields(s)
	return strings.Join(f, " ")
}

// isReminderRequest 是否为提醒类请求（触发词粗筛：含「提醒」或「叫我」）。
func isReminderRequest(text string) bool {
	t := strings.TrimSpace(text)
	return strings.Contains(t, "提醒") || strings.Contains(t, "叫我")
}

// ─── 存储与路由 ───────────────────────────────────────────────

// remindersPath 提醒持久化文件（whisperDataRoot 下，随轻语数据根走）。
func (a *whisperState) remindersPath() string {
	return filepath.Join(a.whisperDataRoot, "weixin_reminders.json")
}

func (a *whisperState) loadReminders() {
	a.remindersMu.Lock()
	defer a.remindersMu.Unlock()
	a.remindersLoaded = true
	b, err := os.ReadFile(a.remindersPath())
	if err != nil {
		return // 首次启动无文件
	}
	var rs []wxReminder
	if err := json.Unmarshal(b, &rs); err != nil {
		slog.Warn("[wx-reminder] 提醒文件解析失败，忽略", "err", err)
		return
	}
	a.reminders = rs
}

func (a *whisperState) saveRemindersLocked() {
	rs := a.reminders
	b, err := json.MarshalIndent(rs, "", "  ")
	if err != nil {
		slog.Warn("[wx-reminder] 提醒序列化失败", "err", err)
		return
	}
	if err := os.WriteFile(a.remindersPath(), b, 0o644); err != nil {
		slog.Warn("[wx-reminder] 提醒落盘失败", "err", err)
	}
}

// addWxReminder 新建提醒并落盘。fireAt 必须在未来。
func (a *whisperState) addWxReminder(text string, fireAt time.Time, assistantID, source string) wxReminder {
	r := wxReminder{
		ID:          fmt.Sprintf("wxr-%d", time.Now().UnixNano()),
		Text:        text,
		FireAt:      fireAt,
		AssistantID: assistantID,
		Source:      source,
		Status:      wxReminderStatusPending,
		CreatedAt:   time.Now(),
	}
	a.remindersMu.Lock()
	a.reminders = append(a.reminders, r)
	a.saveRemindersLocked()
	a.remindersMu.Unlock()
	return r
}

// tryWxReminder 微信消息路由（v4.4 任务化第一档）：命中提醒类请求即处理并
// 返回 (回复, true)；未命中返回 ("", false) 走原有聊天管道。
func (a *whisperState) tryWxReminder(userMsg, assistantID string) (string, bool) {
	if !a.getWeixinTaskCfg().RemindersEnabled || !isReminderRequest(userMsg) {
		return "", false
	}
	now := time.Now()
	fire, stale, ok := parseReminderWhen(userMsg, now)
	if !ok {
		return "想帮你设提醒，但没看懂时间。可以这样说：「30分钟后喝水」「明天早上9点开会」「18:30 接孩子」。", true
	}
	if stale {
		return "这个时间已经过了哦，要设明天的同一时间吗？可以这样说：「明天" + userMsg + "」。", true
	}
	item := stripReminderText(userMsg)
	if item == "" {
		item = "（未说明事项）"
	}
	r := a.addWxReminder(item, fire, assistantID, "weixin")
	return fmt.Sprintf("好，已设提醒：%s（%s）——到点我用微信叫你。", r.Text, r.FireAt.Format("1月2日 15:04")), true
}

// ─── 到点回推循环 ─────────────────────────────────────────────

// wxPushFn 微信回推实现（默认走 weixinServers[assistantID].Push；测试注入）。
type wxPushFn func(assistantID, text string) error

func (a *whisperState) defaultWxPush(assistantID, text string) error {
	a.weixinMu.Lock()
	srv, ok := a.weixinServers[assistantID]
	a.weixinMu.Unlock()
	if !ok || srv == nil {
		return fmt.Errorf("微信助手 %s 未在运行", assistantID)
	}
	return srv.Push(text)
}

// startReminderTicker 启动到点回推循环（20s 粒度）。幂等；Shutdown 时经
// reminderStop 停止。
func (a *whisperState) startReminderTicker() {
	a.reminderOnce.Do(func() {
		stop := make(chan struct{})
		a.reminderStop = stop
		push := a.wxPushFunc
		if push == nil {
			push = a.defaultWxPush
		}
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("[wx-reminder] ticker panic recovered", "panic", r)
				}
			}()
			a.tickReminders(time.Now(), push)
			ticker := time.NewTicker(20 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					a.tickReminders(time.Now(), push)
				}
			}
		}()
		slog.Info("[wx-reminder] 到点回推循环已启动")
	})
}

// tickReminders 一轮扫描：到期 pending → 回推 → 标 done；失败计数，超限标
// failed。返回本轮成功推送条数（测试断言用）。
func (a *whisperState) tickReminders(now time.Time, push wxPushFn) int {
	if !a.getWeixinTaskCfg().RemindersEnabled {
		return 0
	}
	if push == nil {
		push = a.defaultWxPush
	}
	a.remindersMu.Lock()
	defer a.remindersMu.Unlock()
	pushed := 0
	for i := range a.reminders {
		r := &a.reminders[i]
		if r.Status != wxReminderStatusPending || r.FireAt.After(now) {
			continue
		}
		msg := "⏰ 提醒：" + r.Text
		if err := push(r.AssistantID, msg); err != nil {
			r.FailCount++
			slog.Warn("[wx-reminder] 回推失败", "id", r.ID, "fail", r.FailCount, "err", err)
			if r.FailCount >= wxReminderMaxFails {
				r.Status = wxReminderStatusFailed
				slog.Warn("[wx-reminder] 重试超限，标记失败", "id", r.ID)
			}
			continue
		}
		ts := now
		r.Status = wxReminderStatusDone
		r.SentAt = &ts
		pushed++
		slog.Info("[wx-reminder] 已回推", "id", r.ID, "assistant", r.AssistantID, "text", r.Text)
	}
	if pushed > 0 {
		a.saveRemindersLocked()
	}
	return pushed
}

// ─── 绑定面（经 VoiceB 透出，与 WhisperWeixin* 同族）───────────

// WeixinReminderList 全量提醒列表（待触发/已完成/失败，按触发时间升序）。
func (a *whisperState) WeixinReminderList() []map[string]interface{} {
	a.remindersMu.Lock()
	defer a.remindersMu.Unlock()
	out := make([]map[string]interface{}, 0, len(a.reminders))
	rs := append([]wxReminder(nil), a.reminders...)
	sort.Slice(rs, func(i, j int) bool { return rs[i].FireAt.Before(rs[j].FireAt) })
	for _, r := range rs {
		out = append(out, map[string]interface{}{
			"id": r.ID, "text": r.Text,
			"fireAt": r.FireAt.Format(time.RFC3339),
			"assistantId": r.AssistantID, "source": r.Source,
			"status": r.Status, "failCount": r.FailCount,
			"createdAt": r.CreatedAt.Format(time.RFC3339),
			"sentAt": func() interface{} {
				if r.SentAt != nil {
					return r.SentAt.Format(time.RFC3339)
				}
				return nil
			}(),
		})
	}
	return out
}

// WeixinReminderAdd 前端手动建提醒（fireAtRFC3339 为 RFC3339 时间串，必须在未来）。
func (a *whisperState) WeixinReminderAdd(text, fireAtRFC3339 string) (map[string]interface{}, error) {
	fire, err := time.Parse(time.RFC3339, fireAtRFC3339)
	if err != nil {
		return nil, fmt.Errorf("时间格式应为 RFC3339（如 2026-09-01T09:00:00+08:00）: %w", err)
	}
	if !fire.After(time.Now()) {
		return nil, fmt.Errorf("提醒时间必须在未来")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("提醒事项不能为空")
	}
	// 手动建提醒默认经核心助手 gaea 回推（未绑微信时 tick 会计失败并提示）。
	r := a.addWxReminder(text, fire, "gaea", "manual")
	return map[string]interface{}{"id": r.ID, "fireAt": r.FireAt.Format(time.RFC3339), "status": r.Status}, nil
}

// WeixinReminderDelete 删除提醒（任意状态可删）。
func (a *whisperState) WeixinReminderDelete(id string) error {
	a.remindersMu.Lock()
	defer a.remindersMu.Unlock()
	for i := range a.reminders {
		if a.reminders[i].ID == id {
			a.reminders = append(a.reminders[:i], a.reminders[i+1:]...)
			a.saveRemindersLocked()
			return nil
		}
	}
	return fmt.Errorf("提醒 %s 不存在", id)
}

// WeixinReminderConfig 微信任务化配置（当前仅提醒开关）。
func (a *whisperState) WeixinReminderConfig() (map[string]interface{}, error) {
	cfg := a.getWeixinTaskCfg()
	return map[string]interface{}{"remindersEnabled": cfg.RemindersEnabled}, nil
}

// WeixinReminderSetConfig 部分更新微信任务化配置（cfgJSON 缺省字段保持原值）。
func (a *whisperState) WeixinReminderSetConfig(cfgJSON string) error {
	return a.setWeixinTaskCfg(cfgJSON)
}
