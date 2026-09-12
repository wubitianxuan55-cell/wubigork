package app

// ── 原罪工具：插图（sin_illustrate）──
//
// 出图链不复制第二份：直接委托既有 SinIllustrate（角色锚点 + T2 参考槽门控 +
// 原罪自有产物目录 + 画室台账登记），工具只负责「把结果讲给模型听」并把产物
// 记进轨迹（前端过程卡渲染缩略图）。
//
// 与正文标记协议 @@插图|描述@@ 的分工（两者都保留，提示词里写明）：
//   - 标记协议：**写在正文里**的插图。正文先流式出完，前端按标记位置逐张生成
//     （有进度条、串行队列、取消），图片落在情节真正需要它的位置；
//   - 本工具：**这次要的是一张图**。用户点名「画一张」「把刚才那幕画出来」
//     「再画一张」「换个说法重画」时，模型直接调它，产物随过程卡一起出现。
//
// 取消口径：SinIllustrate 不接 ctx（出图是后端阻塞调用，ComfyUI/Herdsman 侧
// 也没有中断接口），所以「停止」在本轮工具执行完后立刻生效，中途不会留下半张
// 图；这一点在过程卡上用运行态如实表达，不假装能中断。

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

func init() {
	registerSinTool(sinToolIllustrate, func(c sinToolContext) sinTool {
		return &sinIllustrateTool{app: c.app, topicID: c.topicID}
	})
}

// sinIllustrateCaptionMaxRunes 过程卡标题的长度上限（caption 缺省取画面描述）。
const sinIllustrateCaptionMaxRunes = 60

// sinIllustrateTool 插图工具实例。注册表每轮新建实例，arts 只在本次执行内累积
// （单次 Execute 内 append，无并发写）。
type sinIllustrateTool struct {
	app     *App
	topicID string
	arts    []sinToolArtifact
}

func (*sinIllustrateTool) Name() string { return sinToolIllustrate }

func (*sinIllustrateTool) Description() string {
	return "为当前故事生成一张插图（本地图像模型，约十几秒到一分钟，用户能看见进度）。" +
		"只在用户明确要图时用：点名「画一张」「把刚才那幕画出来」「再画一张」；正文里顺手配的插图不走这个工具，" +
		"走正文标记协议（在正文里另起一行写 " + sinIllustrationCueOpen + "画面描述" + sinIllustrationCueClose + "），" +
		"两者不能对同一张图重复做。prompt 写给图像模型：人物外貌与服装、姿态与动作、镜头景别、光线与氛围，尽量具体可视；" +
		"要重画某一张时，把那一张的画面描述照抄进 prompt 再写清这次要改什么（例如「换成雨夜」「改成半身近景」）。" +
		"本故事已选角色的外观会自动锚定，不用在 prompt 里重复长相。"
}

func (*sinIllustrateTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "prompt":{"type":"string","description":"画面描述（写给图像模型，具体可视；角色外观由系统自动锚定）"},
  "caption":{"type":"string","description":"可选。这张图在过程卡/导出里的标题，短一句；缺省取画面描述前 60 字。"},
  "size":{"type":"string","description":"可选。画面尺寸，如 1024x1024 / 832x1216；缺省按图像后端默认。"}
},
"required":["prompt"]}`)
}

// ReadOnly=false：真的出图并落盘（还会登记画室台账），前端按写类工具标注。
func (*sinIllustrateTool) ReadOnly() bool { return false }

// Artifacts 本次执行产生的图片产物（sinToolArtifactProvider；Execute 返回后取走）。
func (t *sinIllustrateTool) Artifacts() []sinToolArtifact { return t.arts }

func (t *sinIllustrateTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Prompt  string `json:"prompt"`
		Caption string `json:"caption"`
		Size    string `json:"size"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("参数解析失败: %w", err)
		}
	}
	prompt := strings.TrimSpace(p.Prompt)
	if prompt == "" {
		return "", fmt.Errorf("缺少 prompt（画面描述不能为空）")
	}
	if t.app == nil {
		return "", fmt.Errorf("插图工具未接入应用（app 为空）")
	}

	// messageID<=0 / cue 为空：只出图不落库——本轮助手消息 id 要等落库后才有，
	// 回写由工具循环在落库后按轨迹里的产物补齐（见 sin_tool_loop.go）。
	res, err := t.app.SinIllustrate(t.topicID, 0, "", prompt, strings.TrimSpace(p.Size))
	if err != nil {
		return "", err
	}
	path, _ := res["path"].(string)
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("插图未落盘（图像后端没有返回可保存的图片）")
	}
	used, _ := res["prompt_used"].(string)
	if strings.TrimSpace(used) == "" {
		used = prompt
	}
	caption := strings.TrimSpace(p.Caption)
	if caption == "" {
		caption = truncateRunes(used, sinIllustrateCaptionMaxRunes)
	}
	t.arts = append(t.arts, sinToolArtifact{Kind: sinToolArtifactKindImage, Path: path, Caption: caption})

	var b strings.Builder
	fmt.Fprintf(&b, "已生成插图并保存到：%s\n", path)
	fmt.Fprintf(&b, "画面：%s\n", used)
	if names := sinRefNamesText(res); names != "" {
		fmt.Fprintf(&b, "角色参考：%s\n", names)
	}
	if reason, _ := res["ref_reason"].(string); strings.TrimSpace(reason) != "" {
		fmt.Fprintf(&b, "参考槽说明：%s\n", strings.TrimSpace(reason))
	}
	b.WriteString("这张图会以缩略图出现在用户的消息里，不必再为它写插图标记；接着写正文即可。")
	return b.String(), nil
}

// sinRefNamesText 参考槽命中的角色名（无则空串）。
func sinRefNamesText(res map[string]interface{}) string {
	names, _ := res["ref_characters"].([]string)
	if len(names) == 0 {
		return ""
	}
	return strings.Join(names, "、")
}

// ── 产物类型（sin_tool_illustrate.go 单点定义，导出/前端契约共用）──

// sinToolArtifactKindImage 图片产物（当前唯一取值）。
const sinToolArtifactKindImage = "image"

// sinToolArtifact 工具产物：路径交给过程卡渲染缩略图，不塞进喂回模型的文本。
type sinToolArtifact struct {
	Kind    string `json:"kind"`              // "image"
	Path    string `json:"path"`              // 本地绝对路径
	Caption string `json:"caption,omitempty"` // 画面描述（过程卡标题 / 导出替代文本）
}

// sinToolArtifactProvider 可选能力：工具在 Execute 期间记录产物。注册表每轮新建
// 实例，Execute 返回后立刻取走，不需要跨轮共享状态。
type sinToolArtifactProvider interface {
	Artifacts() []sinToolArtifact
}

// sinPersistToolArtifacts 落库后回写：把本轮轨迹里的图片产物按 tool0..toolN
// 并入助手消息 extra.illustrations（画廊/导出与正文标记图同一存储，cue 前缀
// toolN 避免与正文标记的数字 cue 互踩）。失败逐条告警不阻断（产物已在画室
// 台账与磁盘上，轨迹里也有路径可寻回）。
func (a *App) sinPersistToolArtifacts(messageID int64, trace []sinToolTrace) {
	if messageID <= 0 {
		return
	}
	idx := 0
	for _, tr := range trace {
		for _, art := range tr.Artifacts {
			if art.Kind != sinToolArtifactKindImage || strings.TrimSpace(art.Path) == "" {
				continue
			}
			cue := fmt.Sprintf("tool%d", idx)
			idx++
			if err := a.sinAttachIllustration(messageID, cue, art.Path); err != nil {
				slog.Warn("原罪工具插图回写失败", "messageID", messageID, "cue", cue, "path", art.Path, "error", err)
			}
		}
	}
}
