package secure

import "testing"

func TestRoundTrip(t *testing.T) {
	orig := "sk-test-1234567890"
	enc, err := EncryptString(orig)
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	dec, err := DecryptString(enc)
	if err != nil {
		t.Fatalf("DecryptString: %v", err)
	}
	if dec != orig {
		t.Fatalf("round trip = %q, want %q", dec, orig)
	}
}

func TestLegacyPlaintextPassThrough(t *testing.T) {
	plain := "sk-legacy-plain"
	dec, err := DecryptString(plain)
	if err != nil {
		t.Fatalf("DecryptString: %v", err)
	}
	if dec != plain {
		t.Fatalf("dec = %q, want %q", dec, plain)
	}
}

func TestEmpty(t *testing.T) {
	enc, err := EncryptString("")
	if err != nil || enc != "" {
		t.Fatalf("EncryptString(\"\") = %q, %v", enc, err)
	}
	dec, err := DecryptString("")
	if err != nil || dec != "" {
		t.Fatalf("DecryptString(\"\") = %q, %v", dec, err)
	}
}
