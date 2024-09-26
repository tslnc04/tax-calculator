/*
Taxcalcd is a web server that calculates the income tax for a salary. It takes a salary, pay frequency, and state as
query parameters and returns the net income in CSV format.

Usage:

	taxcalcd [flags]

The flags are:

	-c, -cache_size int
		Number of entries to keep in the response cache. Defaults to 1000.

	-h, -help
		Print this help message.

	-log_dir string
		Directory to write logs to. Defaults to a temporary directory.

	-p, -port string
		Port to listen on. Defaults to 8080.

	-v int
		Maximum log verbosity. Defaults to 0.
*/
package main

import (
	"flag"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/tslnc04/tax-calculator/internal/server"
)

//nolint:lll
const usage = `Taxcalcd is a web server that calculates the income tax for a salary. It takes a salary, pay frequency, and state as
query parameters and returns the net income in CSV format.

Usage:

	taxcalcd [flags]

The flags are:

	-c, -cache_size int
		Number of entries to keep in the response cache. Defaults to 1000.

	-h, -help
		Print this help message.

	-log_dir string
		Directory to write logs to. Defaults to a temporary directory.

	-p, -port string
		Port to listen on. Defaults to 8080.

	-r, -rate_limit duration
		Requests to the ADP API are rate limited to one per this duration. Defaults to 1s.

	-v int
		Maximum log verbosity. Defaults to 0.
`

var (
	cacheSize int
	help      bool
	port      string
	rateLimit time.Duration
)

func init() {
	const (
		cacheUsage     = "number of entries to keep in the response cache"
		helpUsage      = "print this help message"
		portUsage      = "port to listen on"
		rateLimitUsage = "requests to the ADP API are rate limited to one per this duration"

		defaultCacheSize = 1000
		defaultHelp      = false
		defaultPort      = ":8080"
		defaultRateLimit = time.Second
	)

	flag.IntVar(&cacheSize, "cache_size", defaultCacheSize, cacheUsage)
	flag.IntVar(&cacheSize, "c", defaultCacheSize, cacheUsage+" (shorthand)")

	flag.BoolVar(&help, "help", defaultHelp, helpUsage)
	flag.BoolVar(&help, "h", defaultHelp, helpUsage+" (shorthand)")

	flag.StringVar(&port, "port", defaultPort, portUsage)
	flag.StringVar(&port, "p", defaultPort, portUsage+" (shorthand)")

	flag.DurationVar(&rateLimit, "rate_limit", defaultRateLimit, rateLimitUsage)
	flag.DurationVar(&rateLimit, "r", defaultRateLimit, rateLimitUsage+" (shorthand)")

	// Tell glog to log to stderr as well as the log file.
	_ = flag.Set("alsologtostderr", "true")
}

func main() {
	flag.Parse()

	if help {
		print(usage)

		return
	}

	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	mux, err := server.NewRequestMux(cacheSize, rateLimit)
	if err != nil {
		glog.Errorf("Failed to create request mux: %s", err)

		os.Exit(2)
	}

	glog.V(10).Infof("Starting server on port %s", port)

	err = http.ListenAndServe(port, mux)
	if err != nil {
		glog.Errorf("Failed to start server: %s", err)

		os.Exit(2)
	}
}
