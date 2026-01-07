
# IPQuery

A fast and efficient IP address query API built in Go. **IPQuery** provides geolocation, ISP, and risk assessment data for IP addresses with a clean API and containerized deployment options.

## Features

- **IP Geolocation**: Retrieve geographical information (country, city, coordinates, timezone) for any IP address
- **ISP Information**: Get autonomous system number (ASN), organization, and ISP details
- **Risk Assessment**: Identify VPNs, proxies, Tor exit nodes, datacenters, and calculate risk scores (via [AbuseIP**DB**](https://www.abuseipdb.com/))
- **REST API**: Simple HTTP/REST endpoints for IP queries
- **Docker Support**: Ready-to-deploy containerized application
- **GeoLite Integration**: Uses [MaxMind GeoLite2](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data/) free databases for accurate geolocation data

![Screenshot from 2026-01-07 12-06-39.png](assets/Screenshot%20from%202026-01-07%2012-06-39.png)

## Prerequisites

> [!IMPORTANT] This service relies on seeing the real client IP at the edge.
> It must be deployed on infrastructure with a public IP address (VPS or equivalent) and a valid FQDN.
> If you deploy in private-only network and expose the service via tunnels (e.g. Cloudflare or Pangolin) 
> choose the _caddy_ option and expose via Caddy or equivalent. Deployments behind private-only networks only are not supported.

## Installation

### Binaries

Clone the repository:

```bash
git clone https://github.com/akyriako/ipquery.git
cd ipquery
```

Build the project:

```bash
make build
```

Run the server:

```bash
./bin/ipquery
```

The API will be available at `http://localhost:8080`

### Docker

Build and run with Docker:

```bash
docker build -t ipquery:latest .
docker run -p 8080:8080 ipquery:latest
```

> [!NOTE]
> That is a simple quick way to run the server, without the risk assessment enricher. For that you need to get 
> an API KEY from [AbuseIP**DB**](https://www.abuseipdb.com/) and set it as environment variable `ABUSEIPDB_API_KEY`.

### Docker Compose (with Caddy)

Use the included deployment files:

```bash
cd deploy/caddy
docker compose up -d
```

```yaml
version: "3.8"
name: ipquery

services:
  api:
    image: akyriako78/ipquery:0.1.17
    environment:
      LISTEN_ADDR: ":8080"
      TRUSTED_PROXY_CIDRS: "172.30.0.0/16"
      GEOLITE2_ASN: "/geolite/GeoLite2-ASN.mmdb"
      GEOLITE2_CITY: "/geolite/GeoLite2-City.mmdb"
      ABUSEIPDB_API_KEY: "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
    networks:
      ipqnet:
        ipv4_address: 172.30.0.10

  caddy:
    image: caddy:2-alpine
    depends_on:
      - api
    ports:
      - "8080:80"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
    networks:
      ipqnet:
        ipv4_address: 172.30.0.2

networks:
  ipqnet:
    driver: bridge
    ipam:
      config:
        - subnet: 172.30.0.0/16

```

> [!NOTE]
> Replace `ABUSEIPDB_API_KEY` with your very own key. **If none is provided, no risk assessment will be performed**.

### Docker Compose (with Kong)

Use the included deployment files:

```bash
cd deploy/kong
docker compose up -d
```

```yaml
version: "3.8"
name: ipquery_gw

services:
  api:
    image: akyriako78/ipquery:0.1.18
    environment:
      LISTEN_ADDR: ":8080"
      GEOLITE2_ASN: "/geolite/GeoLite2-ASN.mmdb"
      GEOLITE2_CITY: "/geolite/GeoLite2-City.mmdb"
      ABUSEIPDB_API_KEY: "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
    networks:
      - kong-net

  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: kong
      POSTGRES_USER: kong
      POSTGRES_PASSWORD: kongpass
    volumes:
      - kong_pg_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U kong -d kong"]
      interval: 5s
      timeout: 5s
      retries: 30
    networks:
      - kong-net

  kong-migrations:
    image: kong:3.7
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      KONG_DATABASE: postgres
      KONG_PG_HOST: postgres
      KONG_PG_DATABASE: kong
      KONG_PG_USER: kong
      KONG_PG_PASSWORD: kongpass
    command: kong migrations bootstrap
    restart: "no"
    networks:
      - kong-net

  kong:
    image: kong:3.7
    depends_on:
      kong-migrations:
        condition: service_completed_successfully
    environment:
      KONG_DATABASE: postgres
      KONG_PG_HOST: postgres
      KONG_PG_DATABASE: kong
      KONG_PG_USER: kong
      KONG_PG_PASSWORD: kongpass

      # Proxy (traffic)
      KONG_PROXY_LISTEN: "0.0.0.0:8000, 0.0.0.0:8443 ssl"
      # Admin API (keep internal if you want, see notes below)
      KONG_ADMIN_LISTEN: "0.0.0.0:8001, 0.0.0.0:8444 ssl"
      # Kong Manager UI
      KONG_ADMIN_GUI_LISTEN: "0.0.0.0:8002"
      KONG_ADMIN_GUI_URL: "http://localhost:8002"

      # OPTIONAL: convenient in dev; restrict/disable in prod
      KONG_ADMIN_GUI_AUTH: "basic-auth"
      KONG_PASSWORD: "adminpass"

      # recommended logs to stdout for docker
      KONG_PROXY_ACCESS_LOG: /dev/stdout
      KONG_ADMIN_ACCESS_LOG: /dev/stdout
      KONG_PROXY_ERROR_LOG: /dev/stderr
      KONG_ADMIN_ERROR_LOG: /dev/stderr

    ports:
      - "8000:8000"   # proxy (http)
      - "8443:8443"   # proxy (https)
      - "8001:8001"   # admin api (http)
      - "8444:8444"   # admin api (https)
      - "8002:8002"   # kong manager (gui)
    networks:
      - kong-net

  kong-config-import:
    image: kong:3.7
    depends_on:
      kong:
        condition: service_started
    networks:
      - kong-net
    volumes:
      - ./kong.yml:/kong/kong.yml:ro
    environment:
      KONG_DATABASE: postgres
      KONG_PG_HOST: postgres
      KONG_PG_DATABASE: kong
      KONG_PG_USER: kong
      KONG_PG_PASSWORD: kongpass
    command: kong config db_import /kong/kong.yml
    restart: "no"

volumes:
  kong_pg_data:

networks:
  kong-net:
    driver: bridge
```

![Screenshot from 2026-01-07 12-10-05.png](assets/Screenshot%20from%202026-01-07%2012-10-05.png)

> [!IMPORTANT]
> Make sure you provide a strong passwords for `KONG_PG_PASSWORD`, `KONG_PASSWORD` and `POSTGRES_PASSWORD`.

> [!NOTE]
> This is a more elaborate setup, but give you the flexibility to take advantage of all the features of Kong Gateway 
> like securing your exposed API endpoints with API Keys and/or impose rate limiting. This docker compose is not
> production-ready, but nevertheless a good starting point to get you going.

## API Endpoints

### `/own`

Resolves your own IP address, returns only the IP as `string`

### `/own/all`

Resolves your own IP address, returns all metadata found in MaxMind GeoLite2 databases as `JSON`, e.g.:

```json
{
  "ip": "141.98.XXX.XXX",
  "isp": {
    "asn": "AS39351",
    "org": "31173 Services AB",
    "isp": "31173 Services AB"
  },
  "location": {
    "country": "Denmark",
    "country_code": "DK",
    "city": "Copenhagen",
    "state": "Capital Region",
    "zipcode": "XXXX",
    "latitude": "XX.XXXX",
    "longitude": "XX.XXXX",
    "timezone": "Europe/Copenhagen",
    "localtime": "2026-01-07T12:06:30+01:00"
  },
  "risk": {
    "abuse_confidence_score": 0,
    "usage_type": "Fixed Line ISP",
    "is_tor": false,
    "total_reports": 1,
    "number_of_users_reported": 1,
    "last_reported_at": "2025-11-16T04:30:39Z"
  }
}
```

> [!NOTE]
> The MaxMind GeoLite2 databases are prebaked in the container image. If you need to provide your own load
> them in volumes and configure `GEOLITE2_ASN` and `GEOLITE2_CITY` environment variables accordingly.

### `/lookup/{ip}`

Resolves the requested IP address, returns all metadata found in MaxMind GeoLite2 databases as `JSON`.

> [!IMPORTANT] 
> Risk (stanza `risk`) is assessed only if a valid `ABUSEIPDB_API_KEY` is provided. 
> AbuseIP**DB** is allowing 1000 requests/day on the free tier, which is more than enough for hobby and non-commercial use.

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

For issues, questions, or suggestions, please open an issue on the GitHub repository.
