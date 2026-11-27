package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/gaea/sandbox"
)

// 2026-11-27 AI 反馈 #2（包2 会话日志实锤）：模型把一份数据字典整个当成 bash
// 的参数发出来（37 个键、没有 command），连发 4 次。报错要点名「这是数据对象、
// 不是命令」并指路；键排序输出——旧报错按 map 随机序罗列，模型以为参数每次都被打散。
func TestBash_MissingCommandDataObject(t *testing.T) {
	b := bash{}
	_, err := b.Execute(context.Background(), json.RawMessage(`{"BC02":5929.68,"BC01":15506.11,"BC37":48.5,"BC03":29836.17}`))
	if err == nil {
		t.Fatal("缺 command 必须报错")
	}
	for _, want := range []string{"4 个参数键", "（BC01,BC02,BC03,BC37）", "数据对象", "不是命令", "write_file 把它写成 JSON 文件", "bash 执行脚本路径"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("数据对象话术应含 %q，实得: %v", want, err)
		}
	}
	// 键多时限长：列前 12 个并说明已排序。
	var big strings.Builder
	big.WriteString("{")
	for i := 0; i < 40; i++ {
		if i > 0 {
			big.WriteString(",")
		}
		fmt.Fprintf(&big, `"item-code-%02d":%d`, 39-i, i)
	}
	big.WriteString("}")
	_, err = b.Execute(context.Background(), json.RawMessage(big.String()))
	if err == nil || !strings.Contains(err.Error(), "40 个参数键（item-code-00,item-code-01,item-code-02,item-code-03,item-code-04,item-code-05,item-code-06,item-code-07,item-code-08,item-code-09,item-code-10,item-code-11 …（共 40 个，已排序））") {
		t.Fatalf("键多时应按序列前 12 个并说明总数，实得: %v", err)
	}
	// 键名写错（单个未知键）不是数据对象，仍走通用话术。
	_, err = b.Execute(context.Background(), json.RawMessage(`{"cmd":"echo hi"}`))
	if err == nil || strings.Contains(err.Error(), "数据对象") {
		t.Fatalf("单个错键不应套数据对象话术，实得: %v", err)
	}
}

// 2026-11-27 AI 反馈 #5 根修：Windows 上经 argv 传给 bash -c 的命令行超过约
// 8.2K 字符会被静默截断（退出码仍为 0）。超过阈值的命令必须改走临时脚本、
// 完整送达——一条远超上限的 heredoc 写出的文件要一行不少。
func TestBash_LongCommandRunsFromScript(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("argv 截断上限是 Windows + Git Bash 的实测现象")
	}
	shell := sandbox.ResolveShell()
	if shell.Kind != sandbox.ShellBash {
		t.Skip("bash unavailable")
	}
	dir := t.TempDir()
	var sb strings.Builder
	sb.WriteString("cat <<'EOF' > out.txt\n")
	const n = 400
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb, "line-%04d %s\n", i, strings.Repeat("数据块", 8))
	}
	sb.WriteString("EOF\nwc -l < out.txt\n")
	cmd := sb.String()
	if utf8.RuneCountInString(cmd) <= bashArgvScriptCap {
		t.Fatalf("测试命令应超过脚本阈值：%d", utf8.RuneCountInString(cmd))
	}
	args, _ := json.Marshal(map[string]any{"command": cmd})
	out, err := (bash{shell: shell, workDir: dir}).Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("长命令应完整执行: %v（输出 %q）", err, out)
	}
	if strings.Contains(out, "here-document") {
		t.Fatalf("长 heredoc 不应再被截断: %q", out)
	}
	data, rerr := os.ReadFile(filepath.Join(dir, "out.txt"))
	if rerr != nil {
		t.Fatalf("heredoc 应写出文件: %v", rerr)
	}
	if got := strings.Count(string(data), "\n"); got != n || !strings.Contains(string(data), "line-0399") {
		t.Fatalf("heredoc 应一行不少：得到 %d 行（应为 %d），尾部 %q", got, n, string(data[len(data)-40:]))
	}
}

// 阈值以内的短命令照旧走 argv -c，不改道。
func TestBash_ShortCommandKeepsArgv(t *testing.T) {
	shell := sandbox.ResolveShell()
	if shell.Kind != sandbox.ShellBash {
		t.Skip("bash unavailable")
	}
	argvOf := func(cmd string) []string {
		argv, _ := sandbox.Command(sandbox.Spec{}, shell, cmd)
		return argv
	}
	argv, cleanup, switched := longCommandScript(argvOf("echo hi"), shell, false, "echo hi")
	if switched || len(argv) != 3 || argv[1] != "-c" || argv[2] != "echo hi" {
		t.Fatalf("短命令不应改道：%v switched=%v", argv, switched)
	}
	cleanup()
	// 改道：argv 变成 [bash, 脚本路径]，脚本内容与命令体逐字节一致，清理后文件不在。
	long := strings.Repeat("x", bashArgvScriptCap+10)
	argv, cleanup, switched = longCommandScript(argvOf(long), shell, false, long)
	if !switched || len(argv) != 2 || argv[1] == "" || argv[1] == "-c" {
		t.Fatalf("长命令应改走脚本文件：%v switched=%v", argv, switched)
	}
	data, err := os.ReadFile(argv[1])
	cleanup()
	if err != nil || string(data) != long {
		t.Fatalf("临时脚本内容应与命令体一致: %v", err)
	}
	if _, err := os.Stat(argv[1]); !os.IsNotExist(err) {
		t.Fatalf("清理后临时脚本应已删除: %v", err)
	}
	// 强制沙箱时不改道（命令要在容器里重放）。
	if _, _, switched = longCommandScript(argvOf(long), shell, true, long); switched {
		t.Fatal("沙箱强制时不应改走宿主临时脚本")
	}
}
