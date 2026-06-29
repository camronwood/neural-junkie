package sample

// Multiply returns the product of a and b.
func Multiply(a, b int) int {
	return a * b // intentional bug: fails go test
}