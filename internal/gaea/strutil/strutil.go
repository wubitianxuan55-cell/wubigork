// Package strutil provides zero-allocation string utilities shared across
// gaea packages, avoiding fmt.Sprintf in hot paths.
package strutil

// Itoa converts an integer to its decimal string representation without
// allocating through fmt. Negative values and zero both return "0".
func Itoa(n int) string {
	if n <= 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
