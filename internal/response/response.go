package response

import "github.com/tslnc04/tax-calculator/internal/jurisdiction"

type Response struct {
	Earnings   Earnings      `json:"earnings"`
	Taxes      Taxes         `json:"taxes"`
	Gross      SummaryEntity `json:"gross"`
	Net        SummaryEntity `json:"net"`
	Deductions Deductions    `json:"deductions"`
}

type SummaryEntity struct {
	Amount       float64 `json:"amount"`
	CurrencyCode string  `json:"currencyCode"`
	Label        string  `json:"label"`
}

type Earnings struct {
	Entities      []EarningsEntity `json:"entities"`
	SummaryEntity SummaryEntity    `json:"summaryEntity"`
}

type EarningsEntity struct {
	Amount       float64 `json:"amount"`
	CurrencyCode string  `json:"currencyCode"`
	Label        string  `json:"label"`
	Hours        float64 `json:"hours"`
}

type Taxes struct {
	Federal       TaxEntities   `json:"federal"`
	State         TaxEntities   `json:"state"`
	Local         TaxEntities   `json:"local"`
	Territory     TaxEntities   `json:"territory"`
	SummaryEntity SummaryEntity `json:"summaryEntity"`
}

type TaxEntity struct {
	Amount             float64                   `json:"amount"`
	CurrencyCode       string                    `json:"currencyCode"`
	Label              string                    `json:"label"`
	Jurisdiction       jurisdiction.Jurisdiction `json:"jurisdiction"`
	ParentJurisdiction jurisdiction.Jurisdiction `json:"parentJurisdiction,omitempty"`
}

type TaxEntities struct {
	Entities      []TaxEntity   `json:"entities"`
	SummaryEntity SummaryEntity `json:"summaryEntity"`
}

type Deductions struct {
	Entities      []struct{}    `json:"entities"`
	SummaryEntity SummaryEntity `json:"summaryEntity"`
}
