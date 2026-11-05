package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db/repos"
)

// 众神殿刀2 存量自愈：等值隐式生日（ackemBirthday==firstMetDate）非 gaea 清
// nil、gaea 改内置 06-11；显式配置的生日不动；首次动刀前落一份 hermes.db 备份。
func TestHealImplicitBirthday(t *testing.T) {
	a := newChatServiceTestApp(t)
	ws := a.whisperState
	root := ws.whisperDataRoot
	met := time.Date(2026, 8, 6, 17, 5, 34, 0, time.Local)

	newStateWithBirthday := func(bd, firstMet *time.Time) whisper.FullState {
		st := whisper.FullState{}
		st.Emotion.PrimaryLabel = "WARM_ENGAGED"
		st.AckemBirthday = bd
		st.FirstMetDate = firstMet
		return st
	}

	// ① 非 gaea：等值隐式生日 → 清 nil 并落库（生产序=先恢复状态再自愈）
	if err := repos.SaveCompanionStateToDB(root, "whisper_heal_plain", newStateWithBirthday(&met, &met)); err != nil {
		t.Fatalf("seed: %v", err)
	}
	orchPlain := whisper.NewOrchestrator("whisper_heal_plain", *whisper.GetPreset("deredere"))
	orchPlain.DataRoot = root
	if seeded, err := repos.LoadCompanionStateFromDB(root, "whisper_heal_plain"); err != nil || seeded == nil {
		t.Fatalf("load seed: %v", err)
	} else {
		orchPlain.State = *seeded
	}
	ws.healImplicitBirthday(orchPlain)
	if orchPlain.State.AckemBirthday != nil {
		t.Fatalf("非 gaea 等值隐式生日应清 nil，got %v", orchPlain.State.AckemBirthday)
	}
	loaded, err := repos.LoadCompanionStateFromDB(root, "whisper_heal_plain")
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if loaded == nil || loaded.AckemBirthday != nil {
		t.Fatalf("自愈应落库（非 gaea），got %+v", loaded)
	}

	// ② gaea：等值 → 内置 06-11（年取首聊年）
	if err := repos.SaveCompanionStateToDB(root, "whisper_heal_gaea", newStateWithBirthday(&met, &met)); err != nil {
		t.Fatalf("seed gaea: %v", err)
	}
	orchGaea := whisper.NewOrchestrator("whisper_heal_gaea", *whisper.GetPreset("gaea"))
	orchGaea.DataRoot = root
	if seeded, err := repos.LoadCompanionStateFromDB(root, "whisper_heal_gaea"); err != nil || seeded == nil {
		t.Fatalf("load seed gaea: %v", err)
	} else {
		orchGaea.State = *seeded
	}
	ws.healImplicitBirthday(orchGaea)
	if orchGaea.State.AckemBirthday == nil {
		t.Fatal("gaea 等值隐式生日应改为内置 06-11")
	}
	if bd := *orchGaea.State.AckemBirthday; bd.Month() != time.June || bd.Day() != 11 {
		t.Fatalf("gaea 自愈后生日 = %s, want 06-11", bd.Format("01-02"))
	}
	loadedGaea, err := repos.LoadCompanionStateFromDB(root, "whisper_heal_gaea")
	if err != nil {
		t.Fatalf("reload gaea: %v", err)
	}
	if loadedGaea == nil || loadedGaea.AckemBirthday == nil || loadedGaea.AckemBirthday.Day() != 11 {
		t.Fatalf("gaea 自愈应落库，got %+v", loadedGaea)
	}

	// ③ 显式配置（不等值）：不动
	explicit := met.AddDate(0, 1, 0)
	if err := repos.SaveCompanionStateToDB(root, "whisper_heal_explicit", newStateWithBirthday(&explicit, &met)); err != nil {
		t.Fatalf("seed explicit: %v", err)
	}
	orchExplicit := whisper.NewOrchestrator("whisper_heal_explicit", *whisper.GetPreset("deredere"))
	orchExplicit.DataRoot = root
	if seeded, err := repos.LoadCompanionStateFromDB(root, "whisper_heal_explicit"); err != nil || seeded == nil {
		t.Fatalf("load seed explicit: %v", err)
	} else {
		orchExplicit.State = *seeded
	}
	ws.healImplicitBirthday(orchExplicit)
	if orchExplicit.State.AckemBirthday == nil || !orchExplicit.State.AckemBirthday.Equal(explicit) {
		t.Fatalf("显式生日不得被自愈触碰，got %v", orchExplicit.State.AckemBirthday)
	}

	// ④ 首次动刀前备份存在
	if _, err := os.Stat(filepath.Join(root, "hermes.db.bak-birthday-heal")); err != nil {
		t.Fatalf("首次自愈应落一份 hermes.db 备份: %v", err)
	}
}
