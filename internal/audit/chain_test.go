// Derived from vstorm-co/agenticos (Apache-2.0) at a4b0b3c: backend/app/services/audit.py
// (verify_chain scenarios). Modified.

package audit

import (
	"fmt"
	"strings"
	"testing"
)

func fields(action, facts string) Fields {
	return Fields{
		At: "2026-11-23T10:00:00.000Z", ActorKind: "user", ActorID: "owner-1",
		Action: action, Subject: "approval/1", FactsJSON: facts,
	}
}

// buildChain links every field set the way the store does and returns the records and
// the checkpoint that an append would leave behind.
func buildChain(t *testing.T, all ...Fields) ([]Record, *Checkpoint) {
	t.Helper()
	var records []Record
	prev := ""
	for i, f := range all {
		hash, err := Hash(prev, f)
		if err != nil {
			t.Fatalf("hash entry %d: %v", i, err)
		}
		records = append(records, Record{Seq: int64(i + 1), ID: fmt.Sprintf("entry-%d", i+1),
			Fields: f, PrevHash: prev, EntryHash: hash})
		prev = hash
	}
	last := records[len(records)-1]
	return records, &Checkpoint{MaxSeq: last.Seq, EntryCount: int64(len(records)), HeadHash: last.EntryHash}
}

func threeEntries(t *testing.T) ([]Record, *Checkpoint) {
	return buildChain(t,
		fields("approval.requested", `{"risk":"high"}`),
		fields("approval.decided", `{"decision":"approved"}`),
		fields("device.revoked", `{}`),
	)
}

func TestIntactChainVerifies(t *testing.T) {
	records, cp := threeEntries(t)
	got := Verify(records, cp)
	if got.FirstBreak != nil || got.EntriesChecked != 3 {
		t.Fatalf("intact chain: got %+v, want 3 entries and no break", got)
	}
}

func TestEditedEntryIsReported(t *testing.T) {
	records, cp := threeEntries(t)
	records[1].Fields.FactsJSON = `{"decision":"denied"}`
	got := Verify(records, cp)
	if got.FirstBreak == nil || got.FirstBreak.Seq != 2 ||
		got.FirstBreak.Reason != "entry_hash does not match the entry's contents" {
		t.Fatalf("edited entry: got %+v, want break at seq 2", got.FirstBreak)
	}
}

func TestDeletedMiddleEntryBreaksNextLink(t *testing.T) {
	records, cp := threeEntries(t)
	records = append(records[:1], records[2:]...)
	got := Verify(records, cp)
	if got.FirstBreak == nil || got.FirstBreak.Seq != 3 ||
		got.FirstBreak.Reason != "prev_hash does not link to the previous entry" {
		t.Fatalf("deleted middle: got %+v, want link break at seq 3", got.FirstBreak)
	}
}

func TestTruncatedTailIsCaughtByCheckpoint(t *testing.T) {
	records, cp := threeEntries(t)
	got := Verify(records[:2], cp)
	if got.FirstBreak == nil || got.FirstBreak.Reason != "chain ends before the checkpoint: tail truncated" {
		t.Fatalf("truncated tail: got %+v, want checkpoint break", got.FirstBreak)
	}
}

func TestLegacyRowWithoutHashIsReported(t *testing.T) {
	records, cp := threeEntries(t)
	records[0].EntryHash = ""
	got := Verify(records, cp)
	if got.FirstBreak == nil || got.FirstBreak.Seq != 1 || got.FirstBreak.Reason != "entry predates the chain" {
		t.Fatalf("legacy row: got %+v, want predates break at seq 1", got.FirstBreak)
	}
}

func TestHashDoesNotDependOnFactKeyOrder(t *testing.T) {
	a, err := Hash("", fields("x", `{"a":1,"b":{"y":2,"z":3}}`))
	if err != nil {
		t.Fatal(err)
	}
	b, err := Hash("", fields("x", `{"b":{"z":3,"y":2},"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("equal facts hashed differently: %s != %s", a, b)
	}
}

func TestMalformedFactsAreReported(t *testing.T) {
	_, err := Hash("", fields("x", `{"a":1} trailing`))
	if err == nil || !strings.Contains(err.Error(), "trailing data") {
		t.Fatalf("trailing data: got %v, want error", err)
	}
}

func TestEmptyChainVerifiesWithoutCheckpoint(t *testing.T) {
	got := Verify(nil, nil)
	if got.FirstBreak != nil || got.EntriesChecked != 0 {
		t.Fatalf("empty chain: got %+v, want no break", got)
	}
}
