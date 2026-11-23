package app

// wenxue_privacy.go — 图文板块私密保护（2026-11-23 用户拍板四形态）。
//
// 形态与口径：
//   1. 失焦自动遮罩 —— 前端 blur/visibilitychange 面（本文件只供开关与档位）；
//   2. 无限制书 PIN 锁 —— PIN 全局一枚护所有 深度成人（无限制） 书：salted
//      SHA-256 存 config（明文不落盘），会话解锁态在内存（重启回锁），
//      连续 5 次失败退避 30s；
//   3. 搜索与日志隔离 —— 统一搜索 drop tuwen 命中（gaea_unified_search.go），
//      日志经 wenxueRedactPath 脱敏；
//   4. 书籍隐藏 —— 隐藏清单（书名 JSON 数组）驱动 书单过滤 + 目录
//      FILE_ATTRIBUTE_HIDDEN（资源管理器默认不显）。
//
// 事实源与边界：
//   - 档位事实源 = 书籍信息.md 档位行（wenxueBookAdultLevelAt）；
//   - 未设 PIN 时隐藏清单不参与书单过滤（避免「藏了却无解锁路径」死局），
//     目录隐藏属性不受此限（资源管理器面独立生效）；
//   - PIN 值只进哈希不进日志；Overview 在锁定态对隐藏书做名字/路径置空
//     （列表不外泄书名）。

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/config"
)

// wenxuePrivacyState 会话态（零值可用——测试裸 &App{} 无需初始化）。
type wenxuePrivacyState struct {
	mu        sync.Mutex
	unlocked  bool      // 会话解锁态（重启回锁）
	pinFails  int       // 连续失败计数
	lockoutAt time.Time // 退避截止时刻（>now 时拒绝尝试）
}

// wenxuePinMaxFails / wenxuePinLockout 退避参数（连续失败阈值与时长）。
const (
	wenxuePinMaxFails = 5
	wenxuePinLockout  = 30 * time.Second
)

// wenxuePinLockedByConfig PIN 是否已设置（config 键非空）。
func wenxuePinLockedByConfig() bool {
	if cfg := config.Load(); cfg != nil {
		return strings.TrimSpace(cfg.WenxuePrivacyPinHash) != ""
	}
	return false
}

// wenxuePrivacyIsLocked 锁定态判定：PIN 已设置且本会话未解锁。
func (a *App) wenxuePrivacyIsLocked() bool {
	if !wenxuePinLockedByConfig() {
		return false
	}
	a.wenxuePrivacyState.mu.Lock()
	defer a.wenxuePrivacyState.mu.Unlock()
	return !a.wenxuePrivacyState.unlocked
}

// wenxuePrivacyHashPin 生成 "salt:sha256hex"（随机 16B salt）。
func wenxuePrivacyHashPin(pin string) string {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		// crypto/rand 失败极罕见（系统熵源故障）；诚实报错不降级明文。
		slog.Error("PIN salt 生成失败", "error", err)
		return ""
	}
	sum := sha256.Sum256(append(salt, []byte(pin)...))
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(sum[:])
}

// wenxuePrivacyCheckPin 常量时间比对 stored("salt:hash") 与 pin。
func wenxuePrivacyCheckPin(stored, pin string) bool {
	saltHex, hashHex, ok := strings.Cut(stored, ":")
	if !ok {
		return false
	}
	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return false
	}
	want, err := hex.DecodeString(hashHex)
	if err != nil {
		return false
	}
	sum := sha256.Sum256(append(salt, []byte(pin)...))
	return subtle.ConstantTimeCompare(sum[:], want) == 1
}

// wenxuePrivacyHiddenBooks 读隐藏书清单（书名数组；坏 JSON 视为空并告警）。
func wenxuePrivacyHiddenBooks() []string {
	if cfg := config.Load(); cfg != nil && strings.TrimSpace(cfg.WenxuePrivacyHiddenBooks) != "" {
		var names []string
		if err := json.Unmarshal([]byte(cfg.WenxuePrivacyHiddenBooks), &names); err != nil {
			slog.Warn("图文隐藏书清单解析失败（视为空）", "error", err)
			return nil
		}
		return names
	}
	return nil
}

// wenxuePrivacySaveHiddenBooks 写隐藏书清单。
func wenxuePrivacySaveHiddenBooks(names []string) error {
	if names == nil {
		names = []string{}
	}
	raw, err := json.Marshal(names)
	if err != nil {
		return err
	}
	return config.Save(config.KeyWenxuePrivacyHiddenBooks, string(raw))
}

// wenxuePrivacyBookHidden 名字是否在隐藏清单。
func wenxuePrivacyBookHidden(name string) bool {
	for _, n := range wenxuePrivacyHiddenBooks() {
		if strings.EqualFold(n, name) {
			return true
		}
	}
	return false
}

// wenxueSetDirHidden 目录系统隐藏属性加/摘（Windows；其他平台 no-op）。
// 失败告警不阻断（属性面是纵深，不是锁本体）。
func wenxueSetDirHidden(dir string, hidden bool) {
	attrs, err := windowsGetFileAttributes(dir)
	if err != nil {
		slog.Warn("图文目录属性读取失败", "dir", wenxueRedactPath(dir), "error", err)
		return
	}
	if hidden {
		attrs |= fileAttributeHidden
	} else {
		attrs &^= fileAttributeHidden
	}
	if err := windowsSetFileAttributes(dir, attrs); err != nil {
		slog.Warn("图文目录隐藏属性写入失败", "dir", wenxueRedactPath(dir), "error", err)
	}
}

// ── Overview 与六动作（前端绑定面） ────────────────────────────────

// WenxuePrivacyBook 单书隐私视图。锁定态下隐藏书 Name/Path 置空（列表不外泄书名）。
type WenxuePrivacyBook struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	AdultLevel string `json:"adultLevel"`
	Hidden     bool   `json:"hidden"`
	Current    bool   `json:"current"`
}

// WenxuePrivacyOverview 私密保护总览（面板与遮罩/解锁流的数据源）。
type WenxuePrivacyOverview struct {
	PinSet      bool                `json:"pinSet"`
	Unlocked    bool                `json:"unlocked"`
	MaskOnBlur  bool                `json:"maskOnBlur"`
	HiddenCount int                 `json:"hiddenCount"`
	Books       []WenxuePrivacyBook `json:"books"`
}

// WenxuePrivacyOverview 返回当前私密保护总览。锁定态下隐藏书名字/路径置空
// （只留 hidden=true 供占位计数），解锁态或未设 PIN 时全量明文。
func (a *App) WenxuePrivacyOverview() WenxuePrivacyOverview {
	pinSet := wenxuePinLockedByConfig()
	locked := a.wenxuePrivacyIsLocked()
	maskOnBlur := true
	if cfg := config.Load(); cfg != nil {
		maskOnBlur = cfg.WenxuePrivacyMaskOnBlur
	}
	hidden := wenxuePrivacyHiddenBooks()
	current := wenxueWorkspace()
	out := WenxuePrivacyOverview{
		PinSet:      pinSet,
		Unlocked:    !locked,
		MaskOnBlur:  maskOnBlur,
		HiddenCount: len(hidden),
		Books:       []WenxuePrivacyBook{},
	}
	root := wenxueWorkspaceRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		slog.Warn("图文私密总览书根读取失败", "error", err)
		return out
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		abs := filepath.Join(root, e.Name())
		book := WenxuePrivacyBook{
			Name:       e.Name(),
			Path:       abs,
			AdultLevel: wenxueBookAdultLevelAt(abs),
			Hidden:     wenxuePrivacyBookHidden(e.Name()),
			Current:    wenxueSamePath(abs, current),
		}
		if locked && book.Hidden {
			book.Name = ""
			book.Path = ""
		}
		out.Books = append(out.Books, book)
	}
	sort.SliceStable(out.Books, func(i, j int) bool {
		if out.Books[i].Current != out.Books[j].Current {
			return out.Books[i].Current
		}
		return strings.ToLower(out.Books[i].Name) < strings.ToLower(out.Books[j].Name)
	})
	return out
}

// wenxueValidatePin PIN 白名单：4~12 位数字（本机防肩窥口径，非加密强度件）。
func wenxueValidatePin(pin string) error {
	pin = strings.TrimSpace(pin)
	if len(pin) < 4 || len(pin) > 12 {
		return errors.New("PIN 须为 4~12 位数字")
	}
	for _, r := range pin {
		if r < '0' || r > '9' {
			return errors.New("PIN 须为 4~12 位数字")
		}
	}
	return nil
}

// WenxuePrivacySetPin 设置/修改 PIN。已设 PIN 时 oldPin 必须正确；
// 设置成功即进入锁定态（重启与本动作都会回锁，显式语义=改完即锁）。
func (a *App) WenxuePrivacySetPin(pin, oldPin string) error {
	pin = strings.TrimSpace(pin)
	if err := wenxueValidatePin(pin); err != nil {
		return err
	}
	if stored := func() string {
		if cfg := config.Load(); cfg != nil {
			return strings.TrimSpace(cfg.WenxuePrivacyPinHash)
		}
		return ""
	}(); stored != "" {
		if !wenxuePrivacyCheckPin(stored, strings.TrimSpace(oldPin)) {
			return errors.New("原 PIN 不正确")
		}
	}
	hash := wenxuePrivacyHashPin(pin)
	if hash == "" {
		return errors.New("PIN 哈希生成失败")
	}
	if err := config.Save(config.KeyWenxuePrivacyPinHash, hash); err != nil {
		return fmt.Errorf("PIN 保存失败: %w", err)
	}
	a.wenxuePrivacyState.mu.Lock()
	a.wenxuePrivacyState.unlocked = false
	a.wenxuePrivacyState.pinFails = 0
	a.wenxuePrivacyState.lockoutAt = time.Time{}
	a.wenxuePrivacyState.mu.Unlock()
	slog.Info("图文私密 PIN 已更新", "pin_set", true)
	return nil
}

// WenxuePrivacyUnlock 会话解锁：PIN 校验通过后本会话内解锁（重启回锁）。
// 连续失败达阈值进入退避。返回 ok=false 时 err 带可读原因。
func (a *App) WenxuePrivacyUnlock(pin string) (bool, error) {
	stored := ""
	if cfg := config.Load(); cfg != nil {
		stored = strings.TrimSpace(cfg.WenxuePrivacyPinHash)
	}
	if stored == "" {
		return false, errors.New("尚未设置 PIN")
	}
	a.wenxuePrivacyState.mu.Lock()
	defer a.wenxuePrivacyState.mu.Unlock()
	if remain := time.Until(a.wenxuePrivacyState.lockoutAt); remain > 0 {
		return false, fmt.Errorf("尝试次数过多，请 %d 秒后再试", int(remain.Seconds())+1)
	}
	if !wenxuePrivacyCheckPin(stored, strings.TrimSpace(pin)) {
		a.wenxuePrivacyState.pinFails++
		if a.wenxuePrivacyState.pinFails >= wenxuePinMaxFails {
			a.wenxuePrivacyState.lockoutAt = time.Now().Add(wenxuePinLockout)
			a.wenxuePrivacyState.pinFails = 0
			return false, errors.New("尝试次数过多，已临时锁定 30 秒")
		}
		return false, errors.New("PIN 不正确")
	}
	a.wenxuePrivacyState.unlocked = true
	a.wenxuePrivacyState.pinFails = 0
	a.wenxuePrivacyState.lockoutAt = time.Time{}
	slog.Info("图文私密会话已解锁")
	return true, nil
}

// WenxuePrivacyLock 手动回锁（即时生效；重启亦回锁）。
func (a *App) WenxuePrivacyLock() {
	a.wenxuePrivacyState.mu.Lock()
	a.wenxuePrivacyState.unlocked = false
	a.wenxuePrivacyState.mu.Unlock()
	slog.Info("图文私密会话已回锁")
}

// WenxuePrivacySetMaskOnBlur 失焦自动遮罩总开关。
func (a *App) WenxuePrivacySetMaskOnBlur(on bool) error {
	v := "0"
	if on {
		v = "1"
	}
	if err := config.Save(config.KeyWenxuePrivacyMaskOnBlur, v); err != nil {
		return fmt.Errorf("遮罩开关保存失败: %w", err)
	}
	return nil
}

// WenxuePrivacySetBookHidden 单书隐藏开关：更新清单并同步目录隐藏属性。
// name 必须是 tuwen 根的直接子目录名（防穿越）。
func (a *App) WenxuePrivacySetBookHidden(name string, hidden bool) error {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, `/\`) || name != filepath.Base(name) {
		return errors.New("书名不合法")
	}
	abs := filepath.Join(wenxueWorkspaceRoot(), name)
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		return errors.New("书籍不存在：" + name)
	}
	names := wenxuePrivacyHiddenBooks()
	out := names[:0:0]
	for _, n := range names {
		if strings.EqualFold(n, name) {
			continue
		}
		out = append(out, n)
	}
	if hidden {
		out = append(out, name)
	}
	if err := wenxuePrivacySaveHiddenBooks(out); err != nil {
		return err
	}
	wenxueSetDirHidden(abs, hidden)
	slog.Info("图文书籍隐藏开关已更新", "book", wenxueRedactPath(name), "hidden", hidden)
	return nil
}

// ── 过滤与脱敏（列表/搜索/日志三面共用） ──────────────────────────

// wenxueFilterHiddenWorkspaces 书单过滤（2026-11-23 修正语义）：隐藏清单内的
// 书恒滤——**不依赖锁定态**（用户点眼睛即生效；此前只在锁定态滤，未设 PIN 时
// 眼睛看似无效）。恢复路径=隐私区眼睛开关（双向，死局不存在）。
// 例外：现书保留（FileTree/引擎锚/UI current 定位依赖它；现书的侧栏分组
// 由 WenxueListProjectSessions 恒滤，浏览面仍不可见）。
func (a *App) wenxueFilterHiddenWorkspaces(views []WorkspaceView) []WorkspaceView {
	hidden := wenxuePrivacyHiddenBooks()
	if len(hidden) == 0 {
		return views
	}
	current := wenxueWorkspace()
	out := views[:0:0]
	for _, v := range views {
		if wenxueSamePath(v.Path, current) {
			out = append(out, v) // 现书恒保留
			continue
		}
		skip := false
		for _, n := range hidden {
			if strings.EqualFold(v.Name, n) {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, v)
		}
	}
	return out
}

// wenxueRedactPath 日志脱敏：路径（或裸书名）落在 无限制档/隐藏清单 书内时
// 一律返回「私密书籍」——余下相对路径也不留（章节文件名可能含章节标题，
// 一样泄密；宁紧勿松）。否则原样返回。空串原样。
func wenxueRedactPath(p string) string {
	if strings.TrimSpace(p) == "" {
		return p
	}
	bookDir := ""
	if filepath.IsAbs(p) {
		// 形态一：绝对路径在 tuwen 根下 → 首段=书名。
		if rel, err := filepath.Rel(wenxueWorkspaceRoot(), filepath.Clean(p)); err == nil &&
			rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "." {
			bookDir, _, _ = strings.Cut(filepath.ToSlash(rel), "/")
		}
	} else if !strings.ContainsAny(p, `/\`) {
		// 形态二：裸书名（隐藏清单内即脱敏）。
		if wenxuePrivacyBookHidden(filepath.Clean(p)) {
			bookDir = filepath.Clean(p)
		}
	}
	if bookDir == "" {
		return p
	}
	sensitive := wenxuePrivacyBookHidden(bookDir)
	if !sensitive {
		if lv := wenxueBookAdultLevelAt(filepath.Join(wenxueWorkspaceRoot(), bookDir)); lv == wenxueAdultUnrestricted {
			sensitive = true
		}
	}
	if !sensitive {
		return p
	}
	return "私密书籍"
}

// wenxueAutoHideUnrestricted 新建无限制档书时自动入隐藏清单 + 目录加隐藏属性
// （建书点调用；失败告警不阻断建书）。
func (a *App) wenxueAutoHideUnrestricted(name string) {
	names := wenxuePrivacyHiddenBooks()
	for _, n := range names {
		if strings.EqualFold(n, name) {
			return // 已在清单
		}
	}
	if err := wenxuePrivacySaveHiddenBooks(append(names, name)); err != nil {
		slog.Warn("无限制书自动隐藏失败（不阻断建书）", "error", err)
		return
	}
	wenxueSetDirHidden(filepath.Join(wenxueWorkspaceRoot(), name), true)
	slog.Info("无限制档新书已自动隐藏", "book", wenxueRedactPath(name))
}

// ── 统一搜索防御纵深（2026-11-23 私密保护） ────────────────────────
// keyword/files 两组本就只锚 gaeaCwd() 办公工作区（tuwen 书稿不进索引），
// 这里是 belt-and-braces：万一索引面日后扩到 tuwen，书稿命中也统一在这里
// 摘掉——跨板块搜索永远不吐 tuwen 路径与片段。

// wenxueDropTuwenKeywordHits keyword 组 tuwen 命中摘除。
func (a *App) wenxueDropTuwenKeywordHits(hits []WorkspaceSearchHit) []WorkspaceSearchHit {
	if len(hits) == 0 {
		return hits
	}
	out := hits[:0:0]
	for _, h := range hits {
		if wenxuePathInTuwenRoot(h.Path) {
			continue
		}
		out = append(out, h)
	}
	return out
}

// wenxueDropTuwenFileHits files 组 tuwen 命中摘除。
func wenxueDropTuwenFileHits(hits []FileSemanticHit) []FileSemanticHit {
	if len(hits) == 0 {
		return hits
	}
	out := hits[:0:0]
	for _, h := range hits {
		if wenxuePathInTuwenRoot(h.Path) {
			continue
		}
		out = append(out, h)
	}
	return out
}

// wenxuePathInTuwenRoot 路径（绝对或相对）是否落在 tuwen 书根内。
func wenxuePathInTuwenRoot(p string) bool {
	if strings.TrimSpace(p) == "" {
		return false
	}
	if filepath.IsAbs(p) {
		rel, err := filepath.Rel(wenxueWorkspaceRoot(), filepath.Clean(p))
		if err != nil {
			return false
		}
		return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
	}
	// 相对形态（WorkspaceSearchHit.Path 是工作区相对路径）：工作区非 tuwen 根
	// 时相对路径不可能指进 tuwen；工作区恰为 tuwen 根的形态由绝对分支兜住。
	return false
}
