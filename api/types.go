package api

import (
	"net"
	"time"
)

type LookupResult struct {
	IP       string       `json:"ip"`
	ISP      ISPInfo      `json:"isp"`
	Location LocationInfo `json:"location"`
	Risk     RiskInfo     `json:"risk"`
}

type ISPInfo struct {
	ASN string `json:"asn"`
	Org string `json:"org"`
	ISP string `json:"isp"`
}

type LocationInfo struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city"`
	State       string  `json:"state"`
	Zipcode     string  `json:"zipcode"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Timezone    string  `json:"timezone"`
	Localtime   string  `json:"localtime"`
}

type RiskInfo struct {
	AbuseConfidenceScore  int       `json:"abuse_confidence_score"`
	UsageType             string    `json:"usage_type"`
	IsTor                 bool      `json:"is_tor"`
	TotalReports          int       `json:"total_reports"`
	NumberOfUsersReported int       `json:"number_of_users_reported"`
	LastReportedAt        time.Time `json:"last_reported_at"`
}

type AbuseIpDbCheckResult struct {
	Data struct {
		IpAddress            string        `json:"ipAddress"`
		IsPublic             bool          `json:"isPublic"`
		IpVersion            int           `json:"ipVersion"`
		IsWhitelisted        bool          `json:"isWhitelisted"`
		AbuseConfidenceScore int           `json:"abuseConfidenceScore"`
		CountryCode          string        `json:"countryCode"`
		CountryName          string        `json:"countryName"`
		UsageType            string        `json:"usageType"`
		Isp                  string        `json:"isp"`
		Domain               string        `json:"domain"`
		Hostnames            []interface{} `json:"hostnames"`
		IsTor                bool          `json:"isTor"`
		TotalReports         int           `json:"totalReports"`
		NumDistinctUsers     int           `json:"numDistinctUsers"`
		LastReportedAt       time.Time     `json:"lastReportedAt"`
		Reports              []struct {
			ReportedAt          time.Time `json:"reportedAt"`
			Comment             string    `json:"comment"`
			Categories          []int     `json:"categories"`
			ReporterId          int       `json:"reporterId"`
			ReporterCountryCode string    `json:"reporterCountryCode"`
			ReporterCountryName string    `json:"reporterCountryName"`
		} `json:"reports"`
	} `json:"data"`
}

type Enricher interface {
	Enrich(ip net.IP, out *LookupResult) error
}
