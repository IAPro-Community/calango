// Derived from vstorm-co/agenticos (Apache-2.0) at a4b0b3c: backend/app/core/audit.py
// (chain_hash) and backend/app/services/audit.py (verify_chain). Modified.

// Package audit computes and verifies the tamper-evident chain of the audit log.
//
// An entry's hash is SHA-256 over the previous entry's hash and the entry's own fields,
// serialized canonically (sorted keys). The row id and the sequence number are left out:
// they are handles, not part of what happened. Verification walks the entries in sequence
// order and reports the first break. A checkpoint of the head catches a tail that was
// truncated, which a walk of the survivors cannot see.
package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Fields are the attested fields of one entry, in the form stored in the database.
type Fields struct {
	At        string // UTC, usage time layout
	ActorKind string
	ActorID   string // empty when the entry has no actor
	Action    string
	Subject   string
	FactsJSON string
}

// Record is one stored entry together with its chain columns.
type Record struct {
	Seq       int64
	ID        string
	Fields    Fields
	PrevHash  string // empty for the first entry
	EntryHash string // empty for rows written before the chain existed
}

// Checkpoint is the high-water mark advanced by every append.
type Checkpoint struct {
	MaxSeq     int64
	EntryCount int64
	HeadHash   string
}

// Break says where and why a chain stopped verifying. Seq and ID are zero for breaks
// about the chain as a whole (a missing or truncated tail).
type Break struct {
	Seq    int64
	ID     string
	Reason string
}

// Verification is the result of walking one chain.
type Verification struct {
	EntriesChecked int
	FirstBreak     *Break
}

// Hash returns the chain hash of an entry linked to prev ("" for the first entry).
func Hash(prev string, f Fields) (string, error) {
	facts, err := canonicalFacts(f.FactsJSON)
	if err != nil {
		return "", err
	}
	var prevHash, actorID any
	if prev != "" {
		prevHash = prev
	}
	if f.ActorID != "" {
		actorID = f.ActorID
	}
	canonical, err := json.Marshal(map[string]any{
		"prev_hash":  prevHash,
		"actor_kind": f.ActorKind,
		"actor_id":   actorID,
		"action":     f.Action,
		"subject":    f.Subject,
		"facts":      facts,
		"at":         f.At,
	})
	if err != nil {
		return "", fmt.Errorf("canonicalize audit entry: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

// canonicalFacts decodes the stored facts keeping numbers as text, so that re-encoding
// them yields the same bytes that were hashed at write time.
func canonicalFacts(raw string) (any, error) {
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("facts are not valid JSON: %w", err)
	}
	if err := dec.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
		return nil, errors.New("facts are not valid JSON: trailing data")
	}
	return v, nil
}

// Verify walks records in sequence order, checking each link and hash, then compares the
// head with the checkpoint. A nil checkpoint means none has been written.
func Verify(records []Record, cp *Checkpoint) Verification {
	var out Verification
	prev := ""
	for _, r := range records {
		if r.EntryHash == "" {
			return breakAt(out, r, "entry predates the chain")
		}
		if r.PrevHash != prev {
			return breakAt(out, r, "prev_hash does not link to the previous entry")
		}
		want, err := Hash(prev, r.Fields)
		if err != nil {
			return breakAt(out, r, err.Error())
		}
		if r.EntryHash != want {
			return breakAt(out, r, "entry_hash does not match the entry's contents")
		}
		prev = r.EntryHash
		out.EntriesChecked++
	}
	if reason := checkpointBreak(records, cp); reason != "" {
		out.FirstBreak = &Break{Reason: reason}
	}
	return out
}

func breakAt(v Verification, r Record, reason string) Verification {
	v.FirstBreak = &Break{Seq: r.Seq, ID: r.ID, Reason: reason}
	return v
}

func checkpointBreak(records []Record, cp *Checkpoint) string {
	if cp == nil {
		if len(records) > 0 {
			return "checkpoint is missing for a non-empty chain"
		}
		return ""
	}
	if len(records) == 0 {
		return "chain is missing; the checkpoint records entries"
	}
	last := records[len(records)-1]
	switch {
	case last.Seq < cp.MaxSeq:
		return "chain ends before the checkpoint: tail truncated"
	case int64(len(records)) < cp.EntryCount:
		return "fewer entries than the checkpoint records"
	case last.Seq == cp.MaxSeq && last.EntryHash != cp.HeadHash:
		return "head hash differs from the checkpoint"
	}
	return ""
}
