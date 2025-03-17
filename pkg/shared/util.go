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
