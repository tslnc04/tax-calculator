// Package server provides the handler for the taxcalcd web server. It is responsible for parsing the request and
// sending it to the tax calculator.
package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/golang/glog"
	lruv2 "github.com/hashicorp/golang-lru/v2"
	"github.com/tslnc04/tax-calculator/internal/request"
	"github.com/tslnc04/tax-calculator/internal/response"
)

const (
	// APIBasePath is the base path for the API. All paths are relative to this.
	APIBasePath = "/api/v1"
)

type responseCache = *lruv2.Cache[string, *response.Response]

// RequestHandler is a handler for the taxcalcd web server. It includes a cache for storing responses from the ADP API.
// Its zero value is not valid and must be initialized with [NewRequestHandler].
type RequestHandler struct {
	cache responseCache
}

// NewRequestHandler creates a new request handler with the given cache size. Each cached response will consume roughly
// 600 bytes.
func NewRequestHandler(cacheSize int) (*RequestHandler, error) {
	cache, err := lruv2.New[string, *response.Response](cacheSize)
	if err != nil {
		return nil, err
	}

	return &RequestHandler{cache: cache}, nil
}

// ServeHTTP handles a request for calculating the net income. It expects the salary to be specified in the query string
// as a float and the pay frequency and state as strings. It will return a CSV response with the net income.
func (handler *RequestHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	glog.V(10).Infof("Handling request from %s to URL `%s` with pattern %s", req.RemoteAddr, req.URL, req.Pattern)

	params, err := parseRequestParams(req.URL)
	if err != nil {
		glog.V(10).Infof("Failed to parse request params: %s", err)

		http.Error(resp, fmt.Sprintf("failed to parse request params: %s", err), http.StatusBadRequest)

		return
	}

	response, err := params.retrieveOrRequest(handler.cache)
	if err != nil {
		glog.V(10).Infof("Failed to retrieve or request: %s", err)

		http.Error(resp, fmt.Sprintf("failed to retrieve or request: %s", err), http.StatusInternalServerError)

		return
	}

	glog.V(10).Infof("Responding with %.2f to request with params %+v", response.Net.Amount, params)

	resp.Header().Set("Content-Type", "text/csv")
	resp.WriteHeader(http.StatusOK)

	fmt.Fprintf(resp, "%.2f\n", response.Net.Amount)
}

type requestParams struct {
	salary       float64
	payFrequency request.PayFrequencyCode
	state        string
}

// parseRequestParams parses the request parameters from the URL and returns a new requestParams struct.
func parseRequestParams(url *url.URL) (*requestParams, error) {
	salary := url.Query().Get("salary")
	if salary == "" {
		return nil, fmt.Errorf("salary must be specified")
	}

	salaryFloat, err := strconv.ParseFloat(salary, 64)
	if err != nil {
		return nil, fmt.Errorf("salary is not a valid float: %w", err)
	}

	payFrequency := url.Query().Get("pay-frequency")
	payFrequencyCode := request.PayFrequencyCode{}
	_ = payFrequencyCode.Set(payFrequency)

	state := url.Query().Get("state")

	return &requestParams{salaryFloat, payFrequencyCode, state}, nil
}

// getCacheKey returns a string representation of the parameters that can be used as a cache key.
func (params *requestParams) getCacheKey() string {
	return fmt.Sprintf("%.2f%s%s", params.salary, params.state, params.payFrequency)
}

// buildRequest creates a new request builder with the parameters from the request.
func (params *requestParams) buildRequest() *request.Builder {
	builder := request.NewBuilder().
		WithSalary(params.salary, request.AnnualSalaryFrequency).
		WithPayFrequency(params.payFrequency)

	if params.state != "" {
		glog.V(10).Infof("Adding state to request: %s", params.state)

		builder.WithJurisdictionsByCode(params.state)
	}

	return builder
}

// retrieveOrRequest attempts to retrieve a response from the cache or send a request to the ADP API.
func (params *requestParams) retrieveOrRequest(cache responseCache) (*response.Response, error) {
	cacheKey := params.getCacheKey()
	cachedResponse, ok := cache.Get(cacheKey)

	if ok {
		glog.V(10).Infof("Found entry in cache for key `%s`, using cached response", cacheKey)

		return cachedResponse, nil
	}

	glog.V(10).Infof("No entry in cache for key `%s`, sending request to ADP API", cacheKey)

	response, err := params.buildRequest().Send()
	if err != nil {
		glog.V(10).Infof("Failed to send request to ADP API: %s", err)

		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	cache.Add(cacheKey, response)

	return response, nil
}

// HandleHealthCheck handles a health check request. It always returns a 204 No Content response.
func HandleHealthCheck(resp http.ResponseWriter, req *http.Request) {
	glog.V(10).Infof("Handling health check request from %s to URL `%s`", req.RemoteAddr, req.URL, req.Pattern)

	resp.WriteHeader(http.StatusNoContent)
}
