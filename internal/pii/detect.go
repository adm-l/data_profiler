package pii

import (
	"regexp"
	"strings"

	"github.com/example/go-data-profiler/internal/domain"
)

type Detection struct {
	Label      string
	Confidence float64
}

var (
	emailRE       = regexp.MustCompile(`(?i)^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	phoneRE       = regexp.MustCompile(`^\+?[0-9][0-9 .()-]{7,20}$`)
	uuidRE        = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	ipv4RE        = regexp.MustCompile(`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`)
	panRE         = regexp.MustCompile(`^[A-Z]{5}[0-9]{4}[A-Z]$`)
	aadhaarRE     = regexp.MustCompile(`^[0-9]{12}$`)
	cardRE        = regexp.MustCompile(`^[0-9 -]{13,23}$`)
)

var headerRules = map[string][]string{
	"email":       {"email", "e_mail", "email_address"},
	"phone":       {"phone", "mobile", "telephone", "phone_number"},
	"person_name": {"first_name", "last_name", "full_name", "person_name", "name"},
	"date_of_birth": {"dob", "birth_date", "date_of_birth"},
	"address":     {"address", "street", "city", "postal_code", "zip"},
	"government_id": {"ssn", "aadhaar", "aadhar", "pan", "passport", "national_id", "government_id"},
	"credit_card": {"card_number", "credit_card", "credit_card_number"},
	"ip_address":  {"ip", "ip_address", "ipv4", "ipv6"},
	"uuid":        {"uuid"},
}

func Detect(column string, dataType string) string {
	d := DetectProfile(column, dataType, nil)
	return d.Label
}

func DetectProfile(column string, dataType string, values []domain.ValueCount) Detection {
	header := normalize(column)

	// Column names are useful evidence, but not enough on their own for
	// ambiguous names such as "name" or "id".
	for label, names := range headerRules {
		for _, rule := range names {
			if header == rule || strings.Contains(header, rule) {
				confidence := 0.75
				if label == "person_name" && header == "name" {
					confidence = 0.65
				}
				if valuesMatch(label, values) {
					confidence = 0.95
				}
				return Detection{Label: label, Confidence: confidence}
			}
		}
	}

	// Pattern evidence catches columns whose names are generic or misleading.
	for _, label := range []string{"email", "phone", "credit_card", "government_id", "uuid", "ip_address"} {
		if valuesMatch(label, values) {
			return Detection{Label: label, Confidence: 0.90}
		}
	}
	return Detection{}
}

func valuesMatch(label string, values []domain.ValueCount) bool {
	checked := 0
	matches := 0
	for _, v := range values {
		if strings.TrimSpace(v.Value) == "" {
			continue
		}
		checked++
		if matchesLabel(label, strings.TrimSpace(v.Value)) {
			matches++
		}
	}
	return checked > 0 && matches >= 1 && float64(matches)/float64(checked) >= 0.5
}

func matchesLabel(label, value string) bool {
	switch label {
	case "email":
		return emailRE.MatchString(value)
	case "phone":
		digits := digitsOnly(value)
		return phoneRE.MatchString(value) && len(digits) >= 10
	case "credit_card":
		digits := digitsOnly(value)
		return len(digits) >= 13 && len(digits) <= 19 && luhn(digits)
	case "government_id":
		upper := strings.ToUpper(value)
		return panRE.MatchString(upper) || aadhaarRE.MatchString(digitsOnly(value))
	case "uuid":
		return uuidRE.MatchString(value)
	case "ip_address":
		return ipv4RE.MatchString(value)
	default:
		return false
	}
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func luhn(s string) bool {
	if len(s) < 13 || len(s) > 19 {
		return false
	}
	sum, alternate := 0, false
	for i := len(s) - 1; i >= 0; i-- {
		n := int(s[i] - '0')
		if alternate {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alternate = !alternate
	}
	return sum%10 == 0
}

func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return s
}
