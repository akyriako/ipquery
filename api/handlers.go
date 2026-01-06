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

  <script src="https://cdn.tailwindcss.com"></script>

  <style>
    :root{
      --background: 0 0% 100%;
      --foreground: 222.2 84% 4.9%;
      --card: 0 0% 100%;
      --card-foreground: 222.2 84% 4.9%;
      --primary: 222.2 47.4% 11.2%;
      --primary-foreground: 210 40% 98%;
      --secondary: 210 40% 96.1%;
      --secondary-foreground: 222.2 47.4% 11.2%;
      --muted: 210 40% 96.1%;
      --muted-foreground: 215.4 16.3% 46.9%;
      --destructive: 0 84.2% 60.2%;
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
      --primary: 210 40% 98%;
      --primary-foreground: 222.2 47.4% 11.2%;
      --secondary: 217.2 32.6% 17.5%;
      --secondary-foreground: 210 40% 98%;
      --muted: 217.2 32.6% 17.5%;
      --muted-foreground: 215 20.2% 65.1%;
      --destructive: 0 62.8% 30.6%;
      --border: 217.2 32.6% 17.5%;
      --input: 217.2 32.6% 17.5%;
      --ring: 212.7 26.8% 83.9%;
    }
  </style>

  <link rel="stylesheet" href="https://unpkg.com/maplibre-gl@4.7.1/dist/maplibre-gl.css" />
  <script src="https://unpkg.com/maplibre-gl@4.7.1/dist/maplibre-gl.js"></script>
</head>

<body class="h-screen bg-[hsl(var(--background))] text-[hsl(var(--foreground))] overflow-hidden">
  <div class="h-full w-full p-4 flex flex-col gap-4">

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

        <p id="searchError" class="mt-2 hidden text-xs text-[hsl(var(--destructive))]"></p>

        <!-- actions under search -->
        <div class="mt-4 flex flex-wrap gap-2">
          <button
            id="ownBtn"
            class="h-9 px-3 rounded-[calc(var(--radius)-4px)] border border-[hsl(var(--border))] bg-[hsl(var(--secondary))] text-[hsl(var(--secondary-foreground))] text-sm font-medium hover:opacity-90"
            type="button"
          >Get your own IP</button>

          <button
            id="exGoogle"
            class="h-9 px-3 rounded-[calc(var(--radius)-4px)] border border-[hsl(var(--border))] bg-[hsl(var(--background))] text-sm font-medium hover:bg-[hsl(var(--muted))]"
            type="button"
          >8.8.8.8</button>

          <button
            id="exCloudflare"
            class="h-9 px-3 rounded-[calc(var(--radius)-4px)] border border-[hsl(var(--border))] bg-[hsl(var(--background))] text-sm font-medium hover:bg-[hsl(var(--muted))]"
            type="button"
          >1.1.1.1</button>
        </div>

        <p class="mt-3 text-xs text-[hsl(var(--muted-foreground))]">
        </p>
      </div>

      <div class="flex-1 grid grid-rows-2 gap-4 min-h-0">

        <div class="min-h-0 rounded-[var(--radius)] border border-[hsl(var(--border))] bg-[hsl(var(--card))] text-[hsl(var(--card-foreground))] shadow-sm overflow-hidden flex flex-col">

          <div class="px-4 py-3 border-b border-[hsl(var(--border))] flex items-center justify-between gap-3">
            <!-- IP header (replaces "Result JSON") -->
            <div class="min-w-0 flex items-center gap-2">
              <div id="resolvedIP" class="text-xl font-bold tracking-tight truncate">—</div>
              <button
                id="copyIPBtn"
                type="button"
                class="shrink-0 inline-flex items-center justify-center h-9 w-9 rounded-[calc(var(--radius)-4px)] border border-[hsl(var(--border))] bg-[hsl(var(--background))] hover:bg-[hsl(var(--muted))]"
                title="Copy IP"
                aria-label="Copy IP"
              >
                <!-- clipboard icon -->
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2v-2"/>
                  <rect x="4" y="2" width="12" height="16" rx="2" ry="2"/>
                  <path d="M9 2h2"/>
                </svg>
              </button>
              <span id="copyToast" class="hidden text-xs text-[hsl(var(--muted-foreground))]">Copied</span>
            </div>

            <!-- JSON/Table toggle (replaces old Get Own IP button location) -->
            <div class="flex items-center gap-2">
              <span class="text-xs text-[hsl(var(--muted-foreground))]">JSON</span>

              <!-- simple shadcn-ish switch -->
              <button
                id="viewToggle"
                type="button"
                role="switch"
                aria-checked="false"
                class="relative inline-flex h-6 w-11 items-center rounded-full border border-[hsl(var(--border))] bg-[hsl(var(--muted))] transition"
                title="Toggle JSON/Table"
              >
                <span
                  id="viewToggleThumb"
                  class="inline-block h-5 w-5 translate-x-0.5 rounded-full bg-[hsl(var(--background))] shadow transition"
                ></span>
              </button>

              <span class="text-xs text-[hsl(var(--muted-foreground))]">Table</span>
            </div>
          </div>

          <!-- content -->
          <div class="flex-1 min-h-0 overflow-auto bg-[hsl(var(--background))]">
            <pre id="jsonOut" class="p-4 text-sm"></pre>

            <div id="tableWrap" class="hidden p-4">
              <table class="w-full text-sm border border-[hsl(var(--border))] rounded-[calc(var(--radius)-4px)] overflow-hidden">
                <tbody id="tableBody"></tbody>
              </table>
              <p class="mt-2 text-xs text-[hsl(var(--muted-foreground))]">
              </p>
            </div>
          </div>

        </div>

        <div class="min-h-0 rounded-[var(--radius)] border border-[hsl(var(--border))] bg-[hsl(var(--card))] text-[hsl(var(--card-foreground))] shadow-sm overflow-hidden flex flex-col">
          <div class="px-4 py-3 border-b border-[hsl(var(--border))]">
            <h2 class="text-sm font-semibold">Location</h2>
            <p id="locationSummary" class="text-xs text-[hsl(var(--muted-foreground))]">
              Loading location…
            </p>
          </div>
          <div id="map" class="flex-1"></div>
        </div>

      </div>

  </div>

<script>
  // DOM refs
  const jsonOut = document.getElementById("jsonOut");
  const tableWrap = document.getElementById("tableWrap");
  const tableBody = document.getElementById("tableBody");

  const searchBtn = document.getElementById("searchBtn");
  const searchBox = document.getElementById("searchBox");
  const searchError = document.getElementById("searchError");

  const ownBtn = document.getElementById("ownBtn");
  const exGoogle = document.getElementById("exGoogle");
  const exCloudflare = document.getElementById("exCloudflare");

  const resolvedIP = document.getElementById("resolvedIP");
  const copyIPBtn = document.getElementById("copyIPBtn");
  const copyToast = document.getElementById("copyToast");

  const viewToggle = document.getElementById("viewToggle");
  const viewToggleThumb = document.getElementById("viewToggleThumb");

  const locationSummary = document.getElementById("locationSummary");

  let map = null;
  let marker = null;

  // view state: "json" or "table"
  let viewMode = "json";
  // last response data (used when toggling views)
  let lastData = null;

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

  // strict IPv4
  function isValidIPv4(ip) {
    const parts = ip.split(".");
    if (parts.length !== 4) return false;

    for (let i = 0; i < 4; i++) {
      const p = parts[i];
      if (!/^\d+$/.test(p)) return false;
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

  function setResolvedIP(ip) {
    resolvedIP.textContent = ip && ip.length ? ip : "—";
  }

  async function copyToClipboard(text) {
    try {
      await navigator.clipboard.writeText(text);
      copyToast.classList.remove("hidden");
      setTimeout(() => copyToast.classList.add("hidden"), 900);
    } catch {
      // fallback
      const ta = document.createElement("textarea");
      ta.value = text;
      document.body.appendChild(ta);
      ta.select();
      document.execCommand("copy");
      document.body.removeChild(ta);
      copyToast.classList.remove("hidden");
      setTimeout(() => copyToast.classList.add("hidden"), 900);
    }
  }

  // flatten object into dot-keys for table view
  function flatten(obj, prefix, out) {
    if (obj === null || obj === undefined) {
      out.push([prefix, "null"]);
      return;
    }

    const t = typeof obj;
    if (t === "string" || t === "number" || t === "boolean") {
      out.push([prefix, String(obj)]);
      return;
    }

    if (Array.isArray(obj)) {
      if (obj.length === 0) {
        out.push([prefix, "[]"]);
        return;
      }
      for (let i = 0; i < obj.length; i++) {
        flatten(obj[i], prefix ? (prefix + "[" + i + "]") : ("[" + i + "]"), out);
      }
      return;
    }

    // object
    const keys = Object.keys(obj);
    if (keys.length === 0) {
      out.push([prefix, "{}"]);
      return;
    }
    for (let i = 0; i < keys.length; i++) {
      const k = keys[i];
      const nextPrefix = prefix ? (prefix + "." + k) : k;
      flatten(obj[k], nextPrefix, out);
    }
  }

  function renderTable(data) {
    tableBody.innerHTML = "";
    if (!data) return;

    const rows = [];
    flatten(data, "", rows);

    // keep table stable order

    for (let i = 0; i < rows.length; i++) {
      const key = rows[i][0];
      const val = rows[i][1];

      const tr = document.createElement("tr");

      const tdK = document.createElement("td");
      tdK.className = "align-top w-1/3 px-3 py-2 text-xs font-medium border-b border-[hsl(var(--border))] text-[hsl(var(--muted-foreground))]";
      tdK.textContent = key
	  .split(".")
	  .pop()
	  .replace(/_/g, " ")
	  .toUpperCase();

      const tdV = document.createElement("td");
      tdV.className = "align-top px-3 py-2 text-xs border-b border-[hsl(var(--border))] font-mono break-all";
      tdV.textContent = val;

      tr.appendChild(tdK);
      tr.appendChild(tdV);
      tableBody.appendChild(tr);
    }
  }

  function applyViewMode() {
    if (viewMode === "json") {
      jsonOut.classList.remove("hidden");
      tableWrap.classList.add("hidden");
      viewToggle.setAttribute("aria-checked", "false");
      viewToggleThumb.classList.remove("translate-x-5");
      viewToggleThumb.classList.add("translate-x-0.5");
    } else {
      jsonOut.classList.add("hidden");
      tableWrap.classList.remove("hidden");
      viewToggle.setAttribute("aria-checked", "true");
      viewToggleThumb.classList.remove("translate-x-0.5");
      viewToggleThumb.classList.add("translate-x-5");
      renderTable(lastData);
    }
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
      center: center,
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
    // basic loading states
    clearSearchError();
    setResolvedIP("—");
    jsonOut.textContent = "Loading " + url + "...";
    lastData = null;
    if (viewMode === "table") {
      tableBody.innerHTML = "";
    }

    try {
      const response = await fetch(url, { headers: { "Accept": "application/json" } });

      if (!response.ok) {
        jsonOut.textContent = "HTTP error: " + response.status + " (" + url + ")";
        return;
      }

      const data = await response.json();
      lastData = data;

      // update header IP from response
      setResolvedIP(data && data.ip ? data.ip : "—");

      // JSON output
      jsonOut.textContent = pretty(data);

      // table output if needed
      if (viewMode === "table") {
        renderTable(data);
      }

      // location summary + map
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

  function lookupIP(ip) {
    const v = (ip || "").trim();

    if (!v) {
      showSearchError("IP address is required.");
      return;
    }
    if (!isValidIPv4(v)) {
      showSearchError("Invalid IPv4 address. Example: 8.8.8.8");
      return;
    }

    clearSearchError();
    fetchAndRender("/lookup/" + encodeURIComponent(v));
  }

  function lookupFromSearch() {
    lookupIP(searchBox.value);
  }

  // events
  searchBtn.addEventListener("click", lookupFromSearch);

  searchBox.addEventListener("input", clearSearchError);
  searchBox.addEventListener("keydown", function (e) {
    if (e.key === "Enter") {
      e.preventDefault();
      lookupFromSearch();
    }
  });

  ownBtn.addEventListener("click", function () {
    // keep whatever is in the search box; just load your own
    loadOwnAll();
  });

  exGoogle.addEventListener("click", function () {
    searchBox.value = "8.8.8.8";
    lookupIP("8.8.8.8");
  });

  exCloudflare.addEventListener("click", function () {
    searchBox.value = "1.1.1.1";
    lookupIP("1.1.1.1");
  });

  copyIPBtn.addEventListener("click", function () {
    const ip = (resolvedIP.textContent || "").trim();
    if (ip && ip !== "—") {
      copyToClipboard(ip);
    }
  });

  viewToggle.addEventListener("click", function () {
    viewMode = (viewMode === "json") ? "table" : "json";
    applyViewMode();
  });

  // init
  applyViewMode();
  loadOwnAll();
</script>
</body>
</html>
`
