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
func registerCharacterPortraitAsset(cwd string, chars []types.Character, charID string) {
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
		err := recordImageHubGeneratedAsset(cwd, "play", "characterlib", "", "", "",
			map[string]interface{}{"character_id": charID},
			imageHubAsset{Kind: ImageHubAssetKindImage, Path: p}, nil)
		if err != nil {
			slog.Warn("角色剧照登记失败（不影响保存）", "character_id", charID, "path", p, "error", err)
		}
		return
	}
}
