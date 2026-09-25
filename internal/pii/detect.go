package pii

import "strings"

func Detect(column string, dataType string) string {
	n := strings.ToLower(column)
	rules := map[string][]string{"email": {"email", "e_mail"}, "phone": {"phone", "mobile", "telephone"}, "person_name": {"first_name", "last_name", "full_name", "name"}, "date_of_birth": {"dob", "birth_date", "date_of_birth"}, "address": {"address", "street", "city", "postal_code", "zip"}, "government_id": {"ssn", "aadhaar", "pan", "passport", "national_id"}, "credit_card": {"card_number", "credit_card"}}
	for label, names := range rules {
		for _, x := range names {
			if n == x || strings.Contains(n, x) {
				return label
			}
		}
	}
	return ""
}
