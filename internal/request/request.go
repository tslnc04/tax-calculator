// Package request contains the types and builder for making requests to the ADP tax calculator API.
package request

import (
	"fmt"
	"strconv"

	"github.com/golang/glog"
	"github.com/tslnc04/tax-calculator/internal/jurisdiction"
)

// APIURL is the URL of the ADP API. Requests are POSTed to this URL.
const APIURL = "https://paycheck-calculator.adp.com/api/pcc/v2/calculations"

// Request represents the request to the ADP API, encoded as JSON.
type Request struct {
	CalculationTypeCode   CalculationTypeCode    `json:"calculationTypeCode"`
	StatutoryPolicyInputs []StatutoryPolicyInput `json:"statutoryPolicyInputs"`
	Jurisdictions         Jurisdictions          `json:"jurisdictions"`
	PayDate               string                 `json:"payDate"`
	PayFrequencyCode      PayFrequencyCode       `json:"payFrequencyCode"`
	BusinessPolicies      []BusinessPolicy       `json:"businessPolicies"`
	AdditionalEarnings    AdditionalEarnings     `json:"additionalEarnings"`
	Deductions            []struct{}             `json:"deductions"`
}

// CalculationTypeCode represents the calculation type code in the ADP API. Should always be GrossToNetTypeCode.
type CalculationTypeCode struct {
	Code string `json:"code"`
}

var (
	// GrossToNetTypeCode is the calculation type code for gross to net income calculations.
	GrossToNetTypeCode = CalculationTypeCode{Code: "GROSS_TO_NET"}
)

// StatutoryPolicyInput is an input to the calculation that specifies options like filing status, withholding, etc.
type StatutoryPolicyInput struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Value      interface{} `json:"value"`
	Type       string      `json:"type"`
	TemplateID string      `json:"templateID"`
}

var (
	// StatutoryPolicy2020W4 is the statutory policy input that tells the calculation to use the 2020 and later W4
	// form. If not specified, the calculation will use the 2019 and earlier W4 form.
	StatutoryPolicy2020W4 = StatutoryPolicyInput{
		ID:         "w4Form2020Indicator",
		Name:       "w4Form2020Indicator",
		Value:      true,
		Type:       "boolean",
		TemplateID: "e01a6863-4fc7-4c2a-ac8c-f8d896c6fba2",
	}
)

// Jurisdictions represents the jurisdictions that the calculation should be done for. This separates lived in and
// worked in jurisdictions.
type Jurisdictions struct {
	WorkedInJurisdictions []*jurisdiction.Jurisdiction `json:"workedInJurisdictions"`
	LivedInJurisdictions  []*jurisdiction.Jurisdiction `json:"livedInJurisdictions"`
}

// PayFrequencyCode determines how frequently the payments in the response are assumed to be. For example, monthly will
// show the net income if the income is paid monthly.
type PayFrequencyCode struct {
	Code string `json:"code"`
}

var (
	// MonthlyPayFrequencyCode is the pay frequency code for monthly payments.
	MonthlyPayFrequencyCode = PayFrequencyCode{Code: "MONTHLY"}
	// SemiMonthlyPayFrequencyCode is the pay frequency code for semi-monthly payments.
	SemiMonthlyPayFrequencyCode = PayFrequencyCode{Code: "SEMI_MONTHLY"}
	// BiWeeklyPayFrequencyCode is the pay frequency code for bi-weekly payments.
	BiWeeklyPayFrequencyCode = PayFrequencyCode{Code: "BI_WEEKLY"}
	// WeeklyPayFrequencyCode is the pay frequency code for weekly payments.
	WeeklyPayFrequencyCode = PayFrequencyCode{Code: "WEEKLY"}
)

func (pfc PayFrequencyCode) String() string {
	switch pfc {
	case MonthlyPayFrequencyCode:
		return "monthly"
	case SemiMonthlyPayFrequencyCode:
		return "semi-monthly"
	case BiWeeklyPayFrequencyCode:
		return "bi-weekly"
	case WeeklyPayFrequencyCode:
		return "weekly"
	default:
		glog.V(10).Infof("Invalid pay frequency being converted to string: %+v", pfc)

		return ""
	}
}

// Set sets the pay frequency code from a string. It is necessary to implement the [flag.Value] interface. It does not
// return an error and instead will default to monthly if the value is not recognized.
func (pfc *PayFrequencyCode) Set(value string) error {
	switch value {
	case "monthly":
		*pfc = MonthlyPayFrequencyCode
	case "semi-monthly":
		*pfc = SemiMonthlyPayFrequencyCode
	case "biweekly":
		*pfc = BiWeeklyPayFrequencyCode
	case "weekly":
		*pfc = WeeklyPayFrequencyCode
	default:
		glog.V(10).Infof("Invalid pay frequency being set: %s", value)

		*pfc = MonthlyPayFrequencyCode
	}

	return nil
}

// BusinessPolicy is the input that determines the gross income. Depending on its label, the inputs field will be
// different.
type BusinessPolicy struct {
	ID     string                `json:"id"`
	Alias  string                `json:"alias"`
	Label  string                `json:"label"`
	Inputs []BusinessPolicyInput `json:"inputs"`
}

// SalaryFrequency is the frequency of the salary, either annual or periodic.
type SalaryFrequency string

const (
	// AnnualSalaryFrequency is the salary frequency for annual salary. The value of this constant should be
	// considered opaque.
	AnnualSalaryFrequency SalaryFrequency = "salary"
	// PeriodicSalaryFrequency is the salary frequency for periodic salary. The value of this constant should be
	// considered opaque.
	PeriodicSalaryFrequency SalaryFrequency = "salary_per_period"
)

func (f SalaryFrequency) String() string {
	switch f {
	case AnnualSalaryFrequency:
		return "annual"
	case PeriodicSalaryFrequency:
		return "periodic"
	default:
		glog.V(10).Infof("Invalid salary frequency being converted to string: %s", string(f))

		return ""
	}
}

func (f SalaryFrequency) validate() error {
	switch f {
	case AnnualSalaryFrequency, PeriodicSalaryFrequency:
		return nil
	default:
		return fmt.Errorf("invalid salary frequency: %s", f)
	}
}

// BusinessPolicyInput is one of the inputs to the business policy. For example, for salary, it is the amount of the
// salary, but for hourly, there is a separate input for the rate and the number of hours.
type BusinessPolicyInput struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
	Type  string      `json:"type"`
}

// newSalaryBusinessPolicy creates a new salary business policy with the given amount and frequency. The index is used
// to create a unique ID for the business policy and should start at 1.
func newSalaryBusinessPolicy(amount float64, frequency SalaryFrequency, index int) BusinessPolicy {
	return BusinessPolicy{
		ID:     fmt.Sprintf("salary-%d", index),
		Alias:  string(frequency),
		Label:  "SALARY",
		Inputs: []BusinessPolicyInput{{Name: "appliedPayPeriodAmount", Value: amount, Type: "amount"}},
	}
}

// newHourlyBusinessPolicy creates a new hourly business policy with the given amount and hours. The index is used to
// create a unique ID for the business policy and should start at 1.
func newHourlyBusinessPolicy(amount float64, hours float64, index int) BusinessPolicy {
	return BusinessPolicy{
		ID:    fmt.Sprintf("hourly-%d", index),
		Alias: "hourly",
		Label: "HOURLY",
		Inputs: []BusinessPolicyInput{
			{Name: "appliedHourlyRate", Value: amount, Type: "rate"},
			{Name: "regularHoursWorked", Value: hours, Type: "quantity"},
		},
	}
}

// AdditionalEarnings represents the additional earnings such as overtime or double time.
type AdditionalEarnings struct {
	PayLines []PayLine `json:"payLines"`
}

// PayLine represents a line in the additional earnings such as the overtime pay. Unit is the number of hours worked and
// amount is hourly rate prior to any client factor.
type PayLine struct {
	EarningType  EarningType   `json:"earningType"`
	Unit         PayLineUnit   `json:"unit"`
	Amount       PayLineAmount `json:"amount"`
	Name         PayLineName   `json:"name"`
	ClientFactor ClientFactor  `json:"clientFactor"`
}

func newOvertimePayLine(hours, rate float64) PayLine {
	return PayLine{
		EarningType:  OvertimeEarningType,
		Unit:         newPayLineUnit(hours),
		Amount:       PayLineAmount{Value: rate},
		Name:         OvertimePayLineName,
		ClientFactor: OvertimeClientFactor,
	}
}

func newDoubleTimePayLine(hours, rate float64) PayLine {
	return PayLine{
		EarningType:  DoubleTimeEarningType,
		Unit:         newPayLineUnit(hours),
		Amount:       PayLineAmount{Value: rate},
		Name:         DoubleTimePayLineName,
		ClientFactor: DoubletimeClientFactor,
	}
}

// EarningType represents the type of earning such as overtime or double time.
type EarningType struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

var (
	// OvertimeEarningType is the earning type for overtime pay.
	OvertimeEarningType = EarningType{
		Value: "OvertimePay",
		Label: "OVERTIME",
		Type:  "HUR",
	}
	// DoubleTimeEarningType is the earning type for double time pay.
	DoubleTimeEarningType = EarningType{
		Value: "DoubletimePay",
		Label: "DOUBLE_TIME",
		Type:  "HUR",
	}
)

// PayLineUnit is the number of units of earning, usually hours. It is the string representation of a float.
type PayLineUnit struct {
	Value string `json:"value"`
}

func newPayLineUnit(value float64) PayLineUnit {
	return PayLineUnit{Value: strconv.FormatFloat(value, 'f', 2, 64)}
}

// PayLineAmount is the amount of earning per unit.
type PayLineAmount struct {
	Value float64 `json:"value"`
}

// PayLineName is the name of the earning.
type PayLineName struct {
	Value string `json:"value"`
}

var (
	// OvertimePayLineName is the name of the overtime pay.
	OvertimePayLineName = PayLineName{
		Value: "Overtime",
	}
	// DoubleTimePayLineName is the name of the double time pay.
	DoubleTimePayLineName = PayLineName{
		Value: "Double time",
	}
)

// ClientFactor is the factor that is multiplied to the earning. For example, for overtime, the client factor is 1.5.
type ClientFactor struct {
	Value float64 `json:"value"`
}

var (
	// OvertimeClientFactor is the client factor for overtime pay.
	OvertimeClientFactor = ClientFactor{
		Value: 1.5,
	}
	// DoubletimeClientFactor is the client factor for double time pay.
	DoubletimeClientFactor = ClientFactor{
		Value: 2,
	}
)
