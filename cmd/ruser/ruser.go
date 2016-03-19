package main

import (
	"log"

	"github.com/huang195/remoteabac/policy"
)

func main() {
	policy, err := policy.New()
	if err != nil {
		log.Fatalf("Received an error: %v\n", err)
	}
	policy.ProcessRequest()
}
