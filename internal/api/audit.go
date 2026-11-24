package api

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"calango/internal/audit"
	"calango/internal/core"
)

// AuditLedger reads the audit chain for verification and export. *store.Store implements it.
type AuditLedger interface {
	VerifyAudit(ctx context.Context) (audit.Verification, error)
	ExportAudit(ctx context.Context) (audit.Export, error)
}

// auditVerification is the AuditVerification schema.
type auditVerification struct {
	OK             bool        `json:"ok"`
	EntriesChecked int         `json:"entries_checked"`
	FirstBreak     *auditBreak `json:"first_break"`
}

// auditBreak is the first_break object of AuditVerification.
type auditBreak struct {
	Seq    int64  `json:"seq"`
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

// auditExportCheckpoint is the first JSON Lines record of an export. It guards the tail:
// without it an offline check cannot tell a truncated chain from a shorter one.
type auditExportCheckpoint struct {
	Type          string `json:"type"`
	MaxSeq        int64  `json:"max_seq"`
	EntryCount    int64  `json:"entry_count"`
	HeadEntryHash string `json:"head_entry_hash"`
}

// auditExportEntry is one stored audit entry as exported. Facts carry names and
// identifiers only; secret values never reach the audit log.
type auditExportEntry struct {
	Type      string          `json:"type"`
	Seq       int64           `json:"seq"`
	ID        string          `json:"id"`
	At        string          `json:"at"`
	ActorKind string          `json:"actor_kind"`
	ActorID   string          `json:"actor_id,omitempty"`
	Action    string          `json:"action"`
	Subject   string          `json:"subject"`
	Facts     json.RawMessage `json:"facts"`
	PrevHash  string          `json:"prev_hash,omitempty"`
	EntryHash string          `json:"entry_hash"`
}

func (h *handler) registerAudit(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/audit/verify", h.need(core.PermRead, h.verifyAudit))
	mux.HandleFunc("GET /v1/audit/export", h.need(core.PermRead, h.exportAudit))
}

func (h *handler) verifyAudit(w http.ResponseWriter, r *http.Request) {
	v, err := h.audit.VerifyAudit(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := auditVerification{OK: v.FirstBreak == nil, EntriesChecked: v.EntriesChecked}
	if b := v.FirstBreak; b != nil {
		out.FirstBreak = &auditBreak{Seq: b.Seq, ID: b.ID, Reason: b.Reason}
	}
	writeJSON(w, http.StatusOK, out)
}

// exportAudit returns the whole chain as JSON Lines (default) or CSV. The body is built in
// full before the response starts, so an encoding error becomes a clean error response.
func (h *handler) exportAudit(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "jsonl"
	}
	var contentType, filename string
	switch format {
	case "jsonl":
		contentType, filename = "application/x-ndjson", "calango-audit.jsonl"
	case "csv":
		contentType, filename = "text/csv; charset=utf-8", "calango-audit.csv"
	default:
		writeServiceError(w, fmt.Errorf("%w: format must be jsonl or csv", core.ErrInvalidInput))
		return
	}

	exp, err := h.audit.ExportAudit(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var body bytes.Buffer
	if format == "csv" {
		err = writeAuditCSV(&body, exp)
	} else {
		err = writeAuditJSONL(&body, exp)
	}
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body.Bytes())
}

func writeAuditJSONL(buf *bytes.Buffer, exp audit.Export) error {
	enc := json.NewEncoder(buf)
	if cp := exp.Checkpoint; cp != nil {
		if err := enc.Encode(auditExportCheckpoint{
			Type: "checkpoint", MaxSeq: cp.MaxSeq, EntryCount: cp.EntryCount, HeadEntryHash: cp.HeadHash,
		}); err != nil {
			return fmt.Errorf("encode audit checkpoint: %w", err)
		}
	}
	for _, r := range exp.Records {
		if !json.Valid([]byte(r.Fields.FactsJSON)) {
			return fmt.Errorf("audit entry %d has facts that are not valid JSON", r.Seq)
		}
		if err := enc.Encode(auditExportEntry{
			Type: "entry", Seq: r.Seq, ID: r.ID, At: r.Fields.At,
			ActorKind: r.Fields.ActorKind, ActorID: r.Fields.ActorID,
			Action: r.Fields.Action, Subject: r.Fields.Subject,
			Facts:    json.RawMessage(r.Fields.FactsJSON),
			PrevHash: r.PrevHash, EntryHash: r.EntryHash,
		}); err != nil {
			return fmt.Errorf("encode audit entry %d: %w", r.Seq, err)
		}
	}
	return nil
}

// writeAuditCSV writes the entries only. CSV has no checkpoint line, so a tail truncation
// is detectable only from the JSON Lines export.
func writeAuditCSV(buf *bytes.Buffer, exp audit.Export) error {
	w := csv.NewWriter(buf)
	if err := w.Write([]string{"seq", "id", "at", "actor_kind", "actor_id", "action",
		"subject", "facts", "prev_hash", "entry_hash"}); err != nil {
		return fmt.Errorf("write audit csv header: %w", err)
	}
	for _, r := range exp.Records {
		if err := w.Write([]string{strconv.FormatInt(r.Seq, 10), r.ID, r.Fields.At,
			r.Fields.ActorKind, r.Fields.ActorID, r.Fields.Action, r.Fields.Subject,
			r.Fields.FactsJSON, r.PrevHash, r.EntryHash}); err != nil {
			return fmt.Errorf("write audit csv entry %d: %w", r.Seq, err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("flush audit csv: %w", err)
	}
	return nil
}
