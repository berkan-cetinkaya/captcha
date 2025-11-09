package main

import (
	"fmt"

	"github.com/joho/godotenv"
)

// LoadEnv development ortamında çağrılır
func LoadEnv() {
	if err := godotenv.Load("examples/demo/.env"); err == nil {
		fmt.Println("✅ examples/demo/.env loaded (dev mode)")
	}
}
