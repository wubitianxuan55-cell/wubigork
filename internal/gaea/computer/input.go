package computer

// W5 动作编排（平台无关；施工单 W5）：闸门（双闸门+DPI 禁用+deny 表）→
// 平台注入原语。工具层只调这几个入口，注入零散逻辑不外漏。
// 纪律对齐 asset-62 §5：element_index/坐标只属于产生它的那次观察——
// 元素动作过 CheckElementIndex；坐标动作过 CheckCoordinates；type/key
// 只需目标窗口活着（codex 同款：type_text 无索引形态）。

import "errors"

// 动作 DPI 失败哨兵：工具层透出时提示坐标动作已整体禁用。
var ErrDpiDisabledSentinel = errors.New("computer: dpi")

// resolveWindow 校验窗口活着并回完整 Window（枚举里找；授权/deny 表用）。
func resolveWindow(windowID int) (Window, error) {
	wins, err := ListWindows()
	if err != nil {
		return Window{}, err
	}
	for _, w := range wins {
		if w.ID == windowID {
			return w, nil
		}
	}
	return Window{}, ErrWindowGone
}

// ClickWindow 窗口相对坐标点击：坐标闸门 → deny 表 → 注入。
func (s *Session) ClickWindow(x, y int, button string, count int) error {
	if s.CoordinatesDisabled() {
		return ErrCoordinatesDisabled
	}
	if err := s.CheckCoordinates(); err != nil {
		return err
	}
	s.mu.Lock()
	snap := s.lastState
	s.mu.Unlock()
	if snap == nil {
		return ErrStaleObservation
	}
	if err := GuardWindow(snap.Window); err != nil {
		return err
	}
	sx, sy := WindowRelativeToScreen(snap.Window, x, y)
	if err := ClickAtScreen(button, count, sx, sy); err != nil {
		if errors.Is(err, errDpiRestoreFailed) {
			s.SetCoordinatesDisabled(true)
		}
		return err
	}
	return nil
}

// ClickWindowElement 元素点击：元素闸门 → deny 表 → Invoke/坐标回落注入。
func (s *Session) ClickWindowElement(index int) error {
	if err := s.CheckElementIndex(index); err != nil {
		return err
	}
	s.mu.Lock()
	snap := s.lastState
	s.mu.Unlock()
	if snap == nil {
		return ErrStaleObservation
	}
	if err := GuardWindow(snap.Window); err != nil {
		return err
	}
	el, ok := snap.ElementAt(index)
	if !ok {
		return ErrStaleObservation
	}
	if err := clickElement(el); err != nil {
		if errors.Is(err, errDpiRestoreFailed) {
			s.SetCoordinatesDisabled(true)
		}
		return err
	}
	return nil
}

// TypeInto 聚焦目标窗口并注入 Unicode 文本（deny 表先过）。
func (s *Session) TypeInto(text string) error {
	s.mu.Lock()
	snap := s.lastState
	s.mu.Unlock()
	if snap == nil {
		return ErrStaleObservation
	}
	if err := GuardWindow(snap.Window); err != nil {
		return err
	}
	FocusWindow(snap.Window.ID)
	return TypeTextRaw(text)
}

// PressKeyOn 聚焦目标窗口并注入组合键（deny 表双保险：键名 + 目标窗口）。
func (s *Session) PressKeyOn(chord string) error {
	if err := GuardChord(chord); err != nil {
		return err
	}
	s.mu.Lock()
	snap := s.lastState
	s.mu.Unlock()
	if snap == nil {
		return ErrStaleObservation
	}
	if err := GuardWindow(snap.Window); err != nil {
		return err
	}
	FocusWindow(snap.Window.ID)
	return PressKeyChord(chord)
}

// ScrollWindow 窗口相对坐标滚轮（坐标闸门同 Click）。
func (s *Session) ScrollWindow(x, y int, dx, dy int) error {
	if s.CoordinatesDisabled() {
		return ErrCoordinatesDisabled
	}
	if err := s.CheckCoordinates(); err != nil {
		return err
	}
	s.mu.Lock()
	snap := s.lastState
	s.mu.Unlock()
	if snap == nil {
		return ErrStaleObservation
	}
	if err := GuardWindow(snap.Window); err != nil {
		return err
	}
	sx, sy := WindowRelativeToScreen(snap.Window, x, y)
	if err := ScrollAtScreen(sx, sy, dx, dy); err != nil {
		if errors.Is(err, errDpiRestoreFailed) {
			s.SetCoordinatesDisabled(true)
		}
		return err
	}
	return nil
}
