// schedule/calendar_parity_test.go — Go/TS 日历语义对拍（IN3-02，批 16 线 3）。
//
// 读取 testdata/calendar-parity.json（与 frontend/src/schedule/calendar.ts 的
// parity 测试块共享同一份数据），对 Go 侧实现逐条断言冻结的双端共识。目的：
// 日历规则求值唯一入口 IsWorkingDate 及其换算族（WdToDate/DateToWd/
// DeadlineWorkdays）若被改动而前端镜像未同步（或反之），本测试即红——
// 「改 Go 漏改 TS」风险的硬对冲（范式同 ops_golden 对拍）。
//
// 期望值是「共识冻结」而非「Go 现算现写」：任何一侧单独漂移都红，两侧同时
// 有意变更时须同步改本 fixture（追加样本只增不改）。
package schedule

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type parityFile struct {
	Calendars map[string]Calendar `json:"calendars"`
	// IsWorkingDate 原始判据用例（不经 NormalizeCalendar，双端一致）。
	IsWorkingDate []struct {
		Cal     string `json:"cal"`
		Date    string `json:"date"`
		Working bool   `json:"working"`
		Why     string `json:"why"`
	} `json:"isWorkingDate"`
	WdToDate []struct {
		Cal   string `json:"cal"`
		Start string `json:"start"`
		Idx   int    `json:"idx"`
		Want  string `json:"want"`
		Why   string `json:"why"`
	} `json:"wdToDate"`
	DateToWd []struct {
		Cal   string `json:"cal"`
		Start string `json:"start"`
		Date  string `json:"date"`
		Want  *int   `json:"want"` // null = 非工作日/早于开工（Go 侧 (0,false)）
		Why   string `json:"why"`
	} `json:"dateToWd"`
	DeadlineWorkdays []struct {
		Cal      string `json:"cal"`
		Start    string `json:"start"`
		Deadline string `json:"deadline"`
		Want     int    `json:"want"`
		Why      string `json:"why"`
	} `json:"deadlineWorkdays"`
}

func loadParity(t *testing.T) parityFile {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "calendar-parity.json"))
	if err != nil {
		t.Fatalf("读取 parity golden 失败：%v", err)
	}
	var f parityFile
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parity golden 解析失败：%v", err)
	}
	// 防空转：样本被清空时对拍形同虚设，直接红。
	if len(f.Calendars) == 0 || len(f.IsWorkingDate) == 0 || len(f.WdToDate) == 0 || len(f.DateToWd) == 0 || len(f.DeadlineWorkdays) == 0 {
		t.Fatalf("parity golden 样本为空（calendars=%d isWorkingDate=%d wdToDate=%d dateToWd=%d deadlineWorkdays=%d）",
			len(f.Calendars), len(f.IsWorkingDate), len(f.WdToDate), len(f.DateToWd), len(f.DeadlineWorkdays))
	}
	return f
}

func parityDate(iso string) time.Time {
	d, err := time.Parse("2006-01-02", iso)
	if err != nil {
		panic(err) // fixture 内日期串由 fixture 自身保证合法
	}
	return d
}

func TestCalendarParityGolden(t *testing.T) {
	f := loadParity(t)

	t.Run("IsWorkingDate", func(t *testing.T) {
		for _, c := range f.IsWorkingDate {
			cal, ok := f.Calendars[c.Cal]
			if !ok {
				t.Fatalf("未知日历 %q", c.Cal)
			}
			if got := IsWorkingDate(parityDate(c.Date), cal); got != c.Working {
				t.Errorf("%s %s：%s — got %v want %v", c.Cal, c.Date, c.Why, got, c.Working)
			}
		}
	})

	t.Run("WdToDate", func(t *testing.T) {
		for _, c := range f.WdToDate {
			cal := f.Calendars[c.Cal]
			got := ISOOf(WdToDate(c.Start, c.Idx, &cal))
			if got != c.Want {
				t.Errorf("%s %s idx=%d：%s — got %s want %s", c.Cal, c.Start, c.Idx, c.Why, got, c.Want)
			}
		}
	})

	t.Run("DateToWd", func(t *testing.T) {
		for _, c := range f.DateToWd {
			cal := f.Calendars[c.Cal]
			got, ok := DateToWd(c.Start, c.Date, &cal)
			switch {
			case c.Want == nil:
				if ok {
					t.Errorf("%s %s→%s：%s — got (%d,true) want 非工作日", c.Cal, c.Start, c.Date, c.Why, got)
				}
			case !ok:
				t.Errorf("%s %s→%s：%s — got 非工作日 want %d", c.Cal, c.Start, c.Date, c.Why, *c.Want)
			case got != *c.Want:
				t.Errorf("%s %s→%s：%s — got %d want %d", c.Cal, c.Start, c.Date, c.Why, got, *c.Want)
			}
		}
	})

	t.Run("DeadlineWorkdays", func(t *testing.T) {
		for _, c := range f.DeadlineWorkdays {
			cal := f.Calendars[c.Cal]
			if got := DeadlineWorkdays(c.Start, c.Deadline, &cal); got != c.Want {
				t.Errorf("%s %s→%s：%s — got %d want %d", c.Cal, c.Start, c.Deadline, c.Why, got, c.Want)
			}
		}
	})
}
