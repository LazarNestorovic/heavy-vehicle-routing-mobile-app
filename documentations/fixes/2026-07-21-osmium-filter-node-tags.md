# Osmium filter je odbacivao node tagove (rest-area/fuel/parking/barrier)

**Datum:** 2026-07-21
**Fajlovi:** [`update-osm.sh:32-40`](../../update-osm.sh), `osm-data/filtered/serbia-hvt.osm.pbf`, `valhalla/custom_files/serbia-hvt.osm.pbf`

## Problem

1. **Node tagovi se nisu čuvali.** `osmium tags-filter` poziv u `update-osm.sh` je imao samo `w/` (way) i `r/` (relation) pravila:
   ```
   w/highway w/maxheight w/maxweight w/maxwidth w/hgv w/hazmat w/bridge w/tunnel w/surface w/maxspeed r/type=restriction
   ```
   U OSM-u su benzinske pumpe (`amenity=fuel`), parkinzi (`amenity=parking`), odmarališta (`highway=rest_area`) i tačkasti barijeri (`barrier=*`, npr. `height_restrictor`) skoro uvek **node** elementi, ne way/relation. Bez `n/` pravila, osmium ih je tiho brisao iz filtriranog fajla. Ovo direktno blokira modul odmarališta/pauza vozača planiran u [`SPECIFIKACIJA.md`](../../SPECIFIKACIJA.md) (sekcija 3.8).

2. **Filtrirani fajl se nikad nije prosleđivao Valhalla build-u.** `osmium tags-filter` piše u `osm-data/filtered/serbia-hvt.osm.pbf` (`$FILTERED_DIR`), ali `docker-compose.yml`-ov `valhalla-build` servis čita ulaz iz `/data/custom_files/serbia-hvt.osm.pbf`, što je mount od `./valhalla/custom_files/`. Skripta nije imala nijedan korak koji kopira jedno u drugo — build je efektivno uvek koristio bilo šta što je ranije ručno stavljeno u `valhalla/custom_files/`, ignorišući nove filtere.

## Popravka

`update-osm.sh`, blok filtriranja (linije ~32-40 pre popravke):

```diff
     osmium tags-filter \
       "$RAW_DIR/serbia-latest.osm.pbf" \
       w/highway w/maxheight w/maxweight w/maxwidth \
       w/hgv w/hazmat w/bridge w/tunnel \
       w/surface w/maxspeed \
+      n/amenity=fuel,parking n/highway=rest_area n/barrier \
       r/type=restriction \
       --output "$FILTERED_DIR/serbia-hvt.osm.pbf" \
       --overwrite
+
+    # Valhalla build čita ulaz iz valhalla/custom_files, ne iz osm-data/filtered,
+    # pa filtrirani fajl moramo kopirati tamo pre pokretanja build-a.
+    cp "$FILTERED_DIR/serbia-hvt.osm.pbf" "$PROJECT_DIR/valhalla/custom_files/serbia-hvt.osm.pbf"
```

Isti `n/` set primenjen je i ručno (bez re-downloada) nad postojećim `osm-data/raw/serbia-latest.osm.pbf`, pošto sirovi fajl već postoji i nema potrebe za novim preuzimanjem sa Geofabrik-a samo zbog izmene filtera.

## Verifikacija

- Veličina filtriranog fajla porasla sa `68 121 223 B` na `68 402 528 B` nakon dodavanja node pravila (potvrđuje da su novi elementi zaista uključeni).
- Stari `valhalla/tiles/` (root-owned, iz prethodnog build-a) obrisan preko `docker compose --profile build run --rm --entrypoint sh valhalla-build -c "rm -rf /data/tiles/*"` da rebuild ne meša podatke iz starog i novog grafa.
- Graf rebuild-ovan preko `docker compose --profile build up valhalla-build --abort-on-container-exit` nad novim `valhalla/custom_files/serbia-hvt.osm.pbf` — **uspešno** (`[3/3] Build zavrsen!`, exit code 0). Graf: 898 337 čvorova, 2 216 948 usmerenih ivica.
- `valhalla-server` pokrenut (`docker compose up -d valhalla`), `/status` odgovara (verzija 3.8.2).
- `/route` sa `costing: truck` testiran za rutu Beograd → Novi Sad (`height: 4.0`, `weight: 40`) — vraća validnu rutu sa `travel_type: truck` maneuvers-ima. Truck costing end-to-end potvrđen kao radan.
- Preostalo (nije blokirajuće za ovu popravku, ali beleži se za budući rad): backend još ne čita node-ove sa `amenity=fuel/parking` iz OSM-a direktno (Valhalla ih ne izlaže kroz `/route`) — to je posao modula odmarališta iz `SPECIFIKACIJA.md` 3.8, ne ovog fix-a.

Vidi i: [guide za rebuild grafa](../guides/rebuild-valhalla-graph.md).
