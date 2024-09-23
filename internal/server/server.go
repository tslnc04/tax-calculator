// Package server provides the handler for the taxcalcd web server. It is responsible for parsing the request and
// sending it to the tax calculator.
package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/golang/glog"
	"github.com/tslnc04/tax-calculator/internal/request"
)

// APIBasePath is the base path for the API. All paths are relative to this.
const APIBasePath = "/api/v1"

// HandleRequest handles a request for calculating the net income. It expects the salary to be specified in the query
// string as a float and the pay frequency and state as strings. It will return a CSV response with the net income.
func HandleRequest(resp http.ResponseWriter, req *http.Request) {
	glog.V(10).Infof("Handling request from %s to URL `%s` with pattern %s", req.RemoteAddr, req.URL, req.Pattern)

	rawSalary := req.URL.Query().Get("salary")

	if rawSalary == "" {
		glog.V(10).Info("Salary was not specified in request")

		http.Error(resp, "salary must be specified", http.StatusBadRequest)

		return
	}

	salary, err := strconv.ParseFloat(rawSalary, 64)
	if err != nil {
		glog.V(10).Infof("Failed to parse salary: %s", err)

		http.Error(resp, fmt.Sprintf("salary is not a valid float: %s", err), http.StatusBadRequest)

		return
	}

	payFrequency := req.URL.Query().Get("pay-frequency")
	payFrequencyCode := request.PayFrequencyCode{}
	_ = payFrequencyCode.Set(payFrequency)

	builder := request.NewBuilder().WithSalary(salary, request.AnnualSalaryFrequency).WithPayFrequency(payFrequencyCode)

	state := req.URL.Query().Get("state")
	if state != "" {
		glog.V(10).Infof("Adding state to request: %s", state)

		builder.WithJurisdictionsByCode(state)
	}

	response, err := builder.Send()
	if err != nil {
		glog.V(10).Infof("Failed to send request to ADP API: %s", err)

		http.Error(resp, fmt.Sprintf("failed to send request: %s", err), http.StatusInternalServerError)

		return
	}

	glog.V(10).Infof(
		"Responding with %.2f to request with salary=%.2f, pay-frequency=%s, state=%s",
		response.Net.Amount, salary, payFrequency, state,
	)

	resp.Header().Set("Content-Type", "text/csv")
	resp.WriteHeader(http.StatusOK)

	fmt.Fprintf(resp, "%.2f\n", response.Net.Amount)
}

// HandleHealthCheck handles a health check request. It returns a 204 No Content response.
func HandleHealthCheck(resp http.ResponseWriter, req *http.Request) {
	glog.V(10).Infof("Handling health check request from %s to URL `%s`", req.RemoteAddr, req.URL, req.Pattern)

	resp.WriteHeader(http.StatusNoContent)
}
