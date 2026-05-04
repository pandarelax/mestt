package main

import (
	"log"

	"pandarelax/mestt/internal/gui"
)

func main() {
	if err := gui.RunFyne(); err != nil {
		log.Fatal(err)
	}
}
