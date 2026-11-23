package main

import (
	"log"
	"os"
	"os/exec"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("carregando .env: %v", err)
	}

	args := append([]string{
		"tool", "tern", "migrate",
		"--migrations", "./internal/database/migrations",
		"--config", "./internal/database/migrations/tern.conf",
	}, os.Args[1:]...)

	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
