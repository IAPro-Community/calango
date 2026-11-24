package api

import (
	"net/http"
	"time"

	"calango/internal/conversations"
	"calango/internal/core"
)

// idempotencyHeader carries the client key that makes message and retry requests safe to repeat.
const idempotencyHeader = "Idempotency-Key"

type createConversationBody struct {
	AgentID string `json:"agent_id"`
	Title   string `json:"title"`
}

type conversationResponse struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type sendMessageBody struct {
	Body string `json:"body"`
}

type messageResponse struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	AuthorKind     string    `json:"author_kind"`
	AuthorID       string    `json:"author_id"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"created_at"`
}

type retryBody struct {
	Instructions string `json:"instructions"`
}

type steerBody struct {
	Guidance string `json:"guidance"`
}

type runResponse struct {
	ID              string     `json:"id"`
	AgentID         string     `json:"agent_id"`
	ConversationID  string     `json:"conversation_id"`
	SourceMessageID string     `json:"source_message_id"`
	Trigger         string     `json:"trigger"`
	Status          string     `json:"status"`
	Instructions    string     `json:"instructions,omitempty"`
	RetryOf         string     `json:"retry_of,omitempty"`
	SteerOf         string     `json:"steer_of,omitempty"`
	TokensIn        int64      `json:"tokens_in"`
	TokensOut       int64      `json:"tokens_out"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	ErrorCode       string     `json:"error_code,omitempty"`
}

type stepResponse struct {
	ID        string    `json:"id"`
	Seq       int64     `json:"seq"`
	Kind      string    `json:"kind"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

type runDetailResponse struct {
	runResponse
	Steps []stepResponse `json:"steps"`
}

type sendMessageResponse struct {
	Message messageResponse `json:"message"`
	Run     runResponse     `json:"run"`
}

func conversationOf(c core.Conversation) conversationResponse {
	return conversationResponse{ID: c.ID, Kind: c.Kind, Title: c.Title, CreatedAt: c.CreatedAt}
}

func messageOf(m core.Message) messageResponse {
	return messageResponse{
		ID: m.ID, ConversationID: m.ConversationID, AuthorKind: m.AuthorKind,
		AuthorID: m.AuthorID, Body: m.Body, CreatedAt: m.CreatedAt,
	}
}

func runOf(r core.Run) runResponse {
	return runResponse{
		ID: r.ID, AgentID: r.AgentID, ConversationID: r.ConversationID,
		SourceMessageID: r.SourceMessageID, Trigger: r.Trigger, Status: string(r.Status),
		Instructions: r.Instructions, RetryOf: r.RetryOf, SteerOf: r.SteerOf, TokensIn: r.TokensIn,
		TokensOut: r.TokensOut, StartedAt: r.StartedAt, EndedAt: r.EndedAt, ErrorCode: r.ErrorCode,
	}
}

func runDetailOf(d conversations.RunDetail) runDetailResponse {
	steps := make([]stepResponse, 0, len(d.Steps))
	for _, st := range d.Steps {
		steps = append(steps, stepResponse{ID: st.ID, Seq: st.Seq, Kind: st.Kind, Payload: st.Payload, CreatedAt: st.CreatedAt})
	}
	return runDetailResponse{runResponse: runOf(d.Run), Steps: steps}
}

func (h *handler) registerConversations(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/conversations", h.need(core.PermRead, h.listConversations))
	mux.HandleFunc("POST /v1/conversations", h.need(core.PermWrite, h.createConversation))
	mux.HandleFunc("GET /v1/conversations/{id}/messages", h.need(core.PermRead, h.listMessages))
	mux.HandleFunc("POST /v1/conversations/{id}/messages", h.need(core.PermWrite, h.sendMessage))
	mux.HandleFunc("GET /v1/runs/{id}", h.need(core.PermRead, h.getRun))
	mux.HandleFunc("POST /v1/runs/{id}/stop", h.need(core.PermWrite, h.stopRun))
	mux.HandleFunc("POST /v1/runs/{id}/retry", h.need(core.PermWrite, h.retryRun))
	mux.HandleFunc("POST /v1/runs/{id}/steer", h.need(core.PermWrite, h.steerRun))
}

func (h *handler) listConversations(w http.ResponseWriter, r *http.Request) {
	list, err := h.conversations.ListConversations(r.Context(), h.ownerID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]conversationResponse, 0, len(list))
	for _, c := range list {
		out = append(out, conversationOf(c))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) createConversation(w http.ResponseWriter, r *http.Request) {
	var body createConversationBody
	if err := decodeJSON(w, r, &body); err != nil {
		return
	}
	c, err := h.conversations.CreateDM(r.Context(), h.ownerID, body.AgentID, body.Title)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, conversationOf(c))
}

func (h *handler) listMessages(w http.ResponseWriter, r *http.Request) {
	list, err := h.conversations.ListMessages(r.Context(), h.ownerID, r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]messageResponse, 0, len(list))
	for _, m := range list {
		out = append(out, messageOf(m))
	}
	writeJSON(w, http.StatusOK, out)
}

// sendMessage stores the message and admits its run. The response carries both, so the
// client does not need a second request to learn the run id.
func (h *handler) sendMessage(w http.ResponseWriter, r *http.Request) {
	var body sendMessageBody
	if err := decodeJSON(w, r, &body); err != nil {
		return
	}
	msg, run, err := h.conversations.SendMessage(r.Context(), h.ownerID, r.PathValue("id"), body.Body, r.Header.Get(idempotencyHeader))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sendMessageResponse{Message: messageOf(msg), Run: runOf(run)})
}

func (h *handler) getRun(w http.ResponseWriter, r *http.Request) {
	d, err := h.conversations.GetRun(r.Context(), h.ownerID, r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, runDetailOf(d))
}

func (h *handler) stopRun(w http.ResponseWriter, r *http.Request) {
	if err := h.conversations.Stop(r.Context(), h.ownerID, r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// retryRun admits a new run for a stopped or failed run. Retrying with the same key returns the same run.
func (h *handler) retryRun(w http.ResponseWriter, r *http.Request) {
	var body retryBody
	if err := decodeJSON(w, r, &body); err != nil {
		return
	}
	run, err := h.conversations.Retry(r.Context(), h.ownerID, r.PathValue("id"), body.Instructions, r.Header.Get(idempotencyHeader))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, runOf(run))
}

// steerRun gives a queued or running run the owner's guidance. It answers with the continuation run.
func (h *handler) steerRun(w http.ResponseWriter, r *http.Request) {
	var body steerBody
	if err := decodeJSON(w, r, &body); err != nil {
		return
	}
	run, err := h.conversations.Steer(r.Context(), h.ownerID, r.PathValue("id"), body.Guidance, r.Header.Get(idempotencyHeader))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, runOf(run))
}
