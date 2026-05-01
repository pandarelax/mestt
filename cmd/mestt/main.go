package main

import (
	"context"
	"log"
	"os"

	"pandarelax/mestt/internal/cli"
)

func main() {
	if err := cli.Run(context.Background(), os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
