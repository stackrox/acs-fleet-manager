package shared

// Contains checks if slice of strings Contains given string
func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

// SafeString ...
func SafeString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// SafeInt64 ...
func SafeInt64(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}
