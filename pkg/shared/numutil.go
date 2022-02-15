package shared

// RoundUp rounds up number to the nearest multiple. number and multiple
// are expected to be positive
func RoundUp(number int, multiple int) int {
	remainder := number % multiple
	if remainder == 0 {
		return number
	}

	return number + (multiple - remainder)
}

// RoundDown rounds up number to the nearest multiple. number and multiple
// are expected to be positive
func RoundDown(number int, multiple int) int {
	return number - (number % multiple)
}
