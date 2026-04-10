package main

import (
	"fmt"
	"strings"
)

// TestStruct is a test struct
type TestStruct struct {
	Name  string
	Count int
}

// sayHello prints a greeting
func sayHello(name string) {
	message := "Hello, " + name + "!"
	fmt.Println(message)

	// This is a comment
	numbers := []int{1, 2, 3, 4, 5}
	total := 0

	for _, num := range numbers {
		total += num
	}

	if total > 10 {
		fmt.Printf("Total is %d\n", total)
	}
}

func main() {
	sayHello("World")

	test := TestStruct{
		Name:  "Example",
		Count: 42,
	}

	fmt.Printf("Struct: %+v\n", test)

	// Test string literals
	multiline := `This is a
multiline string
with multiple lines`

	result := strings.ToUpper(multiline)
	fmt.Println(result)
}
