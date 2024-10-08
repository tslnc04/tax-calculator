// Package server provides the handler for the taxcalcd web server. It is responsible for parsing the request and
// sending it to the tax calculator.
package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/golang/glog"
	lruv2 "github.com/hashicorp/golang-lru/v2"
	"github.com/tslnc04/tax-calculator/internal/request"
	"github.com/tslnc04/tax-calculator/internal/response"
	"golang.org/x/time/rate"
)

const (
	// APIBasePath is the base path for the API. All paths are relative to this.
	APIBasePath = "/api/v1"
)

// NewRequestMux attaches all the routes for the taxcalcd web server to a ServeMux. It returns the ServeMux and an error
// if one occurred.
func NewRequestMux(cacheSize int, rateLimit time.Duration) (*http.ServeMux, error) {
	requestHandler, err := NewRequestHandler(cacheSize, rateLimit)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()

	mux.Handle(APIBasePath+"/", requestHandler)
	mux.HandleFunc("/", HandleHealthCheck)

	return mux, nil
}

type responseCache = *lruv2.Cache[string, *response.Response]

// RequestHandler is a handler for the taxcalcd web server. It includes a cache for storing responses from the ADP API.
// Its zero value is not valid and must be initialized with [NewRequestHandler].
type RequestHandler struct {
	cache   responseCache
	limiter *rate.Limiter
}

// NewRequestHandler creates a new request handler with the given cache size and rate limit. Each cached response will
// consume roughly 600 bytes. Requests to the ADP API are rate limited to one per the given rate limit.
func NewRequestHandler(cacheSize int, rateLimit time.Duration) (*RequestHandler, error) {
	cache, err := lruv2.New[string, *response.Response](cacheSize)
	if err != nil {
		return nil, err
	}

	limiter := rate.NewLimiter(rate.Every(rateLimit), 1)
	handler := &RequestHandler{cache: cache, limiter: limiter}

	return handler, nil
}

// ServeHTTP handles a request for calculating the net income. It expects the salary to be specified in the query string
// as a float and the pay frequency and state as strings. It will return a CSV response with the net income.
func (handler *RequestHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	logRequest(req, "API")

	params, err := parseRequestParams(req.URL)
	if err != nil {
		glog.V(10).Infof("Failed to parse request params: %s", err)

		http.Error(resp, fmt.Sprintf("failed to parse request params: %s", err), http.StatusBadRequest)

		return
	}

	response, err := params.retrieveOrRequest(handler.cache, handler.limiter)
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

// retrieveOrRequest attempts to retrieve a response from the cache or send a request to the ADP API. It will rate limit
// requests to the ADP API.
func (params *requestParams) retrieveOrRequest(cache responseCache, limiter *rate.Limiter) (*response.Response, error) {
	cacheKey := params.getCacheKey()
	cachedResponse, ok := cache.Get(cacheKey)

	if ok {
		glog.V(10).Infof("Found entry in cache for key `%s`, using cached response", cacheKey)

		return cachedResponse, nil
	}

	glog.V(10).Infof("No entry in cache for key `%s`, waiting for rate limit", cacheKey)

	err := limiter.Wait(context.Background())
	if err != nil {
		glog.V(10).Infof("Failed to wait for rate limit: %s", err)

		return nil, fmt.Errorf("failed to wait for rate limit: %w", err)
	}

	glog.V(10).Info("Successfully waited for rate limit, sending request to ADP API")

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
	logRequest(req, "health check")

	resp.WriteHeader(http.StatusNoContent)
}

// logRequest logs an incoming request for the endpoint with a given description. If the request has a `X-Forwarded-For`
// header, it will log the IP address from that header. Otherwise, it will log the IP address from the `RemoteAddr`
// field.
func logRequest(req *http.Request, description string) {
	if req.Header.Get("X-Forwarded-For") != "" {
		glog.V(10).Infof(
			"Handling %s request from %s to URL `%s` matching pattern %s",
			description, req.Header.Get("X-Forwarded-For"), req.URL, req.Pattern)

		return
	}

	glog.V(10).Infof(
		"Handling %s request from %s to URL `%s` matching pattern %s",
		description, req.RemoteAddr, req.URL, req.Pattern)
}
