package gateway

import "github.com/IAPro-Community/IAPro-SDK/cost"

// Price and PriceTable share the SDK tariff contract; the host owns its ledger.
type Price = cost.Tariff
type PriceTable = cost.TariffTable
