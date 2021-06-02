package main

import (
	"log"	
)

func main() {
	if err := RunProxy(); err != nil {
		log.Fatal("Run Proxy Server failed")
	}
}