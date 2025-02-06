package utils

func ValidateSQL(query string) bool {
	// Basic validation: prevent dangerous queries
	blacklist := []string{"DROP", "DELETE", "UPDATE", "ALTER", "TRUNCATE"}
	for _, word := range blacklist {
		if containsIgnoreCase(query, word) {
			return false
		}
	}
	return true
}

func containsIgnoreCase(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || containsIgnoreCase(str[1:], substr))
}