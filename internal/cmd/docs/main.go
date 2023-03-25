package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra/doc"
	"github.com/stoe/gh-report/cmd"
)

func main() {
	output := "./docs"

	if err := os.RemoveAll(output); err != nil {
		log.Fatal(fmt.Errorf("failed to remove existing dir: %w", err))
	}

	if err := os.MkdirAll(output, 0755); err != nil {
		log.Fatal(fmt.Errorf("failed to mkdir: %w", err))
	}

	if err := doc.GenMarkdownTree(cmd.RootCmd, output); err != nil {
		log.Fatal(fmt.Errorf("failed to generate markdown: %w", err))
	}
}
