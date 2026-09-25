package pii

import (
	"testing"

	"github.com/example/go-data-profiler/internal/domain"
)

func TestDetect(t *testing.T) {
	if Detect("email", "text") != "email" {
		t.Fatal("email not detected")
	}
	if Detect("customer_id", "int") != "" {
		t.Fatal("unexpected PII")
	}
}

func TestDetectProfileUsesValueEvidence(t *testing.T) {
	values := []domain.ValueCount{
		{Value: "alice@example.com", Count: 4},
		{Value: "bob@example.com", Count: 3},
	}
	d := DetectProfile("contact", "text", values)
	if d.Label != "email" {
		t.Fatalf("expected email, got %q", d.Label)
	}
	if d.Confidence < 0.9 {
		t.Fatalf("expected high confidence, got %.2f", d.Confidence)
	}
}

func TestDetectProfileRejectsInvalidEmailEvidence(t *testing.T) {
	values := []domain.ValueCount{
		{Value: "c@test.com", Count: 1},
		{Value: "d@test.com", Count: 1},
		{Value: "b@. test.com", Count: 1},
		{Value: "a@test.com", Count: 1},
	}
	d := DetectProfile("email", "text", values)
	if d.Label != "email" {
		t.Fatalf("expected header-based email classification, got %q", d.Label)
	}
	if d.Confidence >= 0.95 {
		t.Fatalf("expected reduced confidence when one value is invalid, got %.2f", d.Confidence)
	}
	if matchesLabel("email", "b@. test.com") {
		t.Fatal("invalid email should not match")
	}
}

func TestDetectProfileRejectsInvalidIPv4(t *testing.T) {
	if matchesLabel("ip_address", "999.1.1.1") {
		t.Fatal("invalid IPv4 should not match")
	}
	if !matchesLabel("ip_address", "192.168.1.10") {
		t.Fatal("valid IPv4 should match")
	}
}

func TestDetectProfileCreditCardUsesLuhn(t *testing.T) {
	values := []domain.ValueCount{{Value: "4111 1111 1111 1111", Count: 2}}
	d := DetectProfile("payment_value", "text", values)
	if d.Label != "credit_card" {
		t.Fatalf("expected credit_card, got %q", d.Label)
	}
}
