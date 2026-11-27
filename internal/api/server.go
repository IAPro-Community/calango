// Package api exposes the node over HTTP. /healthz and POST /v1/pair are open. Every
// other /v1 route needs a bearer token, and the scope of the token's device decides
// which routes it may call.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"calango/internal/agents"
	"calango/internal/approvals"
	"calango/internal/auth"
	"calango/internal/conversations"
	"calango/internal/core"
	"calango/internal/devices"
	"calango/internal/events"
	"calango/internal/maestro"
	"calango/internal/memory"
	"calango/internal/providers"
	"calango/internal/pty"
	"calango/internal/routines"
	"calango/internal/skills"
	"calango/internal/tools"
	"calango/internal/usage"
)

// maxBodyBytes bounds every request body. Agent instructions are the largest field.
const maxBodyBytes = 1 << 20

// Config wires the HTTP handler to the services it calls.
type Config struct {
	Agents    *agents.Service
	Memories  *memory.Service
	Approvals *approvals.Service
	Devices   *devices.Service
	// Accounts manages provider accounts for /v1/provider-accounts.
	Accounts *providers.Service
	Usage    *usage.Service
	// Conversations manages conversation, message and run routes.
	Conversations *conversations.Service
	// Events fans domain changes out to /v1/events subscribers.
	Events *events.Hub
	// Routines manages /v1/routines.
	Routines *routines.Service
	// Tools manages connector routes and agent grants.
	Tools *tools.Service
	// Maestro manages the isolated official CLI. Nil disables its API routes.
	Maestro *maestro.Manager
	// Skills discovers and imports Claude Code and Codex skills. Nil disables its routes.
	Skills *skills.Library
	// Terminal runs integrated terminal sessions for /v1/pty. Nil disables its routes.
	Terminal *pty.Manager
	// Vault stores secrets and never returns a secret value.
	Vault VaultStore
	// Audit reads the audit chain and exports.
	Audit AuditLedger
	// Token is the live local bearer token; it acts with full scope.
	Token *auth.LocalToken
	// OwnerID scopes all reads and writes to one owner.
	OwnerID string
	// DeviceID identifies decisions made with the local token.
	DeviceID string
	// TrustedProxies lists peers whose forwarded-for header counts toward pairing limits.
	TrustedProxies []netip.Prefix
	// CORSOrigins lists browser origins allowed to call the API.
	CORSOrigins []string
}

type handler struct {
	agents        *agents.Service
	memories      *memory.Service
	accounts      *providers.Service
	approvals     *approvals.Service
	devices       *devices.Service
	usage         *usage.Service
	audit         AuditLedger
	vault         VaultStore
	conversations *conversations.Service
	events        *events.Hub
	routines      *routines.Service
	tools         *tools.Service
	maestro       *maestro.Manager
	skills        *skills.Library
	terminal      *pty.Manager
	localToken    *auth.LocalToken
	ownerID       string
	deviceID      string
	pairLimit     *attemptLimiter
	trusted       []netip.Prefix
}

// NewHandler validates cfg and returns the node's HTTP handler.
func NewHandler(cfg Config) (http.Handler, error) {
	switch {
	case cfg.Agents == nil, cfg.Memories == nil, cfg.Approvals == nil, cfg.Devices == nil, cfg.Usage == nil, cfg.Vault == nil, cfg.Audit == nil, cfg.Conversations == nil, cfg.Events == nil, cfg.Accounts == nil, cfg.Routines == nil, cfg.Tools == nil:
		return nil, errors.New("api: agents, memories, approvals, devices, usage, vault, audit, conversations, events, provider accounts, routines and tools services are required")
	case cfg.Token == nil:
		return nil, errors.New("api: bearer token is required")
	case cfg.OwnerID == "", cfg.DeviceID == "":
		return nil, errors.New("api: owner and device ids are required")
	}

	h := &handler{
		agents: cfg.Agents, memories: cfg.Memories, accounts: cfg.Accounts,
		approvals: cfg.Approvals, devices: cfg.Devices, usage: cfg.Usage,
		vault: cfg.Vault, audit: cfg.Audit, conversations: cfg.Conversations,
		events: cfg.Events, routines: cfg.Routines, tools: cfg.Tools,
		maestro: cfg.Maestro, skills: cfg.Skills, terminal: cfg.Terminal, localToken: cfg.Token, ownerID: cfg.OwnerID,
		deviceID: cfg.DeviceID, pairLimit: newAttemptLimiter(pairAttemptsPerWindow, pairWindow, time.Now),
		trusted: cfg.TrustedProxies,
	}
	v1 := http.NewServeMux()
	h.registerAgents(v1)
	h.registerMemories(v1)
	h.registerApprovals(v1)
	h.registerDevices(v1)
	h.registerUsage(v1)
	h.registerVault(v1)
	h.registerAuth(v1)
	h.registerAudit(v1)
	h.registerConversations(v1)
	h.registerEvents(v1)
	h.registerProviderAccounts(v1)
	h.registerRoutines(v1)
	h.registerTools(v1)
	if h.maestro != nil {
		h.registerMaestro(v1)
	}
	if h.skills != nil {
		h.registerSkills(v1)
	}
	if h.terminal != nil {
		h.registerPTY(v1)
	}

	root := http.NewServeMux()
	root.HandleFunc("GET /healthz", healthz)
	root.HandleFunc("POST /v1/pair", h.pair)
	root.Handle("/", h.authenticate(v1))
	return allowCORS(cfg.CORSOrigins, root), nil
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// principal is the caller of one request: the device behind the token and its scope.
type principal struct {
	deviceID string
	scope    core.DeviceScope
}

type principalKey struct{}

func withPrincipal(ctx context.Context, p principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

func principalOf(ctx context.Context) principal {
	p, _ := ctx.Value(principalKey{}).(principal)
	return p
}

// authenticate resolves the bearer token to a principal. The local token is compared in
// constant time and acts with full scope. Any other token must belong to an active device.
func (h *handler) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			unauthorized(w)
			return
		}
		if h.localToken.Matches(token) {
			next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), principal{h.deviceID, core.ScopeFull})))
			return
		}
		d, err := h.devices.Authenticate(r.Context(), token)
		if err != nil {
			if errors.Is(err, core.ErrUnauthorized) {
				unauthorized(w)
				return
			}
			writeServiceError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), principal{d.ID, d.Scope})))
	})
}

// need admits a request only when the caller's scope grants perm.
func (h *handler) need(perm core.Permission, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !principalOf(r.Context()).scope.Allows(perm) {
			writeError(w, http.StatusForbidden, codeForbidden, "this device's scope does not allow the action")
			return
		}
		next(w, r)
	}
}

func bearerToken(r *http.Request) string {
	got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return ""
	}
	return strings.TrimSpace(got)
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="calango"`)
	writeError(w, http.StatusUnauthorized, codeUnauthorized, "missing or invalid bearer token")
}

// decodeJSON reads one JSON object, rejecting unknown fields and trailing data.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	return decodeBody(w, r, dst, false)
}

// decodeOptionalJSON is decodeJSON for routes whose body may be empty.
func decodeOptionalJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	return decodeBody(w, r, dst, true)
}

func decodeBody(w http.ResponseWriter, r *http.Request, dst any, allowEmpty bool) error {
	body := http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if allowEmpty && errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("%w: request body: %v", core.ErrInvalidInput, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: request body must hold a single JSON object", core.ErrInvalidInput)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("api: write response: %v", err)
	}
}
