package workcost

// ── 跨项目对标查漏（11.3；复核增强而非替代）────────────────
//
// 出口判据（longterm plan §4 11.3）：查漏结论带溯源引用；误报率有评测集。
// 定位铁律：本引擎只产出**带证据的线索清单**给作者复核，绝不自动改数据。
//
// 诚实口径：这是确定性统计对标引擎（语料覆盖率阈值 + 宁缺勿误闸），不是
// LLM 判读——结论必须可溯源（逐条引用支撑它的项目/定额），误报率必须在
// 评测集上可复现地钉住（audit_test.go 即评测集：已知发现全中 + 健康对照
// 零误报 + 覆盖不足不报）。
//
// 两类发现：
//   bill_gap  清单缺项——被审项目缺一个「多数同行项目都有」的分部
//             （证据=哪些同行有、各有多少清单项）
//   quota_gap 同名定额缺组成——同名同单位定额组里，多数成员都含的某种
//             工料机资源，个别成员没有（证据=哪些成员含、含量多少）
//
// 宁缺勿误闸（全低阈值=刷屏误报；缺一不报）：
//   语料项目 ≥3；分部/资源在**其它成员**中 ≥2 个出现且覆盖率 ≥60%。

import (
	"fmt"
	"sort"
	"strings"
)

// auditThresholds 宁缺勿误闸（评测集钉死语义；调参=改这里+跑评测集）。
const (
	auditMinProjects      = 3 // 语料项目数下限（含被审项目）
	auditMinPeerSupport   = 2 // 其它成员中至少几个出现
	auditCoverageRequired = 0.6
)

// AuditEvidence 一条溯源引用：支撑本结论的具体项目/定额。
type AuditEvidence struct {
	Ref     string `json:"ref"`     // 项目名 / 定额编码
	Source  string `json:"source"`  // 来源文件（项目导入：<文件名> 的溯源）
	Support string `json:"support"` // 支撑内容人可读（清单项数/含量）
}

// AuditFinding 一条查漏线索。
type AuditFinding struct {
	Kind     string          `json:"kind"` // bill_gap | quota_gap
	Target   string          `json:"target"`
	Detail   string          `json:"detail"`
	Evidence []AuditEvidence `json:"evidence"`
}

// AuditReport 查漏报告（只读）。
type AuditReport struct {
	BillGaps    []AuditFinding `json:"billGaps"`
	QuotaGaps   []AuditFinding `json:"quotaGaps"`
	AuditedProj string         `json:"auditedProj"`
	ProjectHits int            `json:"projectHits"`
}

// auditProject 单个项目的对标画像。
type auditProject struct {
	ID        int64
	Name      string
	FileName  string
	ItemCount int
	Divisions map[string]int // 分部 → 清单项数
}

// auditQuota 定额成员画像（同名同单位组的一员）。
type auditQuota struct {
	Code     string
	Title    string
	Unit     string
	Source   string
	Resource map[string]string // resourceCode → 资源标题
}

// AuditBillGaps 纯函数：清单缺项对标（被审项目 vs 语料）。
func AuditBillGaps(projects []auditProject, targetID int64) []AuditFinding {
	out := []AuditFinding{}
	if len(projects) < auditMinProjects {
		return out
	}
	var target *auditProject
	peers := []*auditProject{}
	for i := range projects {
		p := &projects[i]
		if p.ID == targetID {
			target = p
		} else {
			peers = append(peers, p)
		}
	}
	if target == nil || len(peers) == 0 {
		return out
	}
	// 分部 → 有该分部的同行项目。
	support := map[string][]*auditProject{}
	for _, p := range peers {
		if p.ItemCount <= 0 {
			continue // 空项目不进语料（拉低覆盖率的噪声）
		}
		for div := range p.Divisions {
			support[div] = append(support[div], p)
		}
	}
	divs := make([]string, 0, len(support))
	for d := range support {
		divs = append(divs, d)
	}
	sort.Strings(divs)
	peerN := 0
	for _, p := range peers {
		if p.ItemCount > 0 {
			peerN++
		}
	}
	for _, div := range divs {
		if _, has := target.Divisions[div]; has {
			continue
		}
		hits := support[div]
		if len(hits) < auditMinPeerSupport {
			continue
		}
		if float64(len(hits)) < auditCoverageRequired*float64(peerN) {
			continue
		}
		ev := make([]AuditEvidence, 0, len(hits))
		for _, h := range hits {
			ev = append(ev, AuditEvidence{Ref: h.Name, Source: h.FileName,
				Support: fmt.Sprintf("「%s」分部 %d 项清单", div, h.Divisions[div])})
		}
		out = append(out, AuditFinding{
			Kind:     "bill_gap",
			Target:   div,
			Detail:   fmt.Sprintf("「%s」分部在本项目缺失：%d/%d 个同行项目都有（占 %.0f%%），疑似漏项", div, len(hits), peerN, 100*float64(len(hits))/float64(peerN)),
			Evidence: ev,
		})
	}
	return out
}

// AuditQuotaGaps 纯函数：同名定额缺组成对标（全语料，逐组逐资源）。
func AuditQuotaGaps(quotas []auditQuota) []AuditFinding {
	out := []AuditFinding{}
	// 同名同单位分组。
	groups := map[string][]auditQuota{}
	var keys []string
	for _, q := range quotas {
		k := strings.TrimSpace(q.Title) + "\x00" + strings.TrimSpace(q.Unit)
		if _, seen := groups[k]; !seen {
			keys = append(keys, k)
		}
		groups[k] = append(groups[k], q)
	}
	sort.Strings(keys)
	for _, k := range keys {
		ms := groups[k]
		if len(ms) < auditMinProjects {
			continue // 组太小，没有「多数派」可言
		}
		// 资源 → 含它的成员。
		support := map[string][]auditQuota{}
		rtitle := map[string]string{}
		for _, q := range ms {
			for rc, rt := range q.Resource {
				support[rc] = append(support[rc], q)
				if _, ok := rtitle[rc]; !ok {
					rtitle[rc] = rt
				}
			}
		}
		rcs := make([]string, 0, len(support))
		for rc := range support {
			rcs = append(rcs, rc)
		}
		sort.Strings(rcs)
		n := len(ms)
		for _, rc := range rcs {
			hits := support[rc]
			if len(hits) < auditMinPeerSupport {
				continue
			}
			if float64(len(hits)) < auditCoverageRequired*float64(n) {
				continue
			}
			// 缺它的成员 = 发现。
			for _, q := range ms {
				if _, has := q.Resource[rc]; has {
					continue
				}
				ev := make([]AuditEvidence, 0, len(hits))
				for _, h := range hits {
					ev = append(ev, AuditEvidence{Ref: h.Code, Source: h.Source,
						Support: fmt.Sprintf("「%s」（%s）含 %s", h.Title, h.Unit, rtitle[rc])})
				}
				out = append(out, AuditFinding{
					Kind:   "quota_gap",
					Target: fmt.Sprintf("%s（%s）", q.Title, q.Unit),
					Detail: fmt.Sprintf("定额 %s「%s」未含资源 %s「%s」：同名的 %d 个定额里 %d 个都含（占 %.0f%%），疑似缺组成行",
						q.Code, q.Title, rc, rtitle[rc], n, len(hits), 100*float64(len(hits))/float64(n)),
					Evidence: ev,
				})
			}
		}
	}
	return out
}

// BillAuditScan 跨项目对标查漏（只读）：bill_gap 针对被审项目（projectID），
// quota_gap 全语料逐同名组。projectID 不存在时报 billGaps 空（诚实口径）。
func (s *Store) BillAuditScan(projectID int64) (*AuditReport, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	projects, err := s.BillProjects()
	if err != nil {
		return nil, err
	}
	profiles := make([]auditProject, 0, len(projects))
	rep := &AuditReport{BillGaps: []AuditFinding{}, QuotaGaps: []AuditFinding{}}
	for _, p := range projects {
		items, err := s.BillItems(p.ID)
		if err != nil {
			return nil, err
		}
		div := map[string]int{}
		for _, it := range items {
			if d := strings.TrimSpace(it.Division); d != "" {
				div[d]++
			}
		}
		profiles = append(profiles, auditProject{
			ID: p.ID, Name: p.Name, FileName: p.FileName, ItemCount: len(items), Divisions: div,
		})
		if p.ID == projectID {
			rep.AuditedProj = p.Name
		}
	}
	rep.ProjectHits = len(profiles)
	rep.BillGaps = AuditBillGaps(profiles, projectID)

	quotas, err := s.allQuotasWithItems()
	if err != nil {
		return nil, err
	}
	qs := make([]auditQuota, 0, len(quotas))
	for _, q := range quotas {
		res := make(map[string]string, len(q.Items))
		for _, it := range q.Items {
			if rc := strings.TrimSpace(it.ResourceCode); rc != "" {
				res[rc] = it.Title
			}
		}
		qs = append(qs, auditQuota{Code: q.Code, Title: q.Title, Unit: q.Unit, Source: q.Source, Resource: res})
	}
	rep.QuotaGaps = AuditQuotaGaps(qs)
	return rep, nil
}
