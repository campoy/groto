package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/campoy/groto/parser"
)

func main() {
	proto, err := parser.Parse(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	b, _ := json.MarshalIndent(proto, "", "\t")
	fmt.Printf("%s\n", b)
}
