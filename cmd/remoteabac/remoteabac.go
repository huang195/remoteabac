package main

import (
	"github.com/huang195/remoteabac/server"
)

func main() {
	server := server.New()
	server.Run()
}
