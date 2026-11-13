package app

// office_com_settle.go — dsh-office COM 实时协同档波3：settle 回合收尾钩子。
//
// 上游语义（dsh-office 宿主在 agent idle 时做两件事，OfficeHelper.cs Run()
// 分派实测核对）：
//   1. settle 命令：helper Settle() 把静默模式后台打开的文档自动保存关闭
//     （有未保存修改先保存再关；用户中途调到前台的移交用户、只存不关；空了
//      的自家实例退出）。回执是说明行字符串数组——纯文本、无结构化路径字段，
//      实测形态："saved and closed <path>" / "closed <path>" /
//      "saved <path> (the user has it on screen)" / "left open <path> (...)" /
//      "could not close <path>: <原因>"。
//   2. card 命令 {text:"AI 已完成", hold:false}：告诉迷你卡回合结束了。helper
//      侧卡片从未启动时（本档 open/edit 恒 card=false，见 officeComSettings）
//      Report 被跳过、原样回 "ok"——容错保留调用形态对齐上游。
//
// gaea 侧挂点：gaea_handler.go 的 TurnDone 槽（与 maybeDreamAfterTurn 同列）。
// 纪律：TurnDone 每回合都触发，守卫不过（本回合无 office_* 写面活动且无静默
// 文档）必须零 helper 命令——绝不每回合 ping/拉起 helper 进程。

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"
)

// settle/card 命令超时：settle 要逐文档 COM 保存关闭取宽容值；card 跟在
// settle 后走同一活实例，短超时即可（失败不报）。
const (
	officeComSettleTimeout = 60 * time.Second
	officeComCardTimeout   = 15 * time.Second
)

// officeComSettling 单飞位：上一轮 settle 还在跑就不叠加（遗留静默文档下回合
// 再收；TurnDone 串行触发，正常不重叠，纯防御）。
var officeComSettling atomic.Bool

// maybeOfficeComSettle TurnDone 槽调用（成功回合末）：守卫通过时同步执行收尾
// （常规是亚秒级 COM 保存，60s 是病态上界——TurnDone 是回合末事件，此处短阻塞
// 不冻结进行中的对话流）。
func (a *App) maybeOfficeComSettle() {
	officeCom.mu.Lock()
	// 守卫：本回合内有 office_* 写面活动（lastActive 晚于上次 settle 尝试）
	// 或存在静默打开文档才发命令；否则连 client 都不创建（helper 进程都不拉）。
	active := officeCom.lastActive.After(officeCom.lastSettle)
	silentDocs := make(map[string]string, len(officeCom.silentOpen))
	for k, v := range officeCom.silentOpen {
		silentDocs[k] = v
	}
	officeCom.mu.Unlock()
	if !active && len(silentDocs) == 0 {
		return
	}
	if !officeComSettling.CompareAndSwap(false, true) {
		return
	}
	defer officeComSettling.Store(false)
	// 尝试时刻先记账（失败也算）：helper 起不来/超时时绝不每回合重试 ping，
	// 下一次 office_* 写面活动重新解锁。静默文档记账保留（helper 没收到命令，
	// 文档仍在后台），下回合凭 silentOpen 守卫再试。
	officeCom.mu.Lock()
	officeCom.lastSettle = time.Now()
	officeCom.mu.Unlock()

	raw, err := officeCom.call(context.Background(), "settle", nil, officeComSettleTimeout)
	if err != nil {
		slog.Debug("office settle 失败（下回合重试）", "err", err)
		return
	}
	var lines []string
	if err := json.Unmarshal(raw, &lines); err != nil {
		slog.Debug("office settle 回执解析失败（跳过证据落账）", "err", err)
	} else {
		a.officeComSettleEvidence(lines, silentDocs)
	}
	// card 收尾：容错 fire-and-forget，失败不报（纯装饰；helper 侧卡片未启动
	// 时本就跳过 Report）。
	if _, cerr := officeCom.call(context.Background(), "card",
		map[string]any{"text": "AI 已完成", "hold": false}, officeComCardTimeout); cerr != nil {
		slog.Debug("office card 收尾失败（忽略）", "err", cerr)
	}
	// 收尾完成：清静默文档记账（settle 侧已保存关闭或移交用户）。
	officeCom.mu.Lock()
	officeCom.silentOpen = nil
	officeCom.mu.Unlock()
}

// officeComSettleEvidence 消化 settle 说明行：对发生保存的行落 Journal 证据卡
// （tool=office_save，与 office_save 工具同账；COM 静默收尾没有基线快照——诚实
// 缺省）并向前端发 office_file_saved 事件（gaea-event 通道，payload path/rel
// 双键，前端线订阅消费）。
func (a *App) officeComSettleEvidence(lines []string, silentDocs map[string]string) {
	for _, line := range lines {
		parsed := parseOfficeComSettleLine(line)
		if !parsed.saved {
			continue
		}
		path := parsed.path
		if path == "" {
			// 行内路径解析不出（回执格式将来变化）：退回本方静默记账兜底。
			for _, v := range silentDocs {
				path = v
				break
			}
		}
		appendOfficeEvidence("office_save", officeComJournalRel(path),
			"COM 静默收尾（无基线快照）", line, "", "")
		// 前端事件仅在内核就绪时发（core.emit 要求存活 ctx；裸 App 测试态
		// core==nil 直接跳过，Journal 落账不受影响）。
		if a.core != nil {
			a.emit("gaea-event", map[string]interface{}{
				"kind": "office_file_saved",
				"path": path,
				"rel":  officeComJournalRel(path),
			})
		}
	}
}

// officeComSettleLine 是一条 settle 回执行的解析结果。
type officeComSettleLine struct {
	saved bool   // 是否发生保存（"saved and closed …" / "saved …"）
	path  string // 行内解析出的文档绝对路径（解析不出为空）
}

// parseOfficeComSettleLine 解析 helper Settle() 说明行（纯文本前缀+路径拼接，
// 形态见文件头注释——与 OfficeHelper.cs Settle() 逐字对齐，不猜意图）。
func parseOfficeComSettleLine(line string) officeComSettleLine {
	l := strings.TrimSpace(line)
	// 移交用户行带固定后缀（helper 字面量），先剥掉再认前缀。
	l = strings.TrimSuffix(l, " (the user has it on screen)")
	for _, p := range []struct {
		prefix string
		saved  bool
	}{
		{"saved and closed ", true},
		{"saved ", true},
		{"closed ", false},
		{"left open ", false},
	} {
		if strings.HasPrefix(l, p.prefix) {
			return officeComSettleLine{
				saved: p.saved,
				path:  strings.TrimSpace(strings.TrimPrefix(l, p.prefix)),
			}
		}
	}
	return officeComSettleLine{}
}
