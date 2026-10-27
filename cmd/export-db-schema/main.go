package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/RomaticDOG/fund/internal/database"
)

func main() {
	output := flag.String("output", "", "SQL output file (default: stdout)")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(1)
	}
	if *output == "" {
		if err := database.WriteFreshSchema(os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	file, err := os.Create(*output)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	writeErr := database.WriteFreshSchema(file)
	closeErr := file.Close()
	if writeErr != nil {
		fmt.Fprintln(os.Stderr, writeErr)
		os.Exit(1)
	}
	if closeErr != nil {
		fmt.Fprintln(os.Stderr, closeErr)
		os.Exit(1)
	}
}
