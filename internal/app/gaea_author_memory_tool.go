package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/authorprefs"
	"github.com/gaea/gaea/internal/gaea/spaces"
)

type authorMemoryTool struct{ a *App }

func (authorMemoryTool) Name() string { return "author_memory" }

func (authorMemoryTool) Description() string {
	return "管理作者显式偏好：remember/decide/replace/forget/query。只接受用户明确表达、已接受的建议或手工录入；不得从正文自动推断偏好。本书偏好覆盖全局偏好；未确认冲突不得生效。"
}

func (authorMemoryTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "action":{"type":"string","enum":["remember","decide","replace","forget","query"]},
  "event_id":{"type":"string","description":"幂等事件 ID；同一事件重放不重复写入"},
  "target_id":{"type":"string","description":"decide/replace/forget 的目标偏好 ID"},
  "accept":{"type":"boolean","description":"decide：是否接受待确认/冲突偏好"},
  "text":{"type":"string","description":"replace：新文本"},
  "kind":{"type":"string","description":"偏好类别，如 voice/pacing/character/world/avoid/workflow"},
  "scope":{"type":"string","enum":["global","genre","workflow","book"]},
  "key":{"type":"string","description":"可选的稳定键；同键不同文本会进入 conflict"},
  "preference_text":{"type":"string","description":"remember：偏好正文"},
  "evidence":{"type":"array","items":{"type":"string"},"description":"remember：用户原话或来源"},
  "importance":{"type":"integer","minimum":1,"maximum":100}
},
"required":["action"]
}`)
}

func (authorMemoryTool) ReadOnly() bool { return false }

func (authorMemoryTool) PersistWrite() bool { return true }

func (authorMemoryTool) SpaceTag() string { return spaces.SpacePlay }

func (t authorMemoryTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	if t.a == nil {
		return "", fmt.Errorf("author_memory: app unavailable")
	}
	var req struct {
		Action         string   `json:"action"`
		EventID        string   `json:"event_id"`
		TargetID       string   `json:"target_id"`
		Accept         bool     `json:"accept"`
		Text           string   `json:"text"`
		Kind           string   `json:"kind"`
		Scope          string   `json:"scope"`
		Key            string   `json:"key"`
		PreferenceText string   `json:"preference_text"`
		Evidence       []string `json:"evidence"`
		Importance     int      `json:"importance"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return "", fmt.Errorf("author_memory: invalid args: %w", err)
	}
	req.Action = strings.TrimSpace(req.Action)
	var bookRoot string
	if pm := t.a.getPM(); pm != nil {
		bookRoot = pm.Dir
	}
	mem, err := authorprefs.Open(gaeaCwd(), bookRoot)
	if err != nil {
		return "", err
	}
	if req.Action == "query" {
		q, err := mem.Query(strings.TrimSpace(req.Kind), strings.TrimSpace(req.Scope))
		if err != nil {
			return "", err
		}
		return authorQueryJSON(q)
	}
	if req.EventID == "" {
		req.EventID = fmt.Sprintf("tool-%d", time.Now().UnixNano())
	}
	ev := authorprefs.Event{
		ID: req.EventID, Action: req.Action, TargetID: req.TargetID,
		Accept: req.Accept, Text: req.Text,
		Preference: authorprefs.Preference{
			Key: req.Key, Kind: req.Kind, Scope: req.Scope, Text: req.PreferenceText,
			Evidence: req.Evidence, Source: authorprefs.SourceManual,
			Importance: req.Importance,
		},
	}
	p, changed, err := mem.Apply(ev)
	if err != nil {
		return "", err
	}
	out, _ := json.Marshal(map[string]any{"ok": true, "changed": changed, "preference": p})
	return string(out), nil
}

func authorQueryJSON(q authorprefs.QueryResult) (string, error) {
	out, err := json.Marshal(q)
	return string(out), err
}

func (a *writingState) authorMemorySection(pmPath string) string {
	var bookRoot string
	if pmPath != "" {
		bookRoot = pmPath
	}
	mem, err := authorprefs.Open(gaeaCwd(), bookRoot)
	if err != nil {
		return ""
	}
	q, err := mem.Query("", "")
	if err != nil {
		return ""
	}
	return authorprefs.RenderQuery(q)
}
