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

func assertEqual[T comparable](t *testing.T, lhs, rhs T) {
	if lhs != rhs {
		t.Fatalf("expected %v = %v", lhs, rhs)
	}
}
