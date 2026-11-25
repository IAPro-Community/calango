package gateway

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
)

// LoadPriceTable loads an explicit versioned tariff. Missing configuration is
// intentionally unknown cost. A configured but invalid file fails startup.
func LoadPriceTable(path string) (PriceTable, error) {
	if path == "" {
		return PriceTable{}, nil
	}
	// Reject special files before opening (a FIFO may block on open).
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return PriceTable{}, errors.New("configured tariff must be a regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return PriceTable{}, errors.New("cannot open configured tariff file")
	}
	defer func() { _ = f.Close() }()
	info, err = f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return PriceTable{}, errors.New("configured tariff must be a regular file")
	}
	const maxSize = 1 << 20
	data, err := io.ReadAll(io.LimitReader(f, maxSize+1))
	if err != nil || len(data) > maxSize {
		return PriceTable{}, errors.New("cannot read tariff file or file exceeds 1 MiB")
	}
	if err := rejectDuplicateTariffKeys(json.NewDecoder(strings.NewReader(string(data))), 0); err != nil {
		return PriceTable{}, errors.New("invalid tariff JSON or duplicate object key")
	}
	type entry struct {
		Input                 *int64 `json:"input_per_million"`
		Output                *int64 `json:"output_per_million"`
		AccountIdentitySHA256 string `json:"account_identity_sha256"`
	}
	var input struct {
		Version  string           `json:"version"`
		Currency string           `json:"currency"`
		Models   map[string]entry `json:"models"`
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil {
		return PriceTable{}, errors.New("invalid tariff JSON")
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return PriceTable{}, errors.New("tariff JSON contains trailing data")
	}
	if strings.TrimSpace(input.Version) == "" || input.Currency != "USD" || input.Models == nil {
		return PriceTable{}, errors.New("tariff requires version, USD currency and models")
	}
	table := PriceTable{Version: input.Version, Models: make(map[string]Price, len(input.Models))}
	for key, price := range input.Models {
		provider, model, ok := strings.Cut(key, "/")
		if !ok || provider == "" || model == "" || strings.TrimSpace(key) != key || price.Input == nil || price.Output == nil {
			return PriceTable{}, errors.New("tariff entry requires provider/model and both prices")
		}
		table.Models[key] = Price{InputPerMillion: *price.Input, OutputPerMillion: *price.Output, AccountIdentitySHA256: price.AccountIdentitySHA256}
	}
	if err := table.validate(); err != nil {
		return PriceTable{}, errors.New("tariff prices must be non-negative")
	}
	return table, nil
}

// rejectDuplicateTariffKeys walks the JSON before typed decoding, which would
// otherwise silently overwrite duplicate keys. Struct fields are matched
// case-insensitively by encoding/json; model map keys are case-sensitive.
func rejectDuplicateTariffKeys(dec *json.Decoder, depth int) error {
	if depth > 32 {
		return errors.New("tariff nesting too deep")
	}
	token, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]bool)
		for dec.More() {
			key, err := dec.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok {
				return errors.New("invalid object key")
			}
			if depth != 1 {
				name = strings.ToLower(name)
			}
			if seen[name] {
				return errors.New("duplicate object key")
			}
			seen[name] = true
			if err := rejectDuplicateTariffKeys(dec, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for dec.More() {
			if err := rejectDuplicateTariffKeys(dec, depth+1); err != nil {
				return err
			}
		}
	default:
		return errors.New("unexpected delimiter")
	}
	_, err = dec.Token()
	return err
}
