package main

import (
	"fmt"
	"os"

	"github.com/campoy/groto/groto"
)

func main() {
	f, err := groto.Parse(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	fmt.Println(f)
}
