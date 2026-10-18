package app

import "testing"

func TestManagedCatalogModels_MarksOnlyActiveProfileRunning(t *testing.T) {
	// 2026-10-18 随批修正：35B 档退役后 tianxuanProfiles 只剩 2 档，原硬编码
	// 下标 [3] 越界；改为按末档判定，意图不变（仅活跃档 running，其余 stopped）。
	active := tianxuanProfiles[len(tianxuanProfiles)-1]
	served := []string{active.ModelID}
	models := managedCatalogModels(tianxuanProfiles, served, active.ID)
	if len(models) != len(tianxuanProfiles) {
		t.Fatalf("catalog size = %d, want %d", len(models), len(tianxuanProfiles))
	}
	for _, m := range models {
		want := "stopped"
		if m.ID == served[0] {
			want = "running"
		}
		if m.Status != want {
			t.Fatalf("model %s status = %q, want %q", m.ID, m.Status, want)
		}
		if m.Kind != "llm" {
			t.Fatalf("model %s kind = %q, want llm", m.ID, m.Kind)
		}
	}
}

func TestManagedCatalogModels_StoppedEngineKeepsFullCatalog(t *testing.T) {
	models := managedCatalogModels(strataProfiles, nil, "")
	if len(models) != len(strataProfiles) {
		t.Fatalf("catalog size = %d, want %d", len(models), len(strataProfiles))
	}
	for _, m := range models {
		if m.Status != "stopped" {
			t.Fatalf("stopped engine model %s status = %q", m.ID, m.Status)
		}
	}
}

func TestManagedProfileForModel_MatchesIdAndModelId(t *testing.T) {
	for _, profiles := range [][]engineProfileOption{tianxuanProfiles, strataProfiles} {
		for _, p := range profiles {
			if got, ok := managedProfileForModel(profiles, p.ID); !ok || got.ID != p.ID {
				t.Fatalf("profile id %q not matched: %+v", p.ID, got)
			}
			if got, ok := managedProfileForModel(profiles, p.ModelID); !ok || got.ID != p.ID {
				t.Fatalf("model id %q not matched: %+v", p.ModelID, got)
			}
		}
	}
}
