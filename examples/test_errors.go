package main

import (
	"fmt"
)

func main() {
	// Error 1: Unused variable (go vet will catch this)
	unused := 42
	fmt.Println(unused)

	// Error 2: Undeclared variable
	fmt.Println(undeclared)

	// Error 3: Type mismatch
	var num int = "this is a string"

	// Error 4: Missing package
	result := strings.ToUpper("hello")

	// Error 5: Wrong number of arguments
	fmt.Printf("Name: %s Age: %d\n", "John")

	fmt.Println("If this prints, all errors are fixed!")
}
