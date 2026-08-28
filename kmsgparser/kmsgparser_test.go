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

package kmsgparser

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

// Logger that errors on warnings and errors
type warningAndErrorTestLogger struct {
	t *testing.T
}

func (warningAndErrorTestLogger) Infof(string, ...interface{}) {}
func (w warningAndErrorTestLogger) Warningf(s string, i ...interface{}) {
	w.t.Errorf(s, i...)
}
func (w warningAndErrorTestLogger) Errorf(s string, i ...interface{}) {
	w.t.Errorf(s, i...)
}

func TestParseMessage(t *testing.T) {
	bootTime := time.Unix(0xb100, 0x5ea1).Round(time.Microsecond)
	p := parser{
		log:      warningAndErrorTestLogger{t: t},
		bootTime: bootTime,
	}
	msg, err := p.parseMessage("6,2565,102258085667,-;docker0: port 2(vethc1bb733) entered blocking state")
	if err != nil {
		t.Fatalf("error parsing: %v", err)
	}

	assertEqual(t, msg.Message, "docker0: port 2(vethc1bb733) entered blocking state")
	assertEqual(t, msg.Priority, 6)
	assertEqual(t, msg.SequenceNumber, 2565)
	assertEqual(t, msg.Timestamp, bootTime.Add(102258085667*time.Microsecond))
}

func TestParseMessageFromSample(t *testing.T) {
	testFile, err := os.Open("test_data/sample1.kmsg")
	if err != nil {
		t.Fatalf("open sample data: %v", err)
	}
	t.Cleanup(func() {
		if err := testFile.Close(); err != nil {
			t.Errorf("close sample data: %v", err)
		}
	})

	p := parser{bootTime: time.Unix(0, 0)}
	wantSequences := []int{1804, 1805, 2651}
	scanner := bufio.NewScanner(testFile)
	var gotSequences []int
	for scanner.Scan() {
		msg, err := p.parseMessage(scanner.Text())
		if err != nil {
			t.Fatalf("parse sample message: %v", err)
		}
		gotSequences = append(gotSequences, msg.SequenceNumber)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read sample data: %v", err)
	}

	if len(gotSequences) != len(wantSequences) {
		t.Fatalf("parsed %d messages, want %d", len(gotSequences), len(wantSequences))
	}
	for i := range wantSequences {
		assertEqual(t, gotSequences[i], wantSequences[i])
	}
}

func TestParseMessageRejectsMalformedInput(t *testing.T) {
	tests := map[string]string{
		"missing separator": "6,2565,102258085667,-",
		"missing metadata":  "6,2565;message",
		"invalid priority":  "invalid,2565,102258085667,-;message",
		"invalid sequence":  "6,invalid,102258085667,-;message",
		"invalid timestamp": "6,2565,invalid,-;message",
	}

	p := parser{}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := p.parseMessage(input); err == nil {
				t.Fatal("expected parsing to fail")
			}
		})
	}
}

func TestParseMessageErrorIdentifiesInvalidField(t *testing.T) {
	tests := map[string]struct {
		input        string
		invalidValue string
	}{
		"sequence": {
			input:        "6,invalid-sequence,102258085667,-;message",
			invalidValue: "invalid-sequence",
		},
		"timestamp": {
			input:        "6,2565,invalid-timestamp,-;message",
			invalidValue: "invalid-timestamp",
		},
	}

	p := parser{}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := p.parseMessage(test.input)
			if err == nil {
				t.Fatal("expected parsing to fail")
			}
			if !strings.Contains(err.Error(), test.invalidValue) {
				t.Fatalf("error %q does not identify invalid value %q", err, test.invalidValue)
			}
		})
	}
}

func TestStandardLoggerWithNilLogger(t *testing.T) {
	logger := &StandardLogger{}
	logger.Warningf("warning")
	logger.Infof("information")
	logger.Errorf("error")
}

func TestStandardLoggerFormatting(t *testing.T) {
	var output bytes.Buffer
	logger := &StandardLogger{Logger: log.New(&output, "", 0)}

	logger.Warningf("warning %d", 1)
	logger.Infof("information %d", 2)
	logger.Errorf("error %d", 3)

	want := "[WARNING] warning 1\n[INFO] information 2\n[ERROR] error 3\n"
	if got := output.String(); got != want {
		t.Fatalf("unexpected log output:\n%s", got)
	}
}

func assertEqual[T comparable](t *testing.T, lhs, rhs T) {
	if lhs != rhs {
		t.Fatalf("expected %v = %v", lhs, rhs)
	}
}
