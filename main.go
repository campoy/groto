package main

import (
	"fmt"
	"log"
	"os"

	"github.com/campoy/groto/scanner"
)

func main() {
	s := scanner.NewScanner(os.Stdin)
	for {
		switch tok := s.Scan().(type) {
		case scanner.EOF:
			return
		case scanner.Error:
			log.Fatal(tok)
		default:
			fmt.Printf("%T(%v)\n", tok, tok)
		}
	}
}
