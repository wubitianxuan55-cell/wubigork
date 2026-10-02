package app

// ── 角色剧照接入图像域登记（T1 · play/characterlib）──
//
// 角色在小说/角色库「设为剧照」后，其落盘立绘进入画室素材溯源：
// 来源=characterlib、参数带 character_id。剧照不在本地（data URL/远端）或文件
// 不存在时跳过；登记失败只 warn，不影响保存主流程。

import (
	"log/slog"
	"os"
	"strings"

	"github.com/gaea/gaea/internal/types"
)

// registerCharacterPortraitAsset 把角色最新剧照登记进图像域 ledger。
//
// backend/model（审计 AP7-05）：剧照的实际生成后端/模型由角色库绑定解析
// （portraitImageBinding：PortraitBackend/PortraitModel 空则回落全局绘梦），
// 由调用方传入。此前固定登记空 backend + 空 model，消耗报表按 {model,backend}
// 分组时角色剧照被拆成独立行（同后端被拆成多行）。拿到的是**绑定配置**值而非
// 该次生成的历史值：剧照可能是更早用别的配置生成的，台账只能记当前生效绑定——
// 这一点如实登记为余量（历史值未落盘，无法回溯）。
func registerCharacterPortraitAsset(cwd string, chars []types.Character, charID, backend, model string) {
	for _, ch := range chars {
		if ch.ID != charID {
			continue
		}
		p := strings.TrimSpace(ch.PortraitURL)
		if p == "" {
			return
		}
		if strings.HasPrefix(p, "data:") || strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
			return // 未落盘（远程/内联）无法登记文件资产
		}
		if _, err := os.Stat(p); err != nil {
			return
		}
		err := recordImageHubGeneratedAsset(cwd, "play", "characterlib", backend, model,
			"",
			map[string]interface{}{"character_id": charID},
			imageHubAsset{Kind: ImageHubAssetKindImage, Path: p}, nil, "")
		if err != nil {
			slog.Warn("角色剧照登记失败（不影响保存）", "character_id", charID, "path", p, "error", err)
		}
		return
	}
}
