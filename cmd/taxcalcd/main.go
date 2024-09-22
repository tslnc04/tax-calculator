/*
Taxcalcd is a web server that calculates the income tax for a salary. It takes a salary, pay frequency, and state as
query parameters and returns the net income in CSV format.

Usage:

	taxcalcd [flags]

The flags are:

	-p, -port string
		Port to listen on. Defaults to 8080.

	-log_dir string
		Directory to write logs to. Defaults to a temporary directory.

	-v int
		Maximum log verbosity. Defaults to 0.

	-h, -help
		Print this help message.
*/
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/tslnc04/tax-calculator/internal/server"
)

//nolint:lll
const usage = `Taxcalcd is a web server that calculates the income tax for a salary. It takes a salary, pay frequency, and state as
query parameters and returns the net income in CSV format.

Usage:

	taxcalcd [flags]

The flags are:

	-p, -port string
		Port to listen on. Defaults to 8080.

	-log_dir string
		Directory to write logs to. Defaults to a temporary directory.

	-v int
		Maximum log verbosity. Defaults to 0.

	-h, -help
		Print this help message.
`

var (
	help bool
	port string
)

func init() {
	const (
		helpUsage   = "print this help message"
		portUsage   = "port to listen on"
		defaultPort = ":8080"
	)

	flag.BoolVar(&help, "help", false, helpUsage)
	flag.BoolVar(&help, "h", false, helpUsage+" (shorthand)")

	flag.StringVar(&port, "port", defaultPort, portUsage)
	flag.StringVar(&port, "p", defaultPort, portUsage+" (shorthand)")

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

	http.HandleFunc(fmt.Sprintf("GET %s", server.APIBasePath), server.HandleRequest)
	http.HandleFunc("/", server.HandleHealthCheck)

	glog.V(10).Infof("starting server on port %s", port)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		glog.Errorf("failed to start server: %s", err)

		os.Exit(2)
	}
}
