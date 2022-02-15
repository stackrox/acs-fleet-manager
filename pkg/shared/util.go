package shared

// checks if slice of strings Contains given string
func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func SafeString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}
