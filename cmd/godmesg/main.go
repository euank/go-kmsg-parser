/*
Copyright 2016 Euan Kemp

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/euank/go-kmsg-parser/v3/kmsgparser"
)

func main() {
	tail := flag.Bool("t", false, "start at the tail of kmsg")
	follow := flag.Bool("w", true, "follow kmsg")
	flag.Parse()

	var opts []kmsgparser.Option
	if !*follow {
		opts = append(opts, kmsgparser.WithNoFollow())
	}

	parser, err := kmsgparser.NewParser(opts...)
	if err != nil {
		log.Fatalf("unable to create parser: %v", err)
	}
	defer parser.Close()

	// proper signal handling to demo closing the parser

	if *tail {
		err := parser.SeekEnd()
		if err != nil {
			log.Fatalf("could not tail: %v", err)
		}
	}
	msgs := make(chan kmsgparser.Message)

	ctrlC := make(chan os.Signal, 1)
	signal.Notify(ctrlC, os.Interrupt)
	go func() {
		for range ctrlC {
			fmt.Fprintln(os.Stderr, "Closing parser")
			parser.Close()
		}
	}()

	parseErr := make(chan error, 1)
	go func() {
		parseErr <- parser.Parse(msgs)
	}()
	for msg := range msgs {
		fmt.Printf("(%d) - %s: %s", msg.SequenceNumber, msg.Timestamp.Format(time.RFC3339Nano), msg.Message)
	}
	if err := <-parseErr; err != nil {
		fmt.Fprintf(os.Stderr, "parse exited with error: %v\n", err)
	}
}
