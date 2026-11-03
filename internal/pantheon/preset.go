package pantheon

import (
	"fmt"
	"strings"
)

const (
	PresetRoundtable        = "roundtable"
	PresetTargetedChallenge = "targeted_challenge"
	PresetRedTeam           = "red_team"
)

func NormalizePreset(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", PresetRoundtable:
		return PresetRoundtable, nil
	case PresetTargetedChallenge:
		return PresetTargetedChallenge, nil
	case PresetRedTeam:
		return PresetRedTeam, nil
	default:
		return "", fmt.Errorf("未知议事形态 %q", raw)
	}
}

func PresetLabel(preset string) string {
	switch preset {
	case PresetTargetedChallenge:
		return "定向质询"
	case PresetRedTeam:
		return "红队评审"
	default:
		return "圆桌会诊"
	}
}

func isDirectedPreset(preset string) bool {
	return preset == PresetTargetedChallenge || preset == PresetRedTeam
}
