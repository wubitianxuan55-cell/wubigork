// Package whisper — foreground_state.go
// 100% 对齐 ackem context/foregroundState.ts
// 前台窗口感知：会议/演示/专注场景分类 + 健康抑制策略

package whisper

import (
	"regexp"
	"strings"
	"time"
)

// 前台窗口分类正则
var meetingRE = regexp.MustCompile(`(?i)zoom|teams|腾讯会议|飞书|钉钉|meeting|会议|webex|slack\s*huddle|discord.*call`)
var presentationRE = regexp.MustCompile(`(?i)powerpoint|power point|ppt|幻灯片|presenter|keynote|wps\s*演示|fullscreen\s*slide`)
var focusRE = regexp.MustCompile(`(?i)专注助手|focus\s*assist|请勿打扰|do not disturb|勿扰模式`)

// globalForegroundSnapshot 全局前台快照（进程级单例）
var globalForegroundSnapshot = ForegroundSnapshot{
	Scene:     SceneOther,
	UpdatedAt: time.Now().UnixMilli(),
}

// ClassifyForegroundTitle 根据窗口标题分类场景
func ClassifyForegroundTitle(title string) (ForegroundScene, bool) {
	t := strings.TrimSpace(title)
	if t == "" {
		return SceneOther, false
	}
	if meetingRE.MatchString(t) {
		return SceneMeeting, true
	}
	if presentationRE.MatchString(t) {
		return ScenePresentation, true
	}
	if focusRE.MatchString(t) {
		return SceneFocus, true
	}
	return SceneOther, false
}

// SetForegroundPollingEnabled 设置前台轮询开关
func SetForegroundPollingEnabled(enabled bool) {
	if !enabled {
		globalForegroundSnapshot = ForegroundSnapshot{
			Enabled:   false,
			Scene:     SceneOther,
			UpdatedAt: time.Now().UnixMilli(),
		}
		return
	}
	globalForegroundSnapshot.Enabled = true
	globalForegroundSnapshot.UpdatedAt = time.Now().UnixMilli()
}

// UpdateForegroundTitle 更新前台窗口标题
func UpdateForegroundTitle(title string) ForegroundSnapshot {
	scene, suppress := ClassifyForegroundTitle(title)
	globalForegroundSnapshot = ForegroundSnapshot{
		Enabled:              globalForegroundSnapshot.Enabled,
		Title:                strings.TrimSpace(title),
		Scene:                scene,
		ShouldSuppressHealth: suppress,
		UpdatedAt:            time.Now().UnixMilli(),
	}
	return GetForegroundSnapshot()
}

// GetForegroundSnapshot 获取当前前台快照
func GetForegroundSnapshot() ForegroundSnapshot {
	s := globalForegroundSnapshot
	return s
}

// ShouldSuppressHealthForForeground 是否因前台场景应抑制健康提醒
func ShouldSuppressHealthForForeground() bool {
	s := globalForegroundSnapshot
	return s.Enabled && s.ShouldSuppressHealth
}
