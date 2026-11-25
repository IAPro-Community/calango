// Package gateway is the only path from Calango to a model. Before a call it
// checks the agent's monthly token budget and reserves the estimated tokens;
// after the call it records the real tokens and their cost. Provider SDKs stay
// behind the Provider interface and are not imported elsewhere.
package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/IAPro-Community/IAPro-SDK/model"
)

var (
	// ErrBudgetExceeded means the call would push the agent past its monthly
	// token limit. No provider is contacted.
	ErrBudgetExceeded = errors.New("gateway: monthly token budget exceeded")
	// ErrUnknownAgent means the caller's agent is not in the ledger.
	ErrUnknownAgent = errors.New("gateway: unknown agent")
	// ErrUnknownProvider means Request.Provider is not configured.
	ErrUnknownProvider = errors.New("gateway: unknown provider")
	// ErrLedger means the call ran (or was refused) but its usage could not be
	// recorded. Callers must not retry such a call on another provider: the first
	// attempt may already have spent tokens.
	ErrLedger = errors.New("gateway: usage ledger failed")
	// ErrUnavailable marks a provider failure that another provider may avoid:
	// transport failure, HTTP 429 or 5xx, or a stream that broke. Adapters wrap
	// it; the caller decides whether output had already reached the client.
	ErrUnavailable = errors.New("gateway: provider unavailable")
)

// maxOutputTokensPerCall bounds MaxOutputTokens so budget arithmetic cannot overflow.
const maxOutputTokensPerCall = 1 << 32

// Provider is implemented by each model vendor adapter.
type Provider interface {
	Name() string
	// Complete sends req and passes each streamed chunk to stream when it is
	// non-nil. It returns the tokens the provider reported. Usage.Estimated is
	// true when the provider could not report exact tokens.
	Complete(ctx context.Context, req Request, stream func(Chunk)) (Usage, error)
}

// EffectsProvider is an optional explicit adapter capability. Unknown
// providers are treated as capable of external effects, even without MCP.
type EffectsProvider interface {
	MayHaveSideEffects() bool
}

// AdmissionProvider holds an account reservation across dispatch and ledger
// recording. On ambiguous failure, the adapter must retain its fence.
type AdmissionProvider interface {
	Admit(context.Context, *Request) (func(error) error, error)
}

// CapacityRoute is a host-only pre-dispatch decision. Never deserialize it
// from an API request or use it as a substitute for revalidation.
type CapacityRoute struct {
	OwnerID     string
	RunID       string
	AccountKey  string
	Model       string
	Scope       model.AccountScope
	ExpiresAt   time.Time
	SourceKey   string
	SourceScope model.AccountScope
	Reason      string
}

type attemptError struct {
	err     error
	effects bool
}

func (e *attemptError) Error() string { return e.err.Error() }
func (e *attemptError) Unwrap() error { return e.err }

// AttemptMayHaveEffects returns conservative evidence for chain retry policy.
func AttemptMayHaveEffects(err error) bool {
	var attempt *attemptError
	if errors.As(err, &attempt) {
		return attempt.effects
	}
	return true
}

// Request is one model call. MaxOutputTokens must be positive: it is part of
// the pre-call budget estimate. Tools, when set, is the MCP endpoint the model
// may call for this turn; providers that cannot use it ignore it.
type Request struct {
	Provider        string
	Model           string
	Prompt          string
	MaxOutputTokens int64
	Tools           *ToolEndpoint
	// CodexHome overrides the account home for an explicitly linked Maestro agent.
	// Only the daemon sets this local execution option; it is not an API field.
	CodexHome      string
	CapacityRoute  *CapacityRoute
	AdmissionScope *model.AccountScope
	AdmissionOwner string
	CheckAdmission func(context.Context) error
}

// ToolEndpoint is the MCP endpoint of one turn. Token is the bearer token that
// authorizes this turn only; it is a secret and never goes into argv or logs.
type ToolEndpoint struct {
	URL   string
	Token string
}

// Chunk is one piece of streamed output.
type Chunk struct {
	Text string
}

// Usage is the token count of one call.
type Usage struct {
	TokensIn  int64
	TokensOut int64
	Estimated bool
}

// Caller identifies who spent the tokens.
type Caller struct {
	AgentID   string
	RoutineID string
	ProjectID string
	NodeID    string
}

// CostStatus says how trustworthy a recorded cost is.
type CostStatus string

const (
	// CostKnown: exact tokens multiplied by a configured price.
	CostKnown CostStatus = "known"
	// CostEstimated: the tokens were estimated, so the cost is too.
	CostEstimated CostStatus = "estimated"
	// CostUnknown: no price is configured for the model. CostMicros is nil.
	CostUnknown CostStatus = "unknown"
)

// Record is one row of the usage ledger.
type Record struct {
	ID             string
	At             time.Time
	Caller         Caller
	Provider       string
	Model          string
	TokensIn       int64
	TokensOut      int64
	CostMicros     *int64
	CostStatus     CostStatus
	PricingVersion string
}

// Ledger persists budgets and usage. The SQLite implementation lives in internal/store.
type Ledger interface {
	// AgentTokenLimit returns the agent's monthly token limit, or an error
	// wrapping ErrUnknownAgent.
	AgentTokenLimit(ctx context.Context, agentID string) (int64, error)
	// MonthlyTokens sums tokens_in+tokens_out for the agent in [from, to).
	MonthlyTokens(ctx context.Context, agentID string, from, to time.Time) (int64, error)
	RecordUsage(ctx context.Context, rec Record) error
}

// ThresholdEvent reports that an agent's monthly usage crossed a threshold.
type ThresholdEvent struct {
	AgentID     string
	Percent     int64
	UsedTokens  int64
	LimitTokens int64
}

// Config wires a Gateway. Now and NewID are injected so tests are deterministic.
type Config struct {
	Ledger    Ledger
	Providers []Provider
	// Resolve builds a provider for a name that is not in Providers, such as a
	// per-account key. It may be nil.
	Resolve func(ctx context.Context, name string) (Provider, error)
	Prices  PriceTable
	Now     func() time.Time
	NewID   func(time.Time) (string, error)
	// OnThreshold is called synchronously after the record that crosses 80% or 100%.
	OnThreshold func(ThresholdEvent)
}

// Gateway checks budgets and records usage around every model call.
type Gateway struct {
	ledger      Ledger
	providers   map[string]Provider
	resolve     func(ctx context.Context, name string) (Provider, error)
	prices      PriceTable
	now         func() time.Time
	newID       func(time.Time) (string, error)
	onThreshold func(ThresholdEvent)

	// mu serializes budget checks, reservations and usage writes so concurrent
	// calls cannot both spend the same remaining tokens.
	mu       sync.Mutex
	reserved map[string]int64
}

// New validates cfg and returns a Gateway.
func New(cfg Config) (*Gateway, error) {
	if cfg.Ledger == nil {
		return nil, errors.New("gateway: ledger is required")
	}
	if cfg.Now == nil || cfg.NewID == nil {
		return nil, errors.New("gateway: clock and id generator are required")
	}
	if err := cfg.Prices.validate(); err != nil {
		return nil, err
	}

	providers := make(map[string]Provider, len(cfg.Providers))
	for _, p := range cfg.Providers {
		if p.Name() == "" {
			return nil, errors.New("gateway: provider name is required")
		}
		if _, dup := providers[p.Name()]; dup {
			return nil, fmt.Errorf("gateway: duplicate provider %q", p.Name())
		}
		providers[p.Name()] = p
	}

	return &Gateway{
		ledger:      cfg.Ledger,
		providers:   providers,
		resolve:     cfg.Resolve,
		prices:      cfg.Prices,
		now:         cfg.Now,
		newID:       cfg.NewID,
		onThreshold: cfg.OnThreshold,
		reserved:    make(map[string]int64),
	}, nil
}

// Call runs req for who. It fails with ErrBudgetExceeded before contacting the
// provider when the estimate does not fit the remaining monthly budget. Once
// the provider returns (or fails), the call is recorded even if ctx was
// cancelled, so spent tokens are never lost.
func (g *Gateway) Call(ctx context.Context, who Caller, req Request, stream func(Chunk)) (Usage, error) {
	if who.AgentID == "" {
		return Usage{}, errors.New("gateway: caller agent id is required")
	}
	if req.Model == "" {
		return Usage{}, errors.New("gateway: model is required")
	}
	if req.MaxOutputTokens <= 0 || req.MaxOutputTokens > maxOutputTokensPerCall {
		return Usage{}, fmt.Errorf("gateway: max output tokens %d outside (0, %d]", req.MaxOutputTokens, int64(maxOutputTokensPerCall))
	}
	provider, ok := g.providers[req.Provider]
	if !ok && g.resolve != nil {
		resolved, err := g.resolve(ctx, req.Provider)
		if err != nil {
			return Usage{}, fmt.Errorf("%w: %q: %w", ErrUnknownProvider, req.Provider, err)
		}
		provider, ok = resolved, true
	}
	if !ok {
		return Usage{}, fmt.Errorf("%w: %q", ErrUnknownProvider, req.Provider)
	}

	limit, err := g.ledger.AgentTokenLimit(ctx, who.AgentID)
	if err != nil {
		return Usage{}, err
	}

	now := g.now().UTC()
	estimate := estimateUsage(req)
	reserve := estimate.TokensIn + estimate.TokensOut
	if err := g.reserve(ctx, who.AgentID, limit, reserve, now); err != nil {
		return Usage{}, err
	}
	defer g.release(who.AgentID, reserve)

	// Allocate the ledger identity before dispatch, so failure cannot leave
	// an executed attempt without a usage ID.
	recordID, err := g.newID(now)
	if err != nil {
		return Usage{}, fmt.Errorf("%w: create usage id: %w", ErrLedger, err)
	}
	if recordID == "" {
		return Usage{}, fmt.Errorf("%w: empty usage id", ErrLedger)
	}
	effects := true
	if declared, ok := provider.(EffectsProvider); ok {
		effects = declared.MayHaveSideEffects()
	}

	var finishAdmission func(error) error
	if admission, ok := provider.(AdmissionProvider); ok {
		finishAdmission, err = admission.Admit(ctx, &req)
		if err != nil {
			return Usage{}, err
		}
	}
	if req.CapacityRoute != nil && (finishAdmission == nil || req.AdmissionScope == nil || req.CheckAdmission == nil) {
		if finishAdmission != nil {
			_ = finishAdmission(errors.New("gateway: invalid capacity admission"))
		}
		return Usage{}, errors.New("gateway: capacity route requires a valid account admission")
	}
	usage, callErr := provider.Complete(ctx, req, stream)
	switch {
	case usage.TokensIn < 0 || usage.TokensOut < 0:
		callErr = errors.Join(callErr, fmt.Errorf("gateway: provider %s reported negative tokens", provider.Name()))
		usage = estimate
	case callErr != nil && usage == (Usage{}):
		// Nothing was reported, so charge the reserved estimate. Failed calls
		// must not become a way around the budget.
		usage = estimate
	}

	rec, costErr := g.buildRecord(who, req, usage, now)
	rec.ID = recordID

	recordErr := g.record(context.WithoutCancel(ctx), who.AgentID, limit, rec, now)
	if recordErr != nil {
		recordErr = fmt.Errorf("%w: %w", ErrLedger, recordErr)
	}

	if costErr != nil {
		costErr = fmt.Errorf("%w: calculate usage cost: %w", ErrLedger, costErr)
	}
	err = errors.Join(callErr, costErr, recordErr)
	if finishAdmission != nil {
		err = errors.Join(err, finishAdmission(err))
	}
	if err != nil {
		return usage, &attemptError{err: fmt.Errorf("gateway: call %s/%s: %w", req.Provider, req.Model, err), effects: effects}
	}
	return usage, nil
}

// reserve checks that tokens fit in the agent's remaining monthly budget,
// counting tokens already reserved by in-flight calls, and holds them.
func (g *Gateway) reserve(ctx context.Context, agentID string, limit, tokens int64, now time.Time) error {
	from, to := monthWindow(now)

	g.mu.Lock()
	defer g.mu.Unlock()

	used, err := g.ledger.MonthlyTokens(ctx, agentID, from, to)
	if err != nil {
		return err
	}
	if tokens > limit-used-g.reserved[agentID] {
		return fmt.Errorf("%w: agent %s has %d tokens left this month, call needs about %d",
			ErrBudgetExceeded, agentID, limit-used-g.reserved[agentID], tokens)
	}
	g.reserved[agentID] += tokens
	return nil
}

func (g *Gateway) release(agentID string, tokens int64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.reserved[agentID] -= tokens
	if g.reserved[agentID] == 0 {
		delete(g.reserved, agentID)
	}
}

// record writes the usage row and emits threshold events for any 80% or 100%
// boundary crossed by this record. It holds mu so the before/after totals are
// consistent across concurrent calls.
func (g *Gateway) record(ctx context.Context, agentID string, limit int64, rec Record, now time.Time) error {
	from, to := monthWindow(now)

	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.ledger.RecordUsage(ctx, rec); err != nil {
		return err
	}
	if g.onThreshold == nil {
		return nil
	}
	total, err := g.ledger.MonthlyTokens(ctx, agentID, from, to)
	if err != nil {
		return err
	}
	delta := rec.TokensIn + rec.TokensOut
	for _, pct := range []int64{80, 100} {
		if !reachesPercent(total-delta, limit, pct) && reachesPercent(total, limit, pct) {
			g.onThreshold(ThresholdEvent{
				AgentID:     agentID,
				Percent:     pct,
				UsedTokens:  total,
				LimitTokens: limit,
			})
		}
	}
	return nil
}

func (g *Gateway) buildRecord(who Caller, req Request, usage Usage, at time.Time) (Record, error) {
	rec := Record{
		At:         at,
		Caller:     who,
		Provider:   req.Provider,
		Model:      req.Model,
		TokensIn:   usage.TokensIn,
		TokensOut:  usage.TokensOut,
		CostStatus: CostUnknown,
	}

	price, ok := g.prices.lookup(req.Provider, req.Model)
	if !ok {
		// A missing price is unknown, never zero.
		return rec, nil
	}
	if price.AccountIdentitySHA256 != "" {
		if req.AdmissionScope == nil {
			return rec, nil
		}
		digest := sha256.Sum256([]byte(req.AdmissionScope.AccountID))
		if price.AccountIdentitySHA256 != hex.EncodeToString(digest[:]) {
			return rec, nil
		}
	}
	cost, err := price.cost(usage.TokensIn, usage.TokensOut)
	if err != nil {
		return rec, err
	}
	rec.CostMicros = &cost
	rec.PricingVersion = g.prices.Version
	rec.CostStatus = CostKnown
	if usage.Estimated {
		rec.CostStatus = CostEstimated
	}
	return rec, nil
}

// estimateUsage bounds a call before it is sent: about four bytes per token
// for the prompt plus the full output cap.
func estimateUsage(req Request) Usage {
	promptTokens := (int64(len(req.Prompt)) + 3) / 4
	return Usage{TokensIn: promptTokens, TokensOut: req.MaxOutputTokens, Estimated: true}
}

// monthWindow returns the UTC month containing t as [from, to).
func monthWindow(t time.Time) (time.Time, time.Time) {
	t = t.UTC()
	from := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	return from, from.AddDate(0, 1, 0)
}

// reachesPercent reports used >= limit*percent/100 using exact integer arithmetic.
func reachesPercent(used, limit, percent int64) bool {
	left := new(big.Int).Mul(big.NewInt(used), big.NewInt(100))
	right := new(big.Int).Mul(big.NewInt(limit), big.NewInt(percent))
	return left.Cmp(right) >= 0
}
