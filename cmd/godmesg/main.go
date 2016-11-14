package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/euank/go-kmsg-parser/kmsgparser"
)

func main() {
	f, err := os.Open("/dev/kmsg")
	if err != nil {
		log.Fatalf("unable to open /dev/kmsg: %v", err)
	}

	kmsg, err := kmsgparser.Parse(f)
	if err != nil {
		log.Fatalf("unable to start parser: %v", err)
	}

	for msg := range kmsg {
		fmt.Printf("(%d) - %s: %s", msg.SequenceNumber, msg.Timestamp.Format(time.RFC3339Nano), msg.Message)
	}
}
