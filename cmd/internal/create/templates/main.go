package main

import (
	. "github.com/gotray/go-python"
	"github.com/gotray/got"
)

func main() {
	got.SetEnv()

	Initialize()
	defer Finalize()
	println("Hello, got project!")
}
