package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

const (
	abuseIpDbCheckerBaseUrl = "https://api.abuseipdb.com/api/v2/check"
)

type AbuseIpDbChecker struct {
	httpClient *http.Client
	apiKey     string
}

func NewAbuseIpDbChecker(apiKey string) *AbuseIpDbChecker {
	return &AbuseIpDbChecker{
		httpClient: &http.Client{Timeout: 500 * time.Millisecond},
		apiKey:     apiKey,
	}
}

func (c AbuseIpDbChecker) Enrich(ip net.IP, out *LookupResult) error {
	params := url.Values{}
	params.Add("ipAddress", ip.String())
	params.Add("maxAgeInDays", "90")
	params.Add("verbose", "")

	abuseIpDbCheckerUrl := abuseIpDbCheckerBaseUrl + "?" + params.Encode()
	req, err := http.NewRequest(http.MethodGet, abuseIpDbCheckerUrl, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Key", c.apiKey)
	req.Header.Add("Accept", "application/json")

	httpResponse, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != 200 {
		return fmt.Errorf("http status %d", httpResponse.StatusCode)
	}

	httpBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return err
	}

	var res AbuseIpDbCheckResult
	if err := json.Unmarshal(httpBody, &res); err != nil {
		return err
	}

	out.Risk = RiskInfo{
		AbuseConfidenceScore:  res.Data.AbuseConfidenceScore,
		UsageType:             res.Data.UsageType,
		IsTor:                 res.Data.IsTor,
		TotalReports:          res.Data.TotalReports,
		NumberOfUsersReported: res.Data.NumDistinctUsers,
		LastReportedAt:        res.Data.LastReportedAt,
	}

	return nil
}
