//go:build ignore
// +build ignore

// Tutorial playground file for learning Go basics. The build tag above keeps it
// out of normal builds (`go build` / `go test`) so it won't clash with the real
// service entrypoint in main.go.
// this is for me to understand the working of go
package main

import (
	"fmt"
	"log"
) // core package in the stdlib for formatted I/O operations(stdio like)

// core package in the stdlib for logging

// import "net" // core package in the stdlib for networking

// import "os" // core package in the stdlib for OS functionality

// import "google.golang.org/grpc" // external package for gRPC functionality

// import stationpb "lastmile/gen/go/api" // local package for station protobuf definitions

// import "lastmile/internal/station" // local package for station service implementation

func main() {
	fmt.Println("Hello, Choose a service \n 1. Play \n 2. exit") // Print Hello, World! to the console
	// fmt.Scanln() // Wait for user input before exiting
	var choice = 0
	fmt.Scanln(&choice)
	// if choice == 1 {
	// 	log.Println("Starting Station Service...")
	// } else if choice == 2 {
	// 	fmt.Println("Exiting...")
	// }
	// else{
	// 	log.Print("Invalid Choice")
	// }
	//
	/*
		 	* Go's fmt.Println() is a variadic function,
			* meaning it is designed to accept a variable number of
			* arguments of any type directly.  */

	var k = "COPYRIGHT"
	if choice == 1 {
		log.Println("Starting Station Service...")
		fmt.Println("This is a sample log message with key:", k)
	} else if choice == 2 {
		fmt.Println("Exiting...")
	} else {
		// Standard log package doesn't have an Error method.
		// Use log.Println for a standard log message to Stderr,
		// or log.Fatal to print and exit the program immediately.
		log.Println("Invalid Choice")
	}
}
