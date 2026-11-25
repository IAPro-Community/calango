package gateway

import (
	"fmt"
	"math/big"
)

const microsPerMillionTokens = 1_000_000

// Price is the cost of one million tokens, in micros of the table's currency.
// Integer micros keep cost exact; floats are never used for money.
type Price struct {
	InputPerMillion  int64
	OutputPerMillion int64
}

// PriceTable maps "provider/model" to its price. Version identifies the
// published price list the entries were taken from, and is stored with every
// known cost. A model without an entry has unknown cost; it is never zero.
type PriceTable struct {
	Version string
	Models  map[string]Price
}

func (t PriceTable) validate() error {
	for key, p := range t.Models {
		if p.InputPerMillion < 0 || p.OutputPerMillion < 0 {
			return fmt.Errorf("gateway: negative price for %s", key)
		}
	}
	return nil
}

// ConfirmedZeroTariff requires an explicit, versioned zero-marginal-cost
// entry. Absence of a price never grants permission to switch accounts.
func (t PriceTable) ConfirmedZeroTariff(provider, model string) bool {
	p, ok := t.lookup(provider, model)
	return ok && t.Version != "" && p.InputPerMillion == 0 && p.OutputPerMillion == 0
}

func (t PriceTable) lookup(provider, model string) (Price, bool) {
	p, ok := t.Models[provider+"/"+model]
	return p, ok
}

// cost returns tokensIn*input + tokensOut*output, in micros, rounded up so a
// fractional micro is never dropped.
func (p Price) cost(tokensIn, tokensOut int64) (int64, error) {
	num := new(big.Int).Mul(big.NewInt(tokensIn), big.NewInt(p.InputPerMillion))
	num.Add(num, new(big.Int).Mul(big.NewInt(tokensOut), big.NewInt(p.OutputPerMillion)))
	num.Add(num, big.NewInt(microsPerMillionTokens-1))
	num.Quo(num, big.NewInt(microsPerMillionTokens))
	if !num.IsInt64() {
		return 0, fmt.Errorf("gateway: cost overflows int64 micros")
	}
	return num.Int64(), nil
}
