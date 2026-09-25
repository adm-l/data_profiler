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

func TestDetectProfileCreditCardUsesLuhn(t *testing.T) {
	values := []domain.ValueCount{{Value: "4111 1111 1111 1111", Count: 2}}
	d := DetectProfile("payment_value", "text", values)
	if d.Label != "credit_card" {
		t.Fatalf("expected credit_card, got %q", d.Label)
	}
}
