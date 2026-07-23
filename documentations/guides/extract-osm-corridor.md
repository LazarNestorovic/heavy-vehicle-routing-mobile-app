# Kako regenerisati bounded OSM koridor za algorithm paket

`backend/internal/algorithm/testdata/beograd-novisad-corridor.osm` je fiksni test fixture (committed u repo, ne generiše se pri svakom build-u) za `backend/internal/algorithm` — sopstvenu A*/Dijkstra implementaciju iz `SPECIFIKACIJA.md` sekcije 3.3.2. Ako treba regenerisati (npr. posle osvežavanja `osm-data/raw/serbia-latest.osm.pbf` preko `update-osm.sh`, ili da se pokrije drugi koridor):

```bash
# 1. Izvuci geografski bbox oko Beograd-Novi Sad koridora iz vec filtriranih podataka
osmium extract --bbox 19.75,44.75,20.45,45.30 --strategy complete_ways \
  osm-data/filtered/serbia-hvt.osm.pbf \
  -o /tmp/beograd-novisad.osm.pbf --overwrite

# 2. Suzi na glavne klase puteva - bez ovog koraka fajl je ~84MB (sve gradske ulice
#    Beograda i Novog Sada u bbox-u), sa njim je ~11MB (samo koridor)
osmium tags-filter /tmp/beograd-novisad.osm.pbf \
  w/highway=motorway,trunk,primary,secondary,motorway_link,trunk_link,primary_link,secondary_link \
  --output /tmp/beograd-novisad-corridor.osm.pbf --overwrite

# 3. Konvertuj u OSM XML - Go kod parsira XML direktno (encoding/xml, bez
#    eksterne PBF parsing biblioteke)
osmium cat /tmp/beograd-novisad-corridor.osm.pbf \
  -o backend/internal/algorithm/testdata/beograd-novisad-corridor.osm -f osm --overwrite
```

## Zašto je fajl committed, a ne generisan/gitignored kao ostali OSM podaci

Za razliku od `osm-data/raw/` i `osm-data/filtered/` (veliki, redovno osvežavani nacionalni podaci, namerno gitignored), ovo je mali, stabilan test fixture specifičan za jedan paket. Go konvencija (`testdata/` folder) i potreba za reproduktivnim testovima bez zavisnosti od lokalno pokrenutog `update-osm.sh` pipeline-a opravdavaju da bude u repo-u (~11MB, prihvatljivo).

## Poznato ograničenje

Ova ekstrakcija ne sadrži node-level `barrier` tagove sa visinskim ograničenjima (samo way-level `maxheight`) — `LoadOSMXML` trenutno čita samo way tagove. Ako je stvarna prepreka na terenu markirana kao barijera na node-u a ne kao `maxheight` na way-u, ovaj bounded graf je neće videti. Vidi [bounded A*/Dijkstra feature](../features/2026-07-21-bounded-astar-dijkstra.md) za konkretan primer (Novi Banovci slučaj) gde je ovo primećeno.
