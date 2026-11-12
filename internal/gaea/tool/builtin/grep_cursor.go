package builtin

// grep 自校验续读游标（D2，dsh-ops 蒸馏 2026-11-10 第二波）。
//
// token=`g1.<bind16>.<snap16>.<next>.<mac16>`（`.` 分隔——防 compressGrep 的
// `:\d+:` 启发式把带数字段误认成命中行）：
//   - bind=sha256(pattern+root+include+围栏版本+跳过集版本) 截 16 hex——
//     「续读不授予任何权限」：root/pattern 永远取自本次参数，token 只比对；
//   - snap=前缀候选文件 (rel,size,mtimeNano) 流式哈希截 16 hex——树身份；
//   - next=下一页的命中偏移（与裸 offset 同单位）；
//   - mac=进程级随机钥 HMAC 截 16 hex——防伪造；重启后 token 自然失效
//     （如实报过期；不加会话绑定，见 distill §8.6 如实缩小清单）。
//
// 发射条件=页满+零缺口+未取消（接 D5：缺口页覆盖是下界，续页会静默跳过
// 没看到的文件）。校验点两道：参数解析后验 mac+bind；遍历至边界文件比对
// snap（树已变=conflict，去掉 cursor 重跑）。

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"os"
	"strconv"
	"strings"
	"sync"
)

// grepCursorDomain 是遍历域标签（对位上游 s1/s2 域隔离：walk 域恒 g1）。
const grepCursorDomain = "g1"

// 围栏/跳过集规则版本：规则变更时 bump，旧 token 全量失效（防「旧游标配新
// 围栏」的覆盖漂移）。
const (
	grepFenceRulesVer = "fence-v1"
	grepSkipRulesVer  = "skip-v1"
)

var (
	grepCursorKeyOnce sync.Once
	grepCursorKey     []byte
)

func cursorMACKey() []byte {
	grepCursorKeyOnce.Do(func() {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			// crypto/rand 失败极罕见；退化为域哈希仍能防模型拼 token。
			sum := sha256.Sum256([]byte("gaea-grep-cursor-fallback"))
			key = sum[:]
		}
		grepCursorKey = key
	})
	return grepCursorKey
}

func cursorMAC(bind, snap, next string) string {
	m := hmac.New(sha256.New, cursorMACKey())
	m.Write([]byte(bind))
	m.Write([]byte{0})
	m.Write([]byte(snap))
	m.Write([]byte{0})
	m.Write([]byte(next))
	return hex.EncodeToString(m.Sum(nil))[:16]
}

// cursorBind 计算绑定哈希：查询身份（pattern/root/include）+ 规则版本。
func cursorBind(pattern, root, include string) string {
	h := sha256.New()
	h.Write([]byte(pattern))
	h.Write([]byte{0})
	h.Write([]byte(root))
	h.Write([]byte{0})
	h.Write([]byte(include))
	h.Write([]byte{0})
	h.Write([]byte(grepFenceRulesVer))
	h.Write([]byte{0})
	h.Write([]byte(grepSkipRulesVer))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// cursorSnapHash 对前缀候选文件身份 (rel,size,mtimeNano) 流式哈希。
type cursorSnap struct {
	h hash.Hash
}

func newCursorSnap() *cursorSnap { return &cursorSnap{h: sha256.New()} }

func (s *cursorSnap) feed(rel string, size int64, mtimeNano int64) {
	s.h.Write([]byte(rel))
	s.h.Write([]byte{0})
	_, _ = fmt.Fprintf(s.h, "%d\x00%d\x00", size, mtimeNano)
}

func (s *cursorSnap) sum() string { return hex.EncodeToString(s.h.Sum(nil))[:16] }

// emitCursor 组装下一页 token。snap=前缀快照（边界文件含）。
func emitCursor(bind, snap string, next int) string {
	ns := strconv.Itoa(next)
	return strings.Join([]string{
		grepCursorDomain, bind, snap, ns, cursorMAC(bind, snap, ns),
	}, ".")
}

// parseCursor 校验并拆 token。返回 bind/snap/next；格式或 mac 不符即错。
func parseCursor(token string) (bind, snap string, next int, err error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 5 || parts[0] != grepCursorDomain {
		return "", "", 0, fmt.Errorf("cursor 格式非法（应为 %s.<bind>.<snap>.<next>.<mac>）", grepCursorDomain)
	}
	bind, snap, ns, mac := parts[1], parts[2], parts[3], parts[4]
	if cursorMAC(bind, snap, ns) != mac {
		return "", "", 0, fmt.Errorf("cursor 校验失败（不属于本进程或被截断/篡改）")
	}
	next, err = strconv.Atoi(ns)
	if err != nil || next < 0 {
		return "", "", 0, fmt.Errorf("cursor next 偏移非法")
	}
	return bind, snap, next, nil
}

// feedCandidate 把一个候选文件喂进前缀快照（越界身份跳过，不中断）。
func (s *cursorSnap) feedCandidate(rel string, info os.FileInfo) {
	if info == nil {
		return
	}
	s.feed(rel, info.Size(), info.ModTime().UnixNano())
}
