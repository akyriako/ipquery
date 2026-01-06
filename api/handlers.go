package api

import (
	"encoding/json"
	"html/template"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	*LookupClient
}

func (s *Server) GetHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) GetOwnIP(w http.ResponseWriter, r *http.Request) {
	ipStr := s.GetClientIP(r)
	if ipStr == "" {
		http.Error(w, "unable to determine client ip", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(ipStr))
}

func (s *Server) GetOwnIPAll(w http.ResponseWriter, r *http.Request) {
	ipStr := s.GetClientIP(r)
	if ipStr == "" {
		http.Error(w, "unable to determine client ip", http.StatusBadRequest)
		return
	}

	s.lookupIPAll(w, r, ipStr)
}

func (s *Server) LookupIPAll(w http.ResponseWriter, r *http.Request) {
	ipStr := chi.URLParam(r, "ip")
	if ipStr == "" {
		http.Error(w, "missing ip parameter", http.StatusBadRequest)
		return
	}

	s.lookupIPAll(w, r, ipStr)
}

func (s *Server) lookupIPAll(w http.ResponseWriter, r *http.Request, ipStr string) {
	ipNet := net.ParseIP(ipStr)
	if ipNet == nil {
		http.Error(w, "invalid ip", http.StatusBadRequest)
		return
	}

	res := LookupResult{IP: ipNet.String()}

	if err := s.AsnReader.Enrich(ipNet, &res); err != nil {
		http.Error(w, "asn lookup failed", http.StatusInternalServerError)
		return
	}
	if err := s.CityReader.Enrich(ipNet, &res); err != nil {
		http.Error(w, "city lookup failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *Server) Index() http.HandlerFunc {
	tpl := template.Must(template.New("landing").Parse(landingHTML))

	return func(w http.ResponseWriter, r *http.Request) {
		// Only serve exactly "/"
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = tpl.Execute(w, nil)
	}
}

const landingHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>ipquery</title>

  <!-- Tailwind Play CDN (keeps frontend simple, no build step) -->
  <script src="https://cdn.tailwindcss.com"></script>

  <!-- Shadcn-like tokens (CSS variables) -->
  <style>
    :root{
      --background: 0 0% 100%;
      --foreground: 222.2 84% 4.9%;
      --card: 0 0% 100%;
      --card-foreground: 222.2 84% 4.9%;
      --popover: 0 0% 100%;
      --popover-foreground: 222.2 84% 4.9%;
      --primary: 222.2 47.4% 11.2%;
      --primary-foreground: 210 40% 98%;
      --secondary: 210 40% 96.1%;
      --secondary-foreground: 222.2 47.4% 11.2%;
      --muted: 210 40% 96.1%;
      --muted-foreground: 215.4 16.3% 46.9%;
      --accent: 210 40% 96.1%;
      --accent-foreground: 222.2 47.4% 11.2%;
      --destructive: 0 84.2% 60.2%;
      --destructive-foreground: 210 40% 98%;
      --border: 214.3 31.8% 91.4%;
      --input: 214.3 31.8% 91.4%;
      --ring: 222.2 84% 4.9%;
      --radius: 0.75rem;
    }
    .dark{
      --background: 222.2 84% 4.9%;
      --foreground: 210 40% 98%;
      --card: 222.2 84% 4.9%;
      --card-foreground: 210 40% 98%;
      --popover: 222.2 84% 4.9%;
      --popover-foreground: 210 40% 98%;
      --primary: 210 40% 98%;
      --primary-foreground: 222.2 47.4% 11.2%;
      --secondary: 217.2 32.6% 17.5%;
      --secondary-foreground: 210 40% 98%;
      --muted: 217.2 32.6% 17.5%;
      --muted-foreground: 215 20.2% 65.1%;
      --accent: 217.2 32.6% 17.5%;
      --accent-foreground: 210 40% 98%;
      --destructive: 0 62.8% 30.6%;
      --destructive-foreground: 210 40% 98%;
      --border: 217.2 32.6% 17.5%;
      --input: 217.2 32.6% 17.5%;
      --ring: 212.7 26.8% 83.9%;
    }
  </style>

  <!-- MapLibre GL (simple, no framework) -->
  <link rel="stylesheet" href="https://unpkg.com/maplibre-gl@4.7.1/dist/maplibre-gl.css" />
  <script src="https://unpkg.com/maplibre-gl@4.7.1/dist/maplibre-gl.js"></script>
</head>

<body class="h-screen bg-[hsl(var(--background))] text-[hsl(var(--foreground))]">
  <div class="h-full w-full flex">

    <!-- LEFT: 50% -->
    <section class="w-1/2 h-full border-r border-[hsl(var(--border))] p-6 flex flex-col gap-6">
      <div class="rounded-[var(--radius)] border border-[hsl(var(--border))] bg-[hsl(var(--card))] text-[hsl(var(--card-foreground))] shadow-sm p-6">
        <h1 class="text-2xl font-semibold tracking-tight">ipquery</h1>
        <p class="mt-2 text-sm text-[hsl(var(--muted-foreground))]">
          Give an IP, or let ipquery discover you current public IP, and you’ll see basic, useful information like the city, country, network provider, and an approximate location on the map. It’s meant for quick checks, understanding traffic, debugging, investigating logs, or just satisfying curiosity.
          IP locations aren’t exact, and this doesn’t try to pretend they are. The goal is to give you enough context to make sense of an address without digging through raw data or third-party dashboards.
        </p>
      </div>

      <div class="rounded-[var(--radius)] border border-[hsl(var(--border))] bg-[hsl(var(--card))] text-[hsl(var(--card-foreground))] shadow-sm p-6">
        <label class="text-sm font-medium" for="searchBox">Search</label>

        <div class="mt-2 flex gap-2">
          <input
            id="searchBox"
            type="text"
            placeholder="Enter IPv4 address (e.g. 8.8.8.8)"
            class="w-full h-10 rounded-[calc(var(--radius)-4px)] border border-[hsl(var(--input))] bg-[hsl(var(--background))] px-3 text-sm outline-none focus:ring-2 focus:ring-[hsl(var(--ring))]"
          />
          <button
            id="searchBtn"
            class="h-10 px-4 rounded-[calc(var(--radius)-4px)] bg-[hsl(var(--primary))] text-[hsl(var(--primary-foreground))] text-sm font-medium hover:opacity-90"
            type="button"
          >Go</button>
        </div>

        <p id="searchError"
           class="mt-2 hidden text-xs text-[hsl(var(--destructive))]">
        </p>


      </div>
    </section>

    <!-- RIGHT: 50% split horizontally -->
    <section class="w-1/2 h-full flex flex-col">
      <!-- TOP (JSON) -->
      <div class="h-1/2 border-b border-[hsl(var(--border))] p-4">
        <div class="h-full rounded-[var(--radius)] border border-[hsl(var(--border))] bg-[hsl(var(--card))] text-[hsl(var(--card-foreground))] shadow-sm overflow-hidden flex flex-col">
          <div class="px-4 py-3 border-b border-[hsl(var(--border))] flex items-center justify-between">
            <div>
              <h2 class="text-sm font-semibold">Result JSON</h2>
            </div>
            <button
              id="refreshBtn"
              class="h-9 px-3 rounded-[calc(var(--radius)-4px)] border border-[hsl(var(--border))] bg-[hsl(var(--secondary))] text-[hsl(var(--secondary-foreground))] text-sm font-medium hover:opacity-90"
              type="button"
            >Get your own IP</button>
          </div>
          <pre id="jsonOut" class="flex-1 p-4 text-xs overflow-auto bg-[hsl(var(--background))]"></pre>
        </div>
      </div>

      <!-- BOTTOM (MAP) -->
      <div class="h-1/2 p-4">
        <div class="h-full rounded-[var(--radius)] border border-[hsl(var(--border))] bg-[hsl(var(--card))] text-[hsl(var(--card-foreground))] shadow-sm overflow-hidden flex flex-col">
          <div class="px-4 py-3 border-b border-[hsl(var(--border))]">
            <h2 class="text-sm font-semibold">Location</h2>
            <p id="locationSummary" class="text-xs text-[hsl(var(--muted-foreground))]">
              Loading location…
            </p>
          </div>
          <div id="map" class="flex-1"></div>
        </div>
      </div>
    </section>
  </div>

<script>
  // DOM references
  const jsonOut = document.getElementById("jsonOut");
  const refreshBtn = document.getElementById("refreshBtn");
  const searchBtn = document.getElementById("searchBtn");
  const searchBox = document.getElementById("searchBox");
  const searchError = document.getElementById("searchError");
  const locationSummary = document.getElementById("locationSummary");

  let map = null;
  let marker = null;

  function pretty(obj) {
    return JSON.stringify(obj, null, 2);
  }

  function clearSearchError() {
    searchError.textContent = "";
    searchError.classList.add("hidden");
    searchBox.classList.remove(
      "border-[hsl(var(--destructive))]",
      "focus:ring-[hsl(var(--destructive))]"
    );
  }

  function showSearchError(msg) {
    searchError.textContent = msg;
    searchError.classList.remove("hidden");
    searchBox.classList.add(
      "border-[hsl(var(--destructive))]",
      "focus:ring-[hsl(var(--destructive))]"
    );
  }

  // IPv4 validation (0-255 each octet)
  function isValidIPv4(ip) {
    const parts = ip.split(".");
    if (parts.length !== 4) return false;

    for (let i = 0; i < 4; i++) {
      const p = parts[i];
      if (!/^\d+$/.test(p)) return false;
      // strict: no leading zeros like "01"
      if (p.length > 1 && p[0] === "0") return false;

      const n = Number(p);
      if (!Number.isFinite(n) || n < 0 || n > 255) return false;
    }
    return true;
  }

  function extractLatLon(data) {
    if (!data || typeof data !== "object") return null;
    if (!data.location || typeof data.location !== "object") return null;

    const lat = data.location.latitude;
    const lon = data.location.longitude;

    if (typeof lat !== "number" || typeof lon !== "number") return null;
    if (!Number.isFinite(lat) || !Number.isFinite(lon)) return null;

    return { lat: lat, lon: lon };
  }

  function updateLocationSummary(data) {
    if (!locationSummary || !data) return;

    const ip = data.ip || "This IP";
    const loc = data.location || {};

    const city = loc.city || "";
    const state = loc.state || "";
    const zip = loc.zipcode || "";
    const country = loc.country || "";

    const parts = [];
    if (city) parts.push(city);
    if (state) parts.push(state);
    if (zip) parts.push(zip);

    const place = parts.length > 0 ? parts.join(", ") : "an unknown location";
    locationSummary.textContent =
      ip + " is located approximately in " + place + (country ? ", " + country : "") + ".";
  }

  function ensureMap(center) {
    if (map) return;

    map = new maplibregl.Map({
      container: "map",
      style: {
        version: 8,
        sources: {
          osm: {
            type: "raster",
            tiles: [
              "https://a.tile.openstreetmap.org/{z}/{x}/{y}.png",
              "https://b.tile.openstreetmap.org/{z}/{x}/{y}.png",
              "https://c.tile.openstreetmap.org/{z}/{x}/{y}.png"
            ],
            tileSize: 256,
            attribution: "© OpenStreetMap contributors"
          }
        },
        layers: [{ id: "osm", type: "raster", source: "osm" }]
      },
      center: center, // [lon, lat]
      zoom: 10
    });

    map.addControl(new maplibregl.NavigationControl(), "top-right");
  }

  function setMarker(lon, lat) {
    ensureMap([lon, lat]);

    if (!marker) {
      marker = new maplibregl.Marker().setLngLat([lon, lat]).addTo(map);
    } else {
      marker.setLngLat([lon, lat]);
    }

    map.flyTo({ center: [lon, lat], zoom: 11 });
  }

  async function fetchAndRender(url) {
    jsonOut.textContent = "Loading " + url + "...";
    clearSearchError();

    try {
      const response = await fetch(url, {
        headers: { "Accept": "application/json" }
      });

      if (!response.ok) {
        jsonOut.textContent = "HTTP error: " + response.status + " (" + url + ")";
        return;
      }

      const data = await response.json();
      jsonOut.textContent = pretty(data);

      updateLocationSummary(data);

      const coords = extractLatLon(data);
      if (coords) {
        setMarker(coords.lon, coords.lat);
      }
    } catch (err) {
      jsonOut.textContent = "Fetch error: " + (err && err.message ? err.message : String(err));
    }
  }

  function loadOwnAll() {
    fetchAndRender("/own/all");
  }

  function lookupFromSearch() {
    const ip = (searchBox.value || "").trim();

    if (!ip) {
      showSearchError("IP address is required.");
      return;
    }

    if (!isValidIPv4(ip)) {
      showSearchError("Invalid IPv4 address. Example: 8.8.8.8");
      return;
    }

    clearSearchError();
    fetchAndRender("/lookup/" + encodeURIComponent(ip));
  }

  // Events
  refreshBtn.addEventListener("click", loadOwnAll);
  searchBtn.addEventListener("click", lookupFromSearch);

  // clear inline error while typing
  searchBox.addEventListener("input", clearSearchError);

  // Enter key triggers lookup
  searchBox.addEventListener("keydown", function (e) {
    if (e.key === "Enter") {
      e.preventDefault();
      lookupFromSearch();
    }
  });

  // Initial load
  loadOwnAll();
</script>
</body>
</html>
`
