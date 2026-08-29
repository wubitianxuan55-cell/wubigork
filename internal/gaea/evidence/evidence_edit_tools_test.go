package evidence

import (
	"encoding/json"
	"testing"
)

// TestIsWriterToolCoversEditTools guards the S0.6 name-list alignment: the
// new edit tools must classify as writers (receipts land as writes, not
// reads) and grep must stay a reader.
func TestIsWriterToolCoversEditTools(t *testing.T) {
	writers := []string{"write_file", "edit_file", "multi_edit", "edit_lines", "move_file", "delete_range", "delete_symbol"}
	for _, name := range writers {
		if !isWriterTool(name) {
			t.Errorf("isWriterTool(%q) = false, want true", name)
		}
	}
	readers := []string{"read_file", "ls", "grep"}
	for _, name := range readers {
		if isWriterTool(name) {
			t.Errorf("isWriterTool(%q) = true, want false", name)
		}
		if !isReaderTool(name) {
			t.Errorf("isReaderTool(%q) = false, want true", name)
		}
	}
}

func TestReceiptFromToolCallEditTools(t *testing.T) {
	cases := []struct {
		name     string
		tool     string
		args     string
		wantRead bool
	}{
		{"edit_lines receipt is a write", "edit_lines",
			`{"path":"a.txt","start_line":1,"end_line":1,"new_content":"x"}`, false},
		{"move_file receipt is a write", "move_file",
			`{"source":"a.txt","destination":"b.txt"}`, false},
		{"grep receipt is a read", "grep", `{"pattern":"x","path":"a.txt"}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := ReceiptFromToolCall(tc.tool, json.RawMessage(tc.args), true, false)
			if !r.Success {
				t.Error("receipt should record success")
			}
			if tc.wantRead {
				if !r.Read || r.Write {
					t.Errorf("%s: Read=%v Write=%v, want read-only", tc.tool, r.Read, r.Write)
					return
				}
			} else {
				if !r.Write || r.Read {
					t.Errorf("%s: Read=%v Write=%v, want write-only", tc.tool, r.Read, r.Write)
				}
			}
		})
	}
}
