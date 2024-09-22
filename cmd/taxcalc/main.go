/*
Taxcalc calculates the income tax for a salary. It takes a salary as a command line argument and calculates the net
income less tax per pay period.

Usage:

	taxcalc [flags] salary

The flags are:

	-s, -state string
	        Calculate income tax for state in addition to federal income tax. This is a two letter abbreviation.

	-p, -pay-frequency string
	        Output net income per pay frequency. Must be one of monthly, bi-weekly, weekly, or semi-monthly. If not
	        specified, the default is monthly.

	-h, -help
	        Print this help message.
*/
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/tslnc04/tax-calculator/internal/request"
)

//nolint:lll
const usage = `Taxcalc calculates the income tax for a salary. It takes a salary as a command line argument and calculates the net
income less tax per pay period.

Usage:

	taxcalc [flags] salary

The flags are:

	-s, -state string
	        Calculate income tax for state in addition to federal income tax. This is a two letter abbreviation.

	-p, -pay-frequency string
	        Output net income per pay frequency. Must be one of monthly, bi-weekly, weekly, or semi-monthly. If not
	        specified, the default is monthly.

	-h, -help
	        Print this help message.
`

var (
	help         bool
	state        string
	payFrequency request.PayFrequencyCode
)

func init() {
	const (
		helpUsage         = "print this help message"
		stateUsage        = "state to calculate income tax for as a two letter abbreviation"
		payFrequencyUsage = "pay frequency to use, either monthly, bi-weekly, weekly, or semi-monthly"
	)

	flag.BoolVar(&help, "help", false, helpUsage)
	flag.BoolVar(&help, "h", false, helpUsage+" (shorthand)")

	flag.StringVar(&state, "state", "", stateUsage)
	flag.StringVar(&state, "s", "", stateUsage+" (shorthand)")

	flag.Var(&payFrequency, "pay-frequency", payFrequencyUsage)
	flag.Var(&payFrequency, "p", payFrequencyUsage+" (shorthand)")

	// Tell glog to log to stderr.
	_ = flag.Set("logtostderr", "true")
}

func main() {
	flag.Parse()

	if help {
		fmt.Print(usage)

		os.Exit(0)
	}

	if flag.NArg() != 1 {
		glog.Error("salary must be specified")
		fmt.Print(usage)

		os.Exit(2)
	}

	salary, err := strconv.ParseFloat(flag.Arg(0), 64)
	if err != nil {
		glog.Errorf("failed to parse salary: %s", err)

		os.Exit(2)
	}

	builder := request.NewBuilder().WithSalary(salary, request.AnnualSalaryFrequency).WithPayFrequency(payFrequency)

	if state != "" {
		state = strings.ToUpper(state)

		glog.V(10).Infof("adding state: %s", state)

		builder.WithJurisdictionsByCode(state)
	}

	response, err := builder.Send()
	if err != nil {
		glog.Errorf("failed to send request: %s", err)

		os.Exit(2)
	}

	fmt.Printf("%.2f\n", response.Net.Amount)
}
