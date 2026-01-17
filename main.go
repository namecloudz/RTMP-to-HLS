package main

import (
	"runtime"

	"rtmp_server/gui"
)

func main() {
	// Use all available CPU cores for multi-threading
	runtime.GOMAXPROCS(runtime.NumCPU())

	gui.Main()
}
