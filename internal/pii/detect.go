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
	emailRE   = regexp.MustCompile(`(?i)^[^@\s]+@[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)+$`)
	phoneRE   = regexp.MustCompile(`^\+?[0-9][0-9 .()-]{7,20}$`)
	uuidRE    = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	ipv4RE    = regexp.MustCompile(`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`)
	panRE     = regexp.MustCompile(`^[A-Z]{5}[0-9]{4}[A-Z]$`)
	aadhaarRE = regexp.MustCompile(`^[0-9]{12}$`)
)

var headerRules = map[string][]string{
	"email":         {"email", "e_mail", "email_address"},
	"phone":         {"phone", "mobile", "telephone", "phone_number"},
	"person_name":   {"first_name", "last_name", "full_name", "person_name", "name"},
	"date_of_birth": {"dob", "birth_date", "date_of_birth"},
	"address":       {"address", "street", "city", "postal_code", "zip"},
	"government_id": {"ssn", "aadhaar", "aadhar", "pan", "passport", "national_id", "government_id"},
	"credit_card":   {"card_number", "credit_card", "credit_card_number"},
	"ip_address":    {"ip", "ip_address", "ipv4", "ipv6"},
	"uuid":          {"uuid"},
}

func Detect(column string, dataType string) string {
	d := DetectProfile(column, dataType, nil)
	return d.Label
}

func DetectProfile(column string, dataType string, values []domain.ValueCount) Detection {
	header := normalize(column)

	for label, names := range headerRules {
		for _, rule := range names {
			if header == rule || strings.Contains(header, rule) {
				confidence := 0.75
				if label == "person_name" && header == "name" {
					confidence = 0.65
				}

				ratio := valueMatchRatio(label, values)
				switch {
				case ratio >= 0.80:
					confidence = 0.95
				case ratio >= 0.50:
					confidence = 0.85
				}
				return Detection{Label: label, Confidence: confidence}
			}
		}
	}

	for _, label := range []string{"email", "phone", "credit_card", "government_id", "uuid", "ip_address"} {
		ratio := valueMatchRatio(label, values)
		if ratio >= 0.80 {
			return Detection{Label: label, Confidence: 0.90}
		}
		if ratio >= 0.50 {
			return Detection{Label: label, Confidence: 0.80}
		}
	}

	return Detection{}
}

func valueMatchRatio(label string, values []domain.ValueCount) float64 {
	var checked, matches int64
	for _, v := range values {
		if strings.TrimSpace(v.Value) == "" {
			continue
		}
		checked += v.Count
		if matchesLabel(label, strings.TrimSpace(v.Value)) {
			matches += v.Count
		}
	}
	if checked == 0 {
		return 0
	}
	return float64(matches) / float64(checked)
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
		return validIPv4(value)
	default:
		return false
	}
}

func validIPv4(value string) bool {
	if !ipv4RE.MatchString(value) {
		return false
	}
	for _, part := range strings.Split(value, ".") {
		n := 0
		for _, r := range part {
			n = n*10 + int(r-'0')
		}
		if n > 255 {
			return false
		}
	}
	return true
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
