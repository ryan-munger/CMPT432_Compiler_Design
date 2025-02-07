package main

import (
	"flag"
	"gopiler/internal"
	"gopiler/web"
)

func main() {
	expose := flag.Bool("e", false, "Bool; Expose site to internet (using your IP)")
	flag.Parse()

	internal.SetWebMode(true)
	go web.StartServer(*expose) // Start the web server in a goroutine
	select {}                   // Keep the main function running
}
