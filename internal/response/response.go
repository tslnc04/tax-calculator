// Package response implements the types for the response from the ADP API. It currently holds no logic.
package response

import "github.com/tslnc04/tax-calculator/internal/jurisdiction"

// Response is the response from the ADP API.
type Response struct {
	Earnings   Earnings      `json:"earnings"`
	Taxes      Taxes         `json:"taxes"`
	Gross      SummaryEntity `json:"gross"`
	Net        SummaryEntity `json:"net"`
	Deductions Deductions    `json:"deductions"`
}

// SummaryEntity is a summary of the response. It usually sums up all the amounts in a section of the response.
type SummaryEntity struct {
	Amount       float64 `json:"amount"`
	CurrencyCode string  `json:"currencyCode"`
	Label        string  `json:"label"`
}

// Earnings is the earnings section of the response. It makes up the gross income.
type Earnings struct {
	Entities      []EarningsEntity `json:"entities"`
	SummaryEntity SummaryEntity    `json:"summaryEntity"`
}

// EarningsEntity is an entity in the earnings section of the response. It represents a single income source.
type EarningsEntity struct {
	Amount       float64 `json:"amount"`
	CurrencyCode string  `json:"currencyCode"`
	Label        string  `json:"label"`
	Hours        float64 `json:"hours"`
}

// Taxes is the tax section of the response. It includes all of the taxes that get subtracted from the gross income.
type Taxes struct {
	Federal       TaxEntities   `json:"federal"`
	State         TaxEntities   `json:"state"`
	Local         TaxEntities   `json:"local"`
	Territory     TaxEntities   `json:"territory"`
	SummaryEntity SummaryEntity `json:"summaryEntity"`
}

// TaxEntities contains all of the tax entities for a jurisdiction.
type TaxEntities struct {
	Entities      []TaxEntity   `json:"entities"`
	SummaryEntity SummaryEntity `json:"summaryEntity"`
}

// TaxEntity is a single source of taxes for a jurisdiction.
type TaxEntity struct {
	Amount             float64                   `json:"amount"`
	CurrencyCode       string                    `json:"currencyCode"`
	Label              string                    `json:"label"`
	Jurisdiction       jurisdiction.Jurisdiction `json:"jurisdiction"`
	ParentJurisdiction jurisdiction.Jurisdiction `json:"parentJurisdiction,omitempty"`
}

// Deductions contains all of the deductions for the response. The format of the entities has not been determined yet.
type Deductions struct {
	Entities      []struct{}    `json:"entities"`
	SummaryEntity SummaryEntity `json:"summaryEntity"`
}
