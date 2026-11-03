package pantheon

import (
	"strings"
	"testing"
)

func observeSession(participants ...Participant) *Session {
	return &Session{
		ID: "pan_observe", Title: "观测", Status: StatusRunning,
		Participants: participants,
	}
}

// 终态席位不许被"重新置为运行"：这是 2026-10-31 那条
// "finished 之后又来一条 started" 的线上故障的回归测试。
func TestObserveSeatRejectsIllegalTransition(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner())
	defer m.Close()
	m.authorityID = "test-authority"
	session := observeSession(Participant{AgentID: "alpha", Name: "ALPHA", Status: SeatFinished})

	err := m.observeSeatLocked(session, "alpha", seatObservation{
		Origin: OriginLauncher, Status: SeatRunning, Evidence: "重新开始发言",
	})
	if err == nil {
		t.Fatal("finished 到 running 必须被拒绝")
	}
	if !strings.Contains(err.Error(), "席位状态不能从") {
		t.Fatalf("拒绝理由不可读：%v", err)
	}
	got := session.Participants[0]
	if got.Status != SeatFinished {
		t.Fatalf("非法转移被写进了状态：%s", got.Status)
	}
	if got.Evidence != "" || got.Observed.Revision != 0 {
		t.Fatalf("非法转移不该留下观测痕迹：%+v", got.Observed)
	}
}

// M10：快照与心跳只重绘，不是证据，不许给安静的席位续命。
func TestObserveSeatSnapshotDoesNotRefreshFreshness(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner())
	defer m.Close()
	m.authorityID = "test-authority"
	session := observeSession(Participant{
		AgentID: "alpha", Name: "ALPHA", Status: SeatWaiting,
		Observed: Observed{Origin: OriginAdapter, AuthorityID: "test-authority", Incarnation: 1, Revision: 3, At: 111},
	})

	for _, origin := range []string{OriginSnapshot, OriginHeartbeat} {
		if err := m.observeSeatLocked(session, "alpha", seatObservation{
			Origin: origin, Status: SeatRunning, At: 999,
		}); err != nil {
			t.Fatalf("%s 观测不该报错：%v", origin, err)
		}
	}
	got := session.Participants[0].Observed
	if got.Revision != 3 || got.At != 111 {
		t.Fatalf("快照/心跳续了命：%+v", got)
	}
	if session.Participants[0].Status != SeatWaiting {
		t.Fatalf("快照/心跳改了状态：%s", session.Participants[0].Status)
	}
	if isFreshnessEvidence(OriginSnapshot) || isFreshnessEvidence(OriginHeartbeat) {
		t.Fatal("快照/心跳不该被当成证据")
	}
}

// 同态观测不是状态转移：只推进 revision 与证据时间。
func TestObserveSeatSameStateAdvancesRevision(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner())
	defer m.Close()
	m.authorityID = "test-authority"
	session := observeSession(Participant{
		AgentID: "alpha", Name: "ALPHA", Status: SeatRunning,
		Observed: Observed{Origin: OriginLauncher, AuthorityID: "test-authority", Incarnation: 1, Revision: 3, At: 111},
	})

	if err := m.observeSeatLocked(session, "alpha", seatObservation{
		Origin: OriginEvent, Status: SeatRunning, Evidence: "又吐了一段", At: 222,
	}); err != nil {
		t.Fatal(err)
	}
	got := session.Participants[0]
	if got.Status != SeatRunning {
		t.Fatalf("状态被改了：%s", got.Status)
	}
	if got.Observed.Revision != 4 || got.Observed.At != 222 {
		t.Fatalf("观测没推进：%+v", got.Observed)
	}
	if got.Observed.Incarnation != 1 {
		t.Fatalf("同态观测不该抬 incarnation：%d", got.Observed.Incarnation)
	}
}

// §2.2：换进程后 revision 从 1 重新计数，旧 authority 的序号不可比。
func TestObserveSeatRevisionResetsOnForeignAuthority(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner())
	defer m.Close()
	m.authorityID = "test-authority"
	session := observeSession(Participant{
		AgentID: "alpha", Name: "ALPHA", Status: SeatFinishedUnverified,
		Observed: Observed{Origin: OriginRecovery, AuthorityID: "old-process", Incarnation: 2, Revision: 41, At: 111},
	})

	if err := m.observeSeatLocked(session, "alpha", seatObservation{
		Origin: OriginAdapter, Status: SeatFinished, Evidence: "补上了收声", At: 333,
	}); err != nil {
		t.Fatal(err)
	}
	got := session.Participants[0].Observed
	if got.Revision != 1 {
		t.Fatalf("换了 authority 还接着数：%d", got.Revision)
	}
	if got.AuthorityID != "test-authority" {
		t.Fatalf("authority 没更新：%s", got.AuthorityID)
	}
	if got.Incarnation != 2 {
		t.Fatalf("incarnation 不该被观测改掉：%d", got.Incarnation)
	}
	if session.Participants[0].Status != SeatFinished {
		t.Fatalf("唯一升级通道没生效：%s", session.Participants[0].Status)
	}
}

// 重试是新的 attempt：抬 incarnation，绕过转移表，旧终态不算数。
func TestRebindSeatBumpsIncarnation(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner())
	defer m.Close()
	m.authorityID = "test-authority"
	session := observeSession(Participant{
		AgentID: "alpha", Name: "ALPHA", Status: SeatFailed,
		Observed: Observed{Origin: OriginProcess, AuthorityID: "test-authority", Incarnation: 1, Revision: 7, At: 111},
	})

	m.rebindSeatLocked(session, "alpha", SeatRetrying, "用户重试当前调用")
	got := session.Participants[0]
	if got.Status != SeatRetrying {
		t.Fatalf("状态没重绑：%s", got.Status)
	}
	if got.Observed.Incarnation != 2 || got.Observed.Revision != 1 {
		t.Fatalf("新 attempt 的観测没重置：%+v", got.Observed)
	}
	if got.Evidence != "用户重试当前调用" {
		t.Fatalf("证据没写上：%q", got.Evidence)
	}
}
