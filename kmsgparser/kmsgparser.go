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

// Package kmsgparser implements a parser for the Linux `/dev/kmsg` format.
// It is based on an understanding of the protocol derived from rsyslog's
// contrib module:
// https://github.com/rsyslog/rsyslog/blob/v8.22.0/contrib/imkmsg/kmsg.c
package kmsgparser

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Calculate the boot time once to figure out timestamps of messages
var bootTime time.Time

// To allow for mock/unit testing
var sysInfoFunc = syscall.Sysinfo
var timeNowFunc = time.Now

type Message struct {
	Priority       int
	SequenceNumber int
	Timestamp      time.Time
	Message        string
}

func getBootTime() (time.Time, error) {
	var sysinfo syscall.Sysinfo_t
	err := sysInfoFunc(&sysinfo)
	if err != nil {
		return time.Time{}, fmt.Errorf("could not get boot time: %v", err)
	}
	// sysinfo only has seconds
	return timeNowFunc().Add(-1 * (time.Duration(sysinfo.Uptime) * time.Second)), nil
}

// don't log by default
var log Logger = &StandardLogger{nil}

// SetLogger sets the logger that will be used to log information about the
// parser completing, or any unexpected read errors.
// If not called, no messages will be output.
func SetLogger(logger Logger) {
	log = logger
}

// Parse will read from the provided reader and provide a channel of messages
// parsed.
// If the provided reader *is not* a proper Linux kmsg device, Parse might not
// behave correctly since it relies on specific behavior of `/dev/kmsg`
//
// A goroutine is created to process the provided reader. The goroutine will
// exit when the given reader is closed.
// Closing the passed in reader will cause the goroutine to exit.
func Parse(r io.Reader) (<-chan Message, error) {
	var err error
	bootTime, err = getBootTime()
	if err != nil {
		return nil, err
	}

	output := make(chan Message, 1)
	go func() {
		defer close(output)
		msg := make([]byte, 8192)
		for {
			// Each read call gives us one full message.
			// https://www.kernel.org/doc/Documentation/ABI/testing/dev-kmsg
			n, err := r.Read(msg)
			if err != nil {
				if err == syscall.EPIPE {
					log.Warningf("short read from kmsg; skipping")
					continue
				}

				if err == io.EOF {
					log.Infof("kmsg reader closed, shutting down")
					return
				}

				log.Errorf("error reading /dev/kmsg: %v", err)
				return
			}

			msgStr := string(msg[:n])

			message, err := parseMessage(msgStr)
			if err != nil {
				log.Warningf("unable to parse kmsg message %q: %v", msgStr, err)
				continue
			}

			output <- message
		}
	}()

	return output, nil
}

func parseMessage(input string) (Message, error) {
	// Format:
	//   PRIORITY,SEQUENCE_NUM,TIMESTAMP,-;MESSAGE
	parts := strings.SplitN(input, ";", 2)
	if len(parts) != 2 {
		return Message{}, fmt.Errorf("invalid kmsg; must contain a ';'")
	}

	metadata, message := parts[0], parts[1]

	metadataParts := strings.Split(metadata, ",")
	if len(metadataParts) < 3 {
		return Message{}, fmt.Errorf("invalid kmsg: must contain at least 3 ',' separated pieces at the start")
	}

	priority, sequence, timestamp := metadataParts[0], metadataParts[1], metadataParts[2]

	prioNum, err := strconv.Atoi(priority)
	if err != nil {
		return Message{}, fmt.Errorf("could not parse %q as priority: %v", priority, err)
	}

	sequenceNum, err := strconv.Atoi(sequence)
	if err != nil {
		return Message{}, fmt.Errorf("could not parse %q as sequence number: %v", priority, err)
	}

	timestampUsFromBoot, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return Message{}, fmt.Errorf("could not parse %q as timestamp: %v", priority, err)
	}

	// Timestamp is actually offset in nanos from boot time
	// TODO cache boot time

	msgTime := bootTime.Add(time.Duration(timestampUsFromBoot) * time.Microsecond)

	return Message{
		Priority:       prioNum,
		SequenceNumber: sequenceNum,
		Timestamp:      msgTime,
		Message:        message,
	}, nil
}
