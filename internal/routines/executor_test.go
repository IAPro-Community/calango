package routines

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"calango/internal/conversations"
	"calango/internal/core"
)

type fakeChat struct {
	creates int
	keys    []string
	bodies  []string
	detail  conversations.RunDetail
}

func (c *fakeChat) CreateDM(_ context.Context, _, _, _ string) (core.Conversation, error) {
	c.creates++
	return core.Conversation{ID: "conv-1"}, nil
}

func (c *fakeChat) SendMessage(_ context.Context, _, _, body, key string) (core.Message, core.Run, error) {
	c.bodies = append(c.bodies, body)
	c.keys = append(c.keys, key)
	return core.Message{ID: "msg-" + key}, core.Run{ID: "run-" + key}, nil
}

func (c *fakeChat) WaitRun(_ context.Context, _, _ string) (conversations.RunDetail, error) {
	return c.detail, nil
}

func doneRun(reply string) conversations.RunDetail {
	payload, _ := json.Marshal(map[string]string{"text": reply})
	return conversations.RunDetail{
		Run:   core.Run{Status: core.RunDone},
		Steps: []core.Step{{Kind: core.StepText, Payload: string(payload)}},
	}
}

func routineFor() core.Routine {
	return core.Routine{ID: "r1", OwnerID: "owner", AgentID: "a1", Name: "digest", Prompt: "summarise"}
}

func occurrence(attempt int) core.Occurrence {
	return core.Occurrence{OccurrenceID: "r1@occ", Attempt: attempt}
}

func TestExecutorOpensOneConversationAndKeysEachAttempt(t *testing.T) {
	ctx := context.Background()
	chat := &fakeChat{detail: doneRun("ok\nROUTINE: {\"had_news\": false, \"finish\": false}")}
	x := NewChatExecutor(chat)

	r := routineFor()
	first, err := x.Execute(ctx, r, occurrence(1))
	if err != nil {
		t.Fatal(err)
	}
	r.ConversationID = first.ConversationID
	if _, err := x.Execute(ctx, r, occurrence(2)); err != nil {
		t.Fatal(err)
	}

	if chat.creates != 1 {
		t.Fatalf("created %d conversations, want 1 per routine", chat.creates)
	}
	if got := strings.Join(chat.keys, ","); got != "r1@occ#1,r1@occ#2" {
		t.Fatalf("idempotency keys = %q, want one per attempt", got)
	}
	if chat.bodies[0] != core.RoutineMessage("summarise") {
		t.Fatalf("sent body = %q, want the routine message", chat.bodies[0])
	}
}

func TestExecutorReadsControlLine(t *testing.T) {
	chat := &fakeChat{detail: doneRun("nothing new\nROUTINE: {\"had_news\": false, \"finish\": true}")}
	res, err := NewChatExecutor(chat).Execute(context.Background(), routineFor(), occurrence(1))
	if err != nil {
		t.Fatal(err)
	}
	if res.HadNews || !res.Finish {
		t.Fatalf("result = %+v, want had_news=false finish=true", res)
	}
}

func TestMissingControlLineDefaultsToNewsAndKeepsRoutineAlive(t *testing.T) {
	chat := &fakeChat{detail: doneRun("a reply without any control line")}
	res, err := NewChatExecutor(chat).Execute(context.Background(), routineFor(), occurrence(1))
	if err != nil {
		t.Fatal(err)
	}
	if !res.HadNews || res.Finish {
		t.Fatalf("result = %+v, want had_news=true finish=false", res)
	}
}

func TestFailedRunKeepsConversationForRetry(t *testing.T) {
	chat := &fakeChat{detail: conversations.RunDetail{Run: core.Run{Status: core.RunFailed, ErrorCode: "provider_unavailable"}}}
	res, err := NewChatExecutor(chat).Execute(context.Background(), routineFor(), occurrence(1))
	if err == nil {
		t.Fatal("failed run returned no error")
	}
	if res.ConversationID != "conv-1" {
		t.Fatalf("conversation id = %q on failure, want conv-1 so retries reuse it", res.ConversationID)
	}
}
