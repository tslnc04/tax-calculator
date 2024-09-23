package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/tslnc04/tax-calculator/internal/jurisdiction"
	"github.com/tslnc04/tax-calculator/internal/response"
)

// Builder is a builder for the request to the ADP API. The zero value is not sendable and must have at least one salary
// or hourly income source added before sending.
type Builder struct {
	URL              string
	payFrequencyCode *PayFrequencyCode
	jurisdictions    []*jurisdiction.Jurisdiction
	salaries         []BusinessPolicy
	hourlies         []BusinessPolicy
	overtime         []PayLine
	doubletime       []PayLine
	errorMessage     string
}

// NewBuilder creates a new builder with the given URL. If no URL is given, the [APIURL] is used. Generally, you should
// not need to specify the URL.
func NewBuilder(url ...string) *Builder {
	if len(url) < 1 {
		url = append(url, APIURL)
	}

	firstURL := url[0]

	glog.V(10).Infof("Initializing builder with url=`%s`", firstURL)

	return &Builder{
		URL: firstURL,
	}
}

// WithPayFrequency sets the pay frequency code for the calculation. If this is not set, the default is monthly.
func (builder *Builder) WithPayFrequency(payFrequencyCode PayFrequencyCode) *Builder {
	if err := builder.validate(); err != nil {
		return builder
	}

	glog.V(10).Infof("Setting pay frequency to %s", payFrequencyCode)

	builder.payFrequencyCode = &payFrequencyCode

	return builder
}

// WithJurisdictions adds to both the lived in and worked in jurisdictions for the calculation. If this is not called,
// the default is just federal.
func (builder *Builder) WithJurisdictions(jurisdictions ...*jurisdiction.Jurisdiction) *Builder {
	if err := builder.validate(); err != nil {
		return builder
	}

	glog.V(10).Infof("Adding %d jurisdictions", len(jurisdictions))

	builder.jurisdictions = append(builder.jurisdictions, jurisdictions...)

	return builder
}

// WithJurisdictionsByCode adds jurisdictions to the calculation by their codes. This has the side effect of attempting
// to dynamically load the jurisdictions by code if [jurisdiction.JurisdictionsByCode] is empty. If a code is not found,
// the builder will not be modified except to signal an error.
func (builder *Builder) WithJurisdictionsByCode(jurisdictionCodes ...string) *Builder {
	if err := builder.validate(); err != nil {
		return builder
	}

	glog.V(10).Infof("Adding %d jurisdictions by code", len(jurisdictionCodes))

	if len(jurisdiction.JurisdictionsByCode) < 1 {
		glog.V(10).Infof("No jurisdictions loaded, attempting to load now")

		_, err := jurisdiction.LoadJurisdictions()
		if err != nil {
			glog.V(10).Infof("Failed to load jurisdictions: %s", err)

			builder.errorMessage = err.Error()

			return builder
		}
	}

	// We store the jurisdictions before appending them to the builder so the builder remains unchanged if this
	// method sets an error.
	var jurisdictions []*jurisdiction.Jurisdiction

	for _, code := range jurisdictionCodes {
		jurisdiction, ok := jurisdiction.JurisdictionsByCode[code]
		if !ok {
			glog.V(10).Infof("No jurisdiction found for code: %s", code)

			builder.errorMessage = fmt.Sprintf("no jurisdiction found for code: %s", code)

			return builder
		}

		jurisdictions = append(jurisdictions, jurisdiction)
	}

	builder.jurisdictions = append(builder.jurisdictions, jurisdictions...)

	return builder
}

// WithSalary adds a salary to the income sources. The amount is in dollars and per the frequency.
func (builder *Builder) WithSalary(amount float64, frequency SalaryFrequency) *Builder {
	if err := builder.validate(); err != nil {
		return builder
	}

	glog.V(10).Infof("Adding salary of %.2f with frequency %s", amount, frequency)

	if amount < 0 {
		glog.V(10).Infof("Salary amount is negative")

		builder.errorMessage = "salary amount must be non-negative"

		return builder
	}

	if err := frequency.validate(); err != nil {
		glog.V(10).Infof("Salary frequency is invalid: %s", err)

		builder.errorMessage = err.Error()

		return builder
	}

	salaryBusinessPolicy := newSalaryBusinessPolicy(amount, frequency, len(builder.salaries)+1)
	builder.salaries = append(builder.salaries, salaryBusinessPolicy)

	return builder
}

// WithHourly adds an hourly rate to the income sources. Rate is in dollars per hour.
func (builder *Builder) WithHourly(hours, rate float64) *Builder {
	if err := builder.validate(); err != nil {
		return builder
	}

	glog.V(10).Infof("Adding hourly rate of %.2f with %.2f hours", rate, hours)

	if hours < 0 {
		glog.V(10).Infof("Hourly hours is negative: %.2f", hours)

		builder.errorMessage = "hourly hours must be non-negative"

		return builder
	}

	if rate < 0 {
		glog.V(10).Infof("Hourly rate is negative: %.2f", rate)

		builder.errorMessage = "hourly rate must be non-negative"

		return builder
	}

	hourlyBusinessPolicy := newHourlyBusinessPolicy(rate, hours, len(builder.hourlies)+1)
	builder.hourlies = append(builder.hourlies, hourlyBusinessPolicy)

	return builder
}

// WithOvertime adds an overtime pay line to the additional earnings. Rate is in dollars per hour.
func (builder *Builder) WithOvertime(hours, rate float64) *Builder {
	if err := builder.validate(); err != nil {
		return builder
	}

	glog.V(10).Infof("Adding overtime rate of %.2f with %.2f hours", rate, hours)

	if hours < 0 {
		glog.V(10).Infof("Overtime hours is negative: %.2f", hours)

		builder.errorMessage = "overtime hours must be non-negative"

		return builder
	}

	if rate < 0 {
		glog.V(10).Infof("Overtime rate is negative: %.2f", rate)

		builder.errorMessage = "overtime rate must be non-negative"

		return builder
	}

	overtimePayLine := newOvertimePayLine(hours, rate)
	builder.overtime = append(builder.overtime, overtimePayLine)

	return builder
}

// WithDoubleTime adds a double time pay line to the additional earnings. Rate is in dollars per hour.
func (builder *Builder) WithDoubleTime(hours, rate float64) *Builder {
	if err := builder.validate(); err != nil {
		return builder
	}

	glog.V(10).Infof("Adding double time rate of %.2f with %.2f hours", rate, hours)

	if hours < 0 {
		glog.V(10).Infof("Double time hours is negative: %.2f", hours)

		builder.errorMessage = "double time hours must be non-negative"

		return builder
	}

	if rate < 0 {
		glog.V(10).Infof("Double time rate is negative: %.2f", rate)

		builder.errorMessage = "double time rate must be non-negative"

		return builder
	}

	doubletimePayLine := newDoubleTimePayLine(hours, rate)
	builder.doubletime = append(builder.doubletime, doubletimePayLine)

	return builder
}

// HandleError consumes the error message and returns it as an error. If there is no error message, this returns nil.
// The builder is guaranteed to be in a valid (but not necessarily sendable) state after this.
func (builder *Builder) HandleError() error {
	if builder.errorMessage == "" {
		return nil
	}

	err := fmt.Errorf(builder.errorMessage)
	builder.errorMessage = ""

	return err
}

// Send sends the request to the ADP API and returns a parsed [response.Response]. This does not modify the builder. If
// there is an error validating or sending the request, this returns an error.
func (builder *Builder) Send() (*response.Response, error) {
	if err := builder.validate(); err != nil {
		return nil, err
	}

	glog.V(10).Infof("Sending request to %s", builder.URL)

	requestJSON, err := json.Marshal(builder.buildRequest())
	if err != nil {
		glog.V(10).Infof("Failed to JSON marshal request to ADP API: %s", err)

		return nil, err
	}

	resp, err := http.Post(builder.URL, "application/json", bytes.NewBuffer(requestJSON))
	if err != nil {
		glog.V(10).Infof("Failed to send request to ADP API: %s", err)

		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		glog.V(10).Infof("Status was not OK sending request to ADP API: %s", resp.Status)

		return nil, fmt.Errorf("status was not OK sending request: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		glog.V(10).Infof("Failed to read response body from ADP API: %s", err)

		return nil, err
	}

	response := &response.Response{}

	err = json.Unmarshal(body, &response)
	if err != nil {
		glog.V(10).Infof("Failed to JSON unmarshal ADP API response: %s", err)

		return nil, err
	}

	return response, nil
}

// buildRequest builds the request to the ADP API. This is called by [Send] and should not be called directly. It
// performs no validation and does not modify the builder.
func (builder *Builder) buildRequest() *Request {
	glog.V(10).Infof("Building request to ADP API")

	payFrequency := builder.payFrequencyCode
	if payFrequency == nil {
		glog.V(10).Info("No pay frequency specified, defaulting to monthly")

		payFrequency = &MonthlyPayFrequencyCode
	}

	jurisdictions := append([]*jurisdiction.Jurisdiction{}, builder.jurisdictions...)
	hasFederal := false

	for _, jurisdiction := range jurisdictions {
		if jurisdiction.JurisdictionCode.Code == "US" {
			hasFederal = true

			break
		}
	}

	if !hasFederal {
		jurisdictions = append(jurisdictions, jurisdiction.GetFederalJurisdiction())
	}

	// Copy the slices and join them so that the builder remains unmodified.
	policies := make([]BusinessPolicy, len(builder.salaries))
	copy(policies, builder.salaries)
	copy(policies[len(builder.salaries):], builder.hourlies)

	payLines := make([]PayLine, len(builder.overtime)+len(builder.doubletime))
	copy(payLines, builder.overtime)
	copy(payLines[len(builder.overtime):], builder.doubletime)

	request := &Request{
		CalculationTypeCode:   GrossToNetTypeCode,
		StatutoryPolicyInputs: []StatutoryPolicyInput{StatutoryPolicy2020W4},
		Jurisdictions: Jurisdictions{
			LivedInJurisdictions:  jurisdictions,
			WorkedInJurisdictions: jurisdictions,
		},
		PayDate:            time.Now().Format(time.DateOnly),
		PayFrequencyCode:   *payFrequency,
		BusinessPolicies:   policies,
		AdditionalEarnings: AdditionalEarnings{PayLines: payLines},
		Deductions:         []struct{}{},
	}

	return request
}

// validate ensures that the builder is in a valid state. If there is an error message, it is returned. Otherwise, nil
// is returned. This does not guarantee that the builder is sendable nor is it guaranteed to be valid after this.
func (builder *Builder) validate() error {
	if builder.errorMessage != "" {
		return fmt.Errorf(builder.errorMessage)
	}

	return nil
}
