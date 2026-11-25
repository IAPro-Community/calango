package gateway

import "github.com/IAPro-Community/IAPro-SDK/cost"

// LoadPriceTable loads a tariff from a daemon-authorized path.
func LoadPriceTable(path string) (PriceTable, error) {
	return cost.LoadTariffFile(path)
}
