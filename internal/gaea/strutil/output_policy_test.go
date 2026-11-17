package strutil

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPolicyBudgetRoundTrip(t *testing.T) {
	bytesPolicy := BytesPolicy(1000)
	tokensPolicy := TokensPolicy(250)
	if bytesPolicy.ByteBudget() != tokensPolicy.ByteBudget() {
		t.Fatalf("round trip mismatch: bytes=%d tokens=%d", bytesPolicy.ByteBudget(), tokensPolicy.ByteBudget())
	}
	if got := tokensPolicy.TokenBudget(); got != 250 {
		t.Fatalf("TokenBudget()=%d, want 250", got)
	}
}

func TestTruncateMiddleCharsKeepsMarkerOutsideBudget(t *testing.T) {
	input := strings.Repeat("a", 1001)
	got := TruncateMiddleChars(input, BytesPolicy(1000), "…")
	if !got.Truncated {
		t.Fatal("expected truncation")
	}
	if len(got.Text) != 1003 {
		t.Fatalf("result bytes=%d, want 1000 retained + 3-byte marker", len(got.Text))
	}
	if got.RetainedBytes != 1000 {
		t.Fatalf("retained bytes=%d, want 1000", got.RetainedBytes)
	}
}

func TestTruncateMiddlePreservesHeadTailAndUTF8(t *testing.T) {
	input := "首" + strings.Repeat("中", 300) + "尾🎉"
	got := TruncateMiddleChars(input, BytesPolicy(180), " … ")
	if !got.Truncated {
		t.Fatal("expected truncation")
	}
	if !utf8.ValidString(got.Text) || strings.ContainsRune(got.Text, utf8.RuneError) {
		t.Fatalf("result is not valid UTF-8: %q", got.Text)
	}
	if !strings.HasPrefix(got.Text, "首") || !strings.HasSuffix(got.Text, "尾🎉") {
		t.Fatalf("head/tail not preserved: %q", got.Text)
	}
	if got.RetainedBytes > 180 {
		t.Fatalf("retained bytes=%d over budget", got.RetainedBytes)
	}
}

func TestTruncateMiddleWithMarkerStaysWithinBudget(t *testing.T) {
	input := strings.Repeat("a", 1001)
	got := TruncateMiddleWithMarker(input, BytesPolicy(1000), "[cut]")
	if len(got.Text) > 1000 {
		t.Fatalf("result bytes=%d over budget 1000", len(got.Text))
	}
	if !strings.Contains(got.Text, "[cut]") {
		t.Fatalf("marker missing: %q", got.Text)
	}

	tiny := TruncateMiddleWithMarker(input, BytesPolicy(3), "[cut]")
	if tiny.Text != "aaa" {
		t.Fatalf("marker-too-large fallback=%q, want longest prefix aaa", tiny.Text)
	}
}

func TestSerializationAllowanceIsIdempotent(t *testing.T) {
	base := BytesPolicy(1000)
	once := base.WithSerializationAllowance()
	twice := once.WithSerializationAllowance()
	if once.Limit != 1200 {
		t.Fatalf("allowance limit=%d, want 1200", once.Limit)
	}
	if twice.Limit != once.Limit {
		t.Fatalf("allowance applied twice: %d -> %d", once.Limit, twice.Limit)
	}
}

func TestSharedOutputBudgetIsLoaded(t *testing.T) {
	budget := SharedOutputBudget()
	if budget.SchemaVersion != 1 {
		t.Fatalf("schema=%d", budget.SchemaVersion)
	}
	if budget.Transport.ByteLimit <= 0 || budget.Display.MaxRenderedRows <= 0 {
		t.Fatalf("invalid budget: %+v", budget)
	}
}
