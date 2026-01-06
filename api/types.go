package api

type LookupResult struct {
	IP       string       `json:"ip"`
	ISP      ISPInfo      `json:"isp"`
	Location LocationInfo `json:"location"`
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
