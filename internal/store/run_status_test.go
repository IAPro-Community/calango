package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"calango/internal/core"
)

func TestWaitingRunKeepsItsSlotAndInterruptEndsIt(t *testing.T) {
	ctx := context.Background()
	st, err := OpenMemory(ctx)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	runID := approvalFixture(t, st)
	convID := strings.Repeat("c", 26)

	slots := func() int {
		t.Helper()
		n, err := st.CountRunningRuns(ctx, convID)
		if err != nil {
			t.Fatalf("count slots: %v", err)
		}
		return n
	}
	if n := slots(); n != 1 {
		t.Fatalf("slots = %d, want 1 for the running run", n)
	}
	if moved, err := st.SetRunStatus(ctx, runID, core.RunRunning, core.RunWaitingApproval); err != nil || !moved {
		t.Fatalf("running to waiting: moved=%v err=%v", moved, err)
	}
	if n := slots(); n != 1 {
		t.Fatalf("waiting run gave up its slot: count = %d, want 1", n)
	}
	if moved, err := st.SetRunStatus(ctx, runID, core.RunRunning, core.RunWaitingApproval); err != nil || moved {
		t.Fatalf("repeated move from running: moved=%v err=%v; want no change", moved, err)
	}
	if _, err := st.SetRunStatus(ctx, runID, core.RunWaitingApproval, core.RunStatus("failed")); err == nil {
		t.Fatal("waiting to failed was allowed; only running and waiting_approval may change here")
	}
	if moved, err := st.SetRunStatus(ctx, runID, core.RunWaitingApproval, core.RunRunning); err != nil || !moved {
		t.Fatalf("waiting to running: moved=%v err=%v", moved, err)
	}

	if moved, err := st.SetRunStatus(ctx, runID, core.RunRunning, core.RunWaitingApproval); err != nil || !moved {
		t.Fatalf("running to waiting again: moved=%v err=%v", moved, err)
	}
	n, err := st.InterruptRunningRuns(ctx, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || n != 1 {
		t.Fatalf("interrupt = %d, %v; want 1 waiting run ended", n, err)
	}
	run, err := st.GetRun(ctx, runID)
	if err != nil {
		t.Fatalf("read run: %v", err)
	}
	if run.Status != core.RunStatus("failed") {
		t.Fatalf("interrupted waiting run status = %s, want failed", run.Status)
	}
	if n := slots(); n != 0 {
		t.Fatalf("slots after interrupt = %d, want 0", n)
	}
}

func TestAgentSlotQueryAndClaimSpanConversations(t *testing.T) {
	ctx := context.Background()
	st, err := OpenMemory(ctx)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	activeID := approvalFixture(t, st)
	agentID := strings.Repeat("a", 26)
	secondConversation := core.Conversation{
		ID: strings.Repeat("d", 26), Kind: "dm", CreatedAt: at,
	}
	if err := st.InsertConversation(ctx, secondConversation, agentID); err != nil {
		t.Fatalf("insert second conversation: %v", err)
	}
	thirdConversation := core.Conversation{
		ID: strings.Repeat("e", 26), Kind: "dm", CreatedAt: at,
	}
	if err := st.InsertConversation(ctx, thirdConversation, agentID); err != nil {
		t.Fatalf("insert third conversation: %v", err)
	}
	queued := core.Run{
		ID: strings.Repeat("Z", 26), AgentID: agentID, ConversationID: secondConversation.ID,
		IdempotencyKey: "queued-first", Trigger: core.TriggerMessage, Status: core.RunQueued,
	}
	if err := st.AdmitRun(ctx, nil, queued); err != nil {
		t.Fatalf("admit first queued run: %v", err)
	}
	later := core.Run{
		ID: strings.Repeat("0", 26), AgentID: agentID, ConversationID: thirdConversation.ID,
		IdempotencyKey: "queued-second", Trigger: core.TriggerMessage, Status: core.RunQueued,
	}
	if err := st.AdmitRun(ctx, nil, later); err != nil {
		t.Fatalf("admit second queued run: %v", err)
	}

	count, err := st.CountActiveAgentRuns(ctx, agentID)
	if err != nil || count != 1 {
		t.Fatalf("active agent runs = %d, %v; want 1", count, err)
	}
	oldest, err := st.OldestQueuedAgentRun(ctx, agentID)
	if err != nil || oldest.ID != queued.ID {
		t.Fatalf("oldest queued agent run = %s, %v; want %s", oldest.ID, err, queued.ID)
	}
	if claimed, err := st.ClaimRun(ctx, queued.ID, at); err != nil || claimed {
		t.Fatalf("claim with another active agent run: claimed=%v err=%v", claimed, err)
	}

	if moved, err := st.SetRunStatus(ctx, activeID, core.RunRunning, core.RunWaitingApproval); err != nil || !moved {
		t.Fatalf("move active run to waiting approval: moved=%v err=%v", moved, err)
	}
	if count, err := st.CountActiveAgentRuns(ctx, agentID); err != nil || count != 1 {
		t.Fatalf("active agent runs while waiting approval = %d, %v; want 1", count, err)
	}
	if claimed, err := st.ClaimRun(ctx, queued.ID, at); err != nil || claimed {
		t.Fatalf("claim while agent waits for approval: claimed=%v err=%v", claimed, err)
	}
	if moved, err := st.SetRunStatus(ctx, activeID, core.RunWaitingApproval, core.RunRunning); err != nil || !moved {
		t.Fatalf("move active run back to running: moved=%v err=%v", moved, err)
	}
	if err := st.FinishRun(ctx, activeID, core.RunDone, at.Add(time.Second), "", 0, 0); err != nil {
		t.Fatalf("finish active run: %v", err)
	}
	if claimed, err := st.ClaimRun(ctx, queued.ID, at.Add(2*time.Second)); err != nil || !claimed {
		t.Fatalf("claim after slot release: claimed=%v err=%v", claimed, err)
	}
}
