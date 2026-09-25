package pii

import "testing"

func TestDetect(t *testing.T) {
	if Detect("email", "text") != "email" {
		t.Fatal("email not detected")
	}
	if Detect("customer_id", "int") != "" {
		t.Fatal("unexpected PII")
	}
}
