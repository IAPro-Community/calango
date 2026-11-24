package routines

import (
	"context"
	"fmt"

	"calango/internal/conversations"
	"calango/internal/core"
)

// Chat is the part of the conversations service that routines use. Every model call still
// goes through the conversations service and the gateway.
type Chat interface {
	CreateDM(ctx context.Context, ownerID, agentID, title string) (core.Conversation, error)
	SendMessage(ctx context.Context, ownerID, conversationID, body, idempotencyKey string) (core.Message, core.Run, error)
	WaitRun(ctx context.Context, ownerID, runID string) (conversations.RunDetail, error)
}

// ChatExecutor runs each routine in a DM of its own: one conversation per routine, one
// agent run per attempt.
type ChatExecutor struct {
	chat Chat
}

// NewChatExecutor builds the executor over the conversations service.
func NewChatExecutor(chat Chat) *ChatExecutor {
	return &ChatExecutor{chat: chat}
}

// Execute sends the routine prompt and waits for the run. Each attempt gets its own
// idempotency key, occurrence plus attempt, so a retry starts a new run instead of
// returning the failed one. The conversation ID is returned even on failure, so a
// retry does not open a second conversation.
func (x *ChatExecutor) Execute(ctx context.Context, r core.Routine, o core.Occurrence) (Result, error) {
	var res Result
	convID := r.ConversationID
	if convID == "" {
		conv, err := x.chat.CreateDM(ctx, r.OwnerID, r.AgentID, r.Name)
		if err != nil {
			return res, fmt.Errorf("create routine conversation: %w", err)
		}
		convID = conv.ID
	}
	res.ConversationID = convID

	key := fmt.Sprintf("%s#%d", o.OccurrenceID, o.Attempt)
	_, run, err := x.chat.SendMessage(ctx, r.OwnerID, convID, core.RoutineMessage(r.Prompt), key)
	if err != nil {
		return res, fmt.Errorf("send routine message: %w", err)
	}
	res.RunID = run.ID

	detail, err := x.chat.WaitRun(ctx, r.OwnerID, run.ID)
	if err != nil {
		return res, fmt.Errorf("wait for run %s: %w", run.ID, err)
	}
	if detail.Run.Status != core.RunDone {
		return res, fmt.Errorf("run %s ended %s: %s", run.ID, detail.Run.Status, detail.Run.ErrorCode)
	}

	reply, err := conversations.ReplyText(detail.Steps)
	if err != nil {
		return res, err
	}
	sig, _ := core.ParseRoutineSignal(reply)
	res.HadNews = sig.HadNews
	res.Finish = sig.Finish
	return res, nil
}
