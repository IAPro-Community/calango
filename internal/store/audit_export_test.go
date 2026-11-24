package store

import (
	"context"
	"testing"

	"calango/internal/audit"
)

func TestAuditExportCarriesChainAndCheckpointTogether(t *testing.T) {
	ctx := context.Background()
	st, err := OpenMemory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	appendThree(t, st)

	exp, err := st.ExportAudit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(exp.Records) != 3 || exp.Checkpoint == nil {
		t.Fatalf("export = %d records, checkpoint %v; want 3 and a checkpoint", len(exp.Records), exp.Checkpoint)
	}
	if exp.Checkpoint.EntryCount != 3 || exp.Checkpoint.HeadHash != exp.Records[2].EntryHash {
		t.Fatalf("checkpoint = %+v, want 3 entries headed by the last entry hash", exp.Checkpoint)
	}
	if exp.Records[0].PrevHash != "" || exp.Records[1].PrevHash != exp.Records[0].EntryHash {
		t.Fatalf("export does not link entries: %+v", exp.Records)
	}
	if got := audit.Verify(exp.Records, exp.Checkpoint); got.FirstBreak != nil {
		t.Fatalf("exported chain does not verify offline: %+v", got.FirstBreak)
	}
}

func TestAuditExportOfEmptyChainHasNoCheckpoint(t *testing.T) {
	ctx := context.Background()
	st, err := OpenMemory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	exp, err := st.ExportAudit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(exp.Records) != 0 || exp.Checkpoint != nil {
		t.Fatalf("empty export = %+v, want no records and no checkpoint", exp)
	}
}
