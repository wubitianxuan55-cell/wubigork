package herdsman

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeConfig 在临时目录写入样例 config.yaml，返回其路径。
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写入样例 config.yaml 失败: %v", err)
	}
	return path
}

// TestCheckLanExposureExposed 覆盖 lan_accessible=true 与 port 提取。
func TestCheckLanExposureExposed(t *testing.T) {
	path := writeConfig(t, `general:
    data_dir: C:/tmp
api:
    lan_accessible: true
    port: 9090
    log_request_payload: false
log:
    level: info
`)
	got := CheckLanExposure(path)
	if got.ConfigMissing {
		t.Errorf("ConfigMissing 应为 false，实际为 true")
	}
	if got.ParseError != "" {
		t.Errorf("ParseError 应为空，实际为 %q", got.ParseError)
	}
	if !got.Exposed {
		t.Errorf("Exposed 应为 true，实际为 false")
	}
	if got.Port != 9090 {
		t.Errorf("Port 应为 9090，实际为 %d", got.Port)
	}
	if got.Guidance == "" {
		t.Errorf("Guidance 应为非空（含中文处置指引）")
	} else if !strings.Contains(got.Guidance, "lan_accessible") || !strings.Contains(got.Guidance, "重启") {
		t.Errorf("Guidance 应包含处置步骤（lan_accessible/重启），实际为 %q", got.Guidance)
	}
	if got.ConfigPath != path {
		t.Errorf("ConfigPath 应为 %q，实际为 %q", path, got.ConfigPath)
	}
}

// TestCheckLanExposureTrueDefaultPort 覆盖 lan_accessible=true 但未写 port（缺省 8080）。
func TestCheckLanExposureTrueDefaultPort(t *testing.T) {
	path := writeConfig(t, `api:
    lan_accessible: true
`)
	got := CheckLanExposure(path)
	if !got.Exposed {
		t.Errorf("Exposed 应为 true，实际为 false")
	}
	if got.Port != 8080 {
		t.Errorf("Port 应缺省为 8080，实际为 %d", got.Port)
	}
	if got.ParseError != "" {
		t.Errorf("ParseError 应为空，实际为 %q", got.ParseError)
	}
}

// TestCheckLanExposureFalse 覆盖 lan_accessible=false（含显式 port）。
func TestCheckLanExposureFalse(t *testing.T) {
	path := writeConfig(t, `api:
    lan_accessible: false
    port: 8080
`)
	got := CheckLanExposure(path)
	if got.Exposed {
		t.Errorf("Exposed 应为 false，实际为 true")
	}
	if got.Port != 8080 {
		t.Errorf("Port 应为 8080，实际为 %d", got.Port)
	}
	if got.Guidance != "" {
		t.Errorf("Exposed=false 时 Guidance 应为空，实际为 %q", got.Guidance)
	}
	if got.ParseError != "" {
		t.Errorf("ParseError 应为空，实际为 %q", got.ParseError)
	}
}

// TestCheckLanExposureDefaults 覆盖 api 段缺省：无 lan_accessible/port 或 api 段缺失。
func TestCheckLanExposureDefaults(t *testing.T) {
	t.Run("api段为空", func(t *testing.T) {
		path := writeConfig(t, `api:
    log_request_payload: true
`)
		got := CheckLanExposure(path)
		if got.Exposed {
			t.Errorf("Exposed 应缺省为 false，实际为 true")
		}
		if got.Port != 8080 {
			t.Errorf("Port 应缺省为 8080，实际为 %d", got.Port)
		}
		if got.ParseError != "" {
			t.Errorf("ParseError 应为空，实际为 %q", got.ParseError)
		}
	})
	t.Run("无api段", func(t *testing.T) {
		path := writeConfig(t, `general:
    data_dir: C:/tmp
download:
    concurrent_downloads: 3
`)
		got := CheckLanExposure(path)
		if got.Exposed {
			t.Errorf("无 api 段时 Exposed 应为 false，实际为 true")
		}
		if got.Port != 8080 {
			t.Errorf("无 api 段时 Port 应缺省为 8080，实际为 %d", got.Port)
		}
		if got.ParseError != "" {
			t.Errorf("ParseError 应为空，实际为 %q", got.ParseError)
		}
	})
}

// TestCheckLanExposureSectionBoundary 覆盖 api 段边界：
// 段前的顶层键、段后的顶层键及其同名子键都不应污染 api 段提取。
func TestCheckLanExposureSectionBoundary(t *testing.T) {
	path := writeConfig(t, `general:
    lan_accessible: true
    port: 7001
api:
    lan_accessible: true
    port: 9090
log:
    port: 7002
    lan_accessible: false
window:
    width: 1280
lan_accessible: false
`)
	got := CheckLanExposure(path)
	if !got.Exposed {
		t.Errorf("api 段 lan_accessible=true，Exposed 应为 true，实际为 false")
	}
	if got.Port != 9090 {
		t.Errorf("应提取 api 段内 port=9090，实际为 %d（受其他段污染）", got.Port)
	}
	if got.ParseError != "" {
		t.Errorf("ParseError 应为空，实际为 %q", got.ParseError)
	}
}

// TestCheckLanExposureMissingFile 覆盖文件不存在：Exposed=false、ConfigMissing=true。
func TestCheckLanExposureMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no_such_config.yaml")
	got := CheckLanExposure(path)
	if got.Exposed {
		t.Errorf("文件不存在时 Exposed 应为 false，实际为 true")
	}
	if !got.ConfigMissing {
		t.Errorf("文件不存在时 ConfigMissing 应为 true，实际为 false")
	}
	if got.ParseError == "" {
		t.Errorf("文件不存在时 ParseError 应说明原因，实际为空")
	}
	if got.Guidance != "" {
		t.Errorf("文件不存在时 Guidance 应为空，实际为 %q", got.Guidance)
	}
}

// TestCheckLanExposureEmptyPath 覆盖空路径：视为配置缺失。
func TestCheckLanExposureEmptyPath(t *testing.T) {
	got := CheckLanExposure("")
	if got.Exposed {
		t.Errorf("空路径时 Exposed 应为 false，实际为 true")
	}
	if !got.ConfigMissing {
		t.Errorf("空路径时 ConfigMissing 应为 true，实际为 false")
	}
	if got.ParseError == "" {
		t.Errorf("空路径时 ParseError 应说明原因，实际为空")
	}
}

// TestCheckLanExposureParseError 覆盖坏格式：非法布尔/非法端口 → ParseError。
func TestCheckLanExposureParseError(t *testing.T) {
	t.Run("lan_accessible非布尔", func(t *testing.T) {
		path := writeConfig(t, `api:
    lan_accessible: maybe
    port: 8080
`)
		got := CheckLanExposure(path)
		if got.ParseError == "" {
			t.Errorf("lan_accessible=maybe 应产生 ParseError，实际为空")
		}
		if got.Exposed {
			t.Errorf("解析失败时 Exposed 应为 false，实际为 true")
		}
	})
	t.Run("port非整数", func(t *testing.T) {
		path := writeConfig(t, `api:
    lan_accessible: true
    port: not-a-port
`)
		got := CheckLanExposure(path)
		if got.ParseError == "" {
			t.Errorf("port=not-a-port 应产生 ParseError，实际为空")
		}
	})
	t.Run("port负数", func(t *testing.T) {
		path := writeConfig(t, `api:
    port: -1
`)
		got := CheckLanExposure(path)
		// 负数能被 Atoi 解析，属宽容解析（不视为 ParseError），且不触发暴露。
		if got.ParseError != "" {
			t.Errorf("ParseError 应为空，实际为 %q", got.ParseError)
		}
		if got.Port != -1 {
			t.Errorf("Port 应为 -1，实际为 %d", got.Port)
		}
		if got.Exposed {
			t.Errorf("Exposed 应为 false，实际为 true")
		}
	})
}

// TestCheckLanExposureCommentAndTab 覆盖注释行、行内注释、Tab 缩进等宽容解析。
func TestCheckLanExposureCommentAndTab(t *testing.T) {
	path := writeConfig(t, "# 顶层注释\n"+
		"api:\n"+
		"\tlan_accessible: true  # 行内注释\n"+
		"\tport: 8123 # 端口\n"+
		"# 段后注释\n"+
		"log:\n"+
		"    level: info\n")
	got := CheckLanExposure(path)
	if !got.Exposed {
		t.Errorf("含注释与 Tab 缩进时 Exposed 应为 true，实际为 false")
	}
	if got.Port != 8123 {
		t.Errorf("应剥离行内注释后提取 port=8123，实际为 %d", got.Port)
	}
	if got.ParseError != "" {
		t.Errorf("ParseError 应为空，实际为 %q", got.ParseError)
	}
}
