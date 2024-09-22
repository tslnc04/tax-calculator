// Package jurisdiction contains the types for tax jurisdictions in the ADP API and functions for loading them
// dynamically.
package jurisdiction

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

// JurisdictionsByCode is a map of jurisdiction codes to jurisdictions. This is dynamically loaded from the ADP API when
// [LoadJurisdictions] is called.
var JurisdictionsByCode = map[string]*Jurisdiction{}

const (
	pwcBaseURL        = "https://pwc.adp.com"
	loaderPath        = "/pwc/dist/loader.js"
	dynamicPathFormat = "/pwc/dist/pcc/%s/esm/pwc-dynamic-control-generator_20.entry.js"
)

var (
	loaderVersionRegex     = regexp.MustCompile(`const [[:alpha:]]=JSON.parse\('(.+)'\)`)
	stateJurisdictionRegex = regexp.MustCompile(
		`info = \{\s*shortName: '(.*?)',\s*longName: '(.*?)',\s*jurisdictionID: '(.*?)'\s*\};`)
	federalJurisdictionRegex = regexp.MustCompile(`const FEDERAL_JURISDICTION = (\{[\S\s]*?\});`)
	unquotedKeyRegex         = regexp.MustCompile(`\s*([[:alpha:]]+):`)
)

// Jurisdiction represents a tax jurisdiction in the ADP API.
type Jurisdiction struct {
	JurisdictionID        string    `json:"jurisdictionID"`
	JurisdictionCode      Code      `json:"jurisdictionCode"`
	JurisdictionLevelCode LevelCode `json:"jurisdictionLevelCode"`
}

// Code represents a tax jurisdiction code in the ADP API. This is a long name and a short name.
type Code struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// LevelCode represents a tax jurisdiction level code in the ADP API. The levels are federal, state, etc.
type LevelCode struct {
	Code string `json:"code"`
}

// FallbackFederalJurisdiction is the federal jurisdiction from version 2024.24.0 of the ADP API. It is always preferred
// to dynamically load jurisdictions, but this provides a fallback. The version of the API this is from is subject to
// change.
var FallbackFederalJurisdiction = &Jurisdiction{
	JurisdictionID:        "dea07e6d-9432-4f65-958b-25f09e18117e",
	JurisdictionCode:      Code{Name: "United States Federal", Code: "US"},
	JurisdictionLevelCode: LevelCode{Code: "FEDERAL"},
}

// LoadJurisdictions uses the JS loader to find the correct version of the API and parses the jurisdictions.
func LoadJurisdictions() ([]*Jurisdiction, error) {
	loaderBytes, err := getLoader(pwcBaseURL + loaderPath)
	if err != nil {
		return nil, err
	}

	pccVersion, err := getPCCVersion(loaderBytes)
	if err != nil {
		return nil, err
	}

	pccDynamicBytes, err := getPCCDynamic(pwcBaseURL + fmt.Sprintf(dynamicPathFormat, pccVersion))
	if err != nil {
		return nil, err
	}

	jurisdictions, err := parseStateJurisdictions(pccDynamicBytes)
	if err != nil {
		return nil, err
	}

	federalJurisdiction, err := parseFederalJurisdiction(pccDynamicBytes)
	if err != nil {
		return nil, err
	}

	jurisdictions = append(jurisdictions, federalJurisdiction)

	populateJurisdictionsByCode(jurisdictions)

	return jurisdictions, nil
}

// GetFederalJurisdiction returns the federal jurisdiction. If it has not been loaded, it returns the fallback.
func GetFederalJurisdiction() *Jurisdiction {
	if len(JurisdictionsByCode) < 1 {
		return FallbackFederalJurisdiction
	}

	federal, ok := JurisdictionsByCode["US"]
	if !ok {
		return FallbackFederalJurisdiction
	}

	return federal
}

func getLoader(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status was not OK getting loader: %s", resp.Status)
	}

	loaderBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return loaderBytes, nil
}

type loaderVersions struct {
	RC string            `json:"RC"`
	GA map[string]string `json:"GA"`
}

func getPCCVersion(loaderBytes []byte) (string, error) {
	matches := loaderVersionRegex.FindSubmatch(loaderBytes)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not find version in loader")
	}

	versionsJSON := matches[1]

	loaderVersions := &loaderVersions{}
	err := json.Unmarshal(versionsJSON, loaderVersions)

	if err != nil {
		return "", err
	}

	version, ok := loaderVersions.GA["pcc"]
	if !ok {
		return "", fmt.Errorf("could not find pcc version in loader")
	}

	return version, nil
}

func getPCCDynamic(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status was not OK getting pcc dynamic: %s", resp.Status)
	}

	pccDynamicBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return pccDynamicBytes, nil
}

func parseStateJurisdictions(pccDynamicBytes []byte) ([]*Jurisdiction, error) {
	matches := stateJurisdictionRegex.FindAllSubmatch(pccDynamicBytes, -1)
	if len(matches) < 1 {
		return nil, fmt.Errorf("could not find state jurisdictions in pcc dynamic")
	}

	var jurisdictions []*Jurisdiction

	for _, match := range matches {
		jurisdiction := &Jurisdiction{
			JurisdictionID: string(match[3]),
			JurisdictionCode: Code{
				Name: string(match[2]),
				Code: string(match[1]),
			},
			JurisdictionLevelCode: LevelCode{
				Code: "STATE",
			},
		}

		jurisdictions = append(jurisdictions, jurisdiction)
	}

	return jurisdictions, nil
}

func parseFederalJurisdiction(pccDynamicBytes []byte) (*Jurisdiction, error) {
	matches := federalJurisdictionRegex.FindSubmatch(pccDynamicBytes)
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not find federal jurisdiction in pcc dynamic")
	}

	quoted := unquotedKeyRegex.ReplaceAll(matches[1], []byte(`"$1":`))
	quoted = bytes.ReplaceAll(quoted, []byte(`'`), []byte(`"`))

	jurisdiction := &Jurisdiction{}
	err := json.Unmarshal(quoted, jurisdiction)

	if err != nil {
		return nil, err
	}

	return jurisdiction, nil
}

func populateJurisdictionsByCode(jurisdictions []*Jurisdiction) {
	for _, jurisdiction := range jurisdictions {
		JurisdictionsByCode[jurisdiction.JurisdictionCode.Code] = jurisdiction
	}
}
