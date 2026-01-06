FROM golang:1.24-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build ONLY the main package under /cmd (not ./...)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /app ./cmd

FROM gcr.io/distroless/static:nonroot

# Keep defaults like "./geolite/GeoLite2-City.mmdb" working:
# distroless default WORKDIR is "/" so "./geolite/..." => "/geolite/..."
COPY --from=build /app /app

# Bake the mmdbs into the image at /geolite
COPY --chown=nonroot:nonroot geolite/GeoLite2-ASN.mmdb /geolite/GeoLite2-ASN.mmdb
COPY --chown=nonroot:nonroot geolite/GeoLite2-City.mmdb /geolite/GeoLite2-City.mmdb
# optional if you use it later:
# COPY --chown=nonroot:nonroot geolite/GeoLite2-Country.mmdb /geolite/GeoLite2-Country.mmdb

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app"]
