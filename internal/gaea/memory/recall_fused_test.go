package memory

import (
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/db"
)

// RecallBlockFused（聊天席融合注入，用户拍板：聊天接、办公保缓存不接）三钉：
// 图谱发现臂、办公席旧口径不变、休眠惩罚排序。
func TestRecallBlockFusedGraphDiscovery(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	// A 词法命中；B 与查询零词面重叠，仅经 A 的 [[互引]] 可达（发现面）。
	if _, err := s.Save(Memory{
		Name: "deploy-runbook", Title: "部署手册",
		Description: "部署步骤与回滚口径", Kind: KindSemantic, Type: TypeProject,
		Body: "部署前置检查，回滚见 [[rollback-policy]]",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save(Memory{
		Name: "rollback-policy", Title: "回滚口径",
		Description: "灰度失败的回滚纪律", Kind: KindSemantic, Type: TypeProject,
		Body: "先摘流量再回滚版本",
	}); err != nil {
		t.Fatal(err)
	}
	set := &Set{Store: s}
	block := set.RecallBlockFused("部署", 800)
	if !strings.Contains(block, "部署手册") {
		t.Fatalf("词法命中应注入：%s", block)
	}
	if !strings.Contains(block, "[MEM:rollback-policy]") {
		t.Fatalf("图谱发现臂未带出互引邻居：%s", block)
	}
	// 办公席旧口径：同一存储、同一查询——零词面命中的 B 不注入（分席差异实证；
	// 判据用 rollback-policy 的引用键，避免撞 deploy-runbook 描述里的「回滚口径」）。
	office := set.RecallBlock("部署", 800)
	if strings.Contains(office, "[MEM:rollback-policy]") {
		t.Fatalf("办公席旧口径不应吃图谱发现面：%s", office)
	}
	if !strings.Contains(office, "部署手册") {
		t.Fatalf("办公席词法命中丢失：%s", office)
	}
}

// 休眠惩罚×稳定性：同词面同词龄的两条，被引用多的排前（引用长稳定性）。
func TestRecallBlockFusedStabilityRanking(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	old := time.Now().Add(-100 * 24 * time.Hour).UTC().Truncate(time.Second)
	// 两条 procedural 同查询词面、同 100 天未触达（都是衰减态）。
	if _, err := s.Save(Memory{
		Name: "rule-cited", Title: "被引用规则",
		Description: "发布前置冒烟口径", Kind: KindProcedural, Type: TypeProject,
		Body: "发布前置冒烟两百校验",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save(Memory{
		Name: "rule-stale", Title: "未被引用规则",
		Description: "发布前置冒烟口径变体", Kind: KindProcedural, Type: TypeProject,
		Body: "发布前置冒烟两百校验补充",
	}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if err := s.AppendFeedbackEvent(Event{At: time.Now().UnixMilli(), Op: OpCite, Name: "rule-cited"}); err != nil {
			t.Fatal(err)
		}
	}
	// 两条同回拨 100 天（updated_at 决定词法臂时序项；公平对比）。
	for _, n := range []string{"rule-cited", "rule-stale"} {
		if _, err := gdb.Exec(`UPDATE facts SET updated_at=?, last_used_at='' WHERE name=?`, old.Format(time.RFC3339), n); err != nil {
			t.Fatal(err)
		}
	}
	set := &Set{Store: s}
	block := set.RecallBlockFused("发布 冒烟", 800)
	ic, istale := strings.Index(block, "被引用规则"), strings.Index(block, "未被引用规则")
	if ic < 0 || istale < 0 {
		t.Fatalf("两条都应注入：%s", block)
	}
	if ic > istale {
		t.Fatalf("稳定性排序失效（引用多的应排前）：%s", block)
	}
}

// 空库/无图/无引用事件：零 SQL 崩溃，回退为纯词法注入（降级语义）。
func TestRecallBlockFusedDegradation(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	if _, err := s.Save(Memory{
		Name: "solo-fact", Title: "孤立事实",
		Description: "没有任何互引与引用事件", Kind: KindSemantic, Type: TypeProject,
		Body: "普通事实",
	}); err != nil {
		t.Fatal(err)
	}
	set := &Set{Store: s}
	block := set.RecallBlockFused("孤立", 800)
	if !strings.Contains(block, "孤立事实") {
		t.Fatalf("降级后词法注入丢失：%s", block)
	}
}
