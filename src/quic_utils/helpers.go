package quic_utils

import (
	"log"
	"fmt"
)

// if error is not success, exit with message
func Check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// if in debug mode, print the error
func Logf(format string, args ... interface{}) {
	if debug {
		fmt.Printf("[log] "+format+"\n", args...)
	}
}