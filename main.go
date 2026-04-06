package main

import (
	"fmt"
	"os"

	"github.com/nutworker/ide/internal/app"
)

func main() {
	// Create and run the application
	application, err := app.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing IDE: %v\n", err)
		os.Exit(1)
	}

	if err := application.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running IDE: %v\n", err)
		os.Exit(1)
	}
}
