package main

import (
	"fmt"
	"core/sample/math"
)

func main() {
	HelloWorld()
	result := math.Add(2, 3)
	fmt.Println("2 + 3 =", result)
}