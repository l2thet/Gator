package main

import (
	"fmt"
	"log"

	"github.com/l2thet/Gator/internal/config"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	cfg.SetUser("Captum")

	cfg, err = config.Read()
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	fmt.Printf("%+v\n", cfg)
}
