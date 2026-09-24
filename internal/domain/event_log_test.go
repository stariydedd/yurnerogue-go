package domain

import (
	"fmt"
	"testing"
)

func TestEventLogRetainsEveryMessageAndBoundsHistory(t *testing.T) {
	s := NewSessionSeed(21)
	for i := 0; i < 40; i++ {
		s.SetMessage(fmt.Sprintf("event %d", i))
	}
	if len(s.EventLog) != 32 || s.EventLog[0] != "event 8" || s.EventLog[31] != "event 39" {
		t.Fatalf("unexpected history: %v", s.EventLog)
	}
	s.SetMessage("")
	if s.Message != "" || len(s.EventLog) != 32 {
		t.Fatal("clearing the current message must preserve history")
	}
	s.SetMessage("repeated")
	s.SetMessage("repeated")
	if s.EventLog[30] != "repeated" || s.EventLog[31] != "repeated" {
		t.Fatal("distinct repeated events must not disappear")
	}
	if len(NewSessionSeed(21).EventLog) != 0 {
		t.Fatal("history leaked into a new run")
	}
}
