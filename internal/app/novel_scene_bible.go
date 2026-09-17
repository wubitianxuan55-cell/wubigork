package app

// 刀7续收官·POV 视图（规格 进度计划/gaea-pov-view-20260917.md）：场景圣经
// 编译器（novelcontext 刀2）的 POV 视角掩码此前只在生成链内部消费——本绑定
// 把编译结果透出给前端「视角」抽屉：povView（POV 已知）/hiddenFacts（不知情
// 且不得泄露）对照为核心区。双路编译：sceneID 非空=该场景；空=整章合成
// （BuildSceneBibleFromChapter，v3/纯 blob 章也可用）。视图 camelCase typed
// （新绑定纪律）；编译内部读取全部静默降级（novelcontext 契约），空区段如实
// 空展示，不报错不编造。

import (
	"fmt"

	"github.com/gaea/gaea/internal/novelcontext"
	"github.com/gaea/gaea/internal/types"
)

// SceneBibleCharView 出场角色浓缩行（novelcontext.SceneChar 的 wire 投影）。
type SceneBibleCharView struct {
	Name         string   `json:"name"`
	RoleType     string   `json:"roleType,omitempty"`
	Status       string   `json:"status,omitempty"`
	Location     string   `json:"location,omitempty"`
	CurrentState string   `json:"currentState,omitempty"` // t5 状态机回灌（心理/处境）
	CareerMain   string   `json:"careerMain,omitempty"`
	CareerSub    []string `json:"careerSub,omitempty"`
	Items        []string `json:"items,omitempty"`
	KnownBy      []string `json:"knownBy,omitempty"`
}

// SceneBibleView 场景圣经 wire 视图（核心区=povView/hiddenFacts 对照）。
type SceneBibleView struct {
	Pov         string               `json:"pov,omitempty"` // POV 角色 ID（场景未设 POV 为空）
	Title       string               `json:"title,omitempty"`
	Location    string               `json:"location,omitempty"`
	TimeOfDay   string               `json:"timeOfDay,omitempty"`
	Mood        string               `json:"mood,omitempty"` // 情感基调
	Tags        []string             `json:"tags,omitempty"`
	Characters  []SceneBibleCharView `json:"characters"`
	PovView     string               `json:"povView,omitempty"` // POV 已知的关键事实
	HiddenFacts []string             `json:"hiddenFacts"`       // POV 不知情且不得泄露
	Foreshadows []string             `json:"foreshadows"`       // 未回收伏笔（创作约束）
	Memories    []string             `json:"memories"`          // 相关记忆（t3-P2 语义召回）
	TimeAnchor  string               `json:"timeAnchor,omitempty"`
	Style       string               `json:"style,omitempty"`
	Thread      string               `json:"thread,omitempty"` // 故事主线
}

// NovelSceneBibleView 场景圣经视图（刀7续 POV 视图）：sceneID 非空=该场景
// 编译（不存在报错）；空=整章合成。恒非 nil 切片字段保证前端零判空。
func (a *writingState) NovelSceneBibleView(chapterNum int, sceneID string) (SceneBibleView, error) {
	pm := a.getPM()
	if pm == nil {
		return SceneBibleView{}, fmt.Errorf("请先打开项目")
	}

	var bible *novelcontext.SceneBible
	if sceneID == "" {
		b, err := novelcontext.BuildSceneBibleFromChapter(pm, chapterNum)
		if err != nil {
			return SceneBibleView{}, fmt.Errorf("编译场景圣经失败: %w", err)
		}
		bible = b
	} else {
		sm := pm.SceneManager(chapterNum)
		scene, err := sm.Read(sceneID)
		if err != nil || scene == nil {
			return SceneBibleView{}, fmt.Errorf("场景不存在: %s", sceneID)
		}
		b, err := novelcontext.CompileSceneBible(pm, chapterNum, scene)
		if err != nil {
			return SceneBibleView{}, fmt.Errorf("编译场景圣经失败: %w", err)
		}
		bible = b
	}

	return sceneBibleViewOf(bible), nil
}

// sceneBibleViewOf SceneBible → wire 视图（切片恒非 nil；空字段如实空）。
func sceneBibleViewOf(b *novelcontext.SceneBible) SceneBibleView {
	v := SceneBibleView{
		Pov:         b.Scene.POVCharID,
		Title:       b.Scene.Title,
		Location:    b.Scene.Location,
		TimeOfDay:   b.Scene.TimeOfDay,
		Mood:        b.Scene.Emotion,
		Tags:        b.Scene.Tags,
		Characters:  []SceneBibleCharView{},
		HiddenFacts: []string{},
		Foreshadows: []string{},
		Memories:    []string{},
		PovView:     b.POVView,
		TimeAnchor:  b.TimeAnchor,
		Style:       b.Style,
		Thread:      b.Thread,
	}
	for _, c := range b.Characters {
		v.Characters = append(v.Characters, SceneBibleCharView{
			Name: c.Name, RoleType: c.RoleType, Status: c.Status, Location: c.Location,
			CurrentState: c.CurrentState, CareerMain: c.CareerMain,
			CareerSub: c.CareerSub, Items: c.Items, KnownBy: c.KnownBy,
		})
	}
	v.HiddenFacts = append(v.HiddenFacts, b.HiddenFacts...)
	v.Foreshadows = append(v.Foreshadows, b.Foreshadows...)
	v.Memories = append(v.Memories, b.Memories...)
	if v.Tags == nil {
		v.Tags = []string{}
	}
	return v
}

// 编译期钉住 SceneChar 字段对齐（novelcontext 改名时此行编译失败提醒同步）。
var _ = types.SceneMeta{}
