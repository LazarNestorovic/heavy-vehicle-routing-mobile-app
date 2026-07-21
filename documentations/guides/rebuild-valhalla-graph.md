# Kako rebuild-ovati Valhalla routing graf

Koristi se kad se promeni filter tagova (`update-osm.sh`) ili kad treba osvežiti OSM podatke.

## Automatski (preporučeno za redovno ažuriranje)

```bash
./update-osm.sh
```

Radi sve korake: preuzima svež ekstrakt sa Geofabrik-a, proverava MD5, filtrira, kopira u `valhalla/custom_files/`, gradi graf, restartuje servis. Loguje u `osm-data/update.log`.

## Ručno (kad ne treba re-download, npr. samo menjaš filter tagova)

```bash
# 1. Filtriraj postojeći raw fajl
osmium tags-filter \
  osm-data/raw/serbia-latest.osm.pbf \
  w/highway w/maxheight w/maxweight w/maxwidth \
  w/hgv w/hazmat w/bridge w/tunnel w/surface w/maxspeed \
  n/amenity=fuel,parking n/highway=rest_area n/barrier \
  r/type=restriction \
  --output osm-data/filtered/serbia-hvt.osm.pbf --overwrite

# 2. Kopiraj u Valhalla build ulaz (ova dva foldera NISU ista lokacija!)
cp osm-data/filtered/serbia-hvt.osm.pbf valhalla/custom_files/serbia-hvt.osm.pbf

# 3. (Opciono, ali preporučeno za čist rebuild) očisti stare tiles preko kontejnera
#    — tiles/ folder je root-owned jer ga pravi Docker, obično ti fali dozvola za direktan rm.
docker compose --profile build run --rm --entrypoint sh valhalla-build -c "rm -rf /data/tiles/*"

# 4. Build
docker compose --profile build up valhalla-build --abort-on-container-exit

# 5. Pokreni/restartuj glavni servis da učita novi graf
docker compose up -d valhalla
# ili, ako je već radio:
docker compose restart valhalla
```

## Provera da graf radi

```bash
curl -s http://localhost:8002/status | jq .
curl -s http://localhost:8002/route -d '{
  "locations": [{"lat":44.8,"lon":20.4},{"lat":45.25,"lon":19.85}],
  "costing": "truck",
  "costing_options": {"truck": {"height": 4.0, "weight": 40}}
}' | jq .trip.summary
```

## Bitna zamka

`osm-data/filtered/` (izlaz filtera) i `valhalla/custom_files/` (ulaz build-a) su **dva odvojena foldera**. `update-osm.sh` sada (od [2026-07-21](../fixes/2026-07-21-osmium-filter-node-tags.md)) kopira jedno u drugo automatski — ako se ta linija ikad ukloni, build će tiho koristiti stare/zastarele podatke bez greške.
