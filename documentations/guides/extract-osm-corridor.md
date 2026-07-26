# Kako regenerisati bounded OSM koridor za algorithm paket

`backend/internal/algorithm/testdata/beograd-novisad-corridor.osm` je fiksni test fixture (committed u repo, ne generiše se pri svakom build-u) za `backend/internal/algorithm` — sopstvenu A*/Dijkstra implementaciju iz `SPECIFIKACIJA.md` sekcije 3.3.2. Ako treba regenerisati (npr. posle osvežavanja `osm-data/raw/serbia-latest.osm.pbf` preko `update-osm.sh`, ili da se pokrije drugi koridor):

```bash
# 1. Izvuci geografski bbox oko Beograd-Novi Sad koridora iz vec filtriranih podataka
osmium extract --bbox 19.75,44.75,20.45,45.30 --strategy complete_ways \
  osm-data/filtered/serbia-hvt.osm.pbf \
  -o /tmp/beograd-novisad.osm.pbf --overwrite

# 2. Suzi na glavne klase puteva - bez ovog koraka fajl je ~84MB (sve gradske ulice
#    Beograda i Novog Sada u bbox-u), sa njim je ~11MB (samo koridor). `n/barrier`
#    je dodato pored `w/highway=...` da node-level barijere (height_restrictor,
#    lift_gate, toll_booth...) prežive ovaj drugi filter prolaz - bez njega bi
#    LoadOSMXML video samo way-level maxheight/maxweight, nikad node barijere.
osmium tags-filter /tmp/beograd-novisad.osm.pbf \
  w/highway=motorway,trunk,primary,secondary,motorway_link,trunk_link,primary_link,secondary_link \
  n/barrier \
  --output /tmp/beograd-novisad-corridor.osm.pbf --overwrite

# 3. Konvertuj u OSM XML - Go kod parsira XML direktno (encoding/xml, bez
#    eksterne PBF parsing biblioteke)
osmium cat /tmp/beograd-novisad-corridor.osm.pbf \
  -o backend/internal/algorithm/testdata/beograd-novisad-corridor.osm -f osm --overwrite
```

## Zašto je fajl committed, a ne generisan/gitignored kao ostali OSM podaci

Za razliku od `osm-data/raw/` i `osm-data/filtered/` (veliki, redovno osvežavani nacionalni podaci, namerno gitignored), ovo je mali, stabilan test fixture specifičan za jedan paket. Go konvencija (`testdata/` folder) i potreba za reproduktivnim testovima bez zavisnosti od lokalno pokrenutog `update-osm.sh` pipeline-a opravdavaju da bude u repo-u (~11MB, prihvatljivo).

## Node-level barijere (dodato 2026-07-26)

`LoadOSMXML` sada čita i node tagove — ako node ima `barrier` tag ZAJEDNO sa `maxheight`/`maxweight` (standardna OSM konvencija za `barrier=height_restrictor`/`lift_gate`), ta ograničenja se primenjuju na sve ivice kroz taj čvor (vidi [bounded A*/Dijkstra feature](../features/2026-07-21-bounded-astar-dijkstra.md)).

**Nalaz nad regenerisanim ekstraktom**: od 2808 node-ova sa `barrier` tagom u ovoj koridor oblasti, samo 21 je zaista povezano na neku way ivicu KOJA JE ZADRŽANA ovim filterom (motorway/trunk/primary/secondary) — i svih 21 su `toll_booth`/`jersey_barrier`, nijedan sa `maxheight`/`maxweight`. Postoje 3 barrier+maxheight node-a u širem bbox-u, ali sva tri su na way-ovima koji NISU u ovoj klasi puteva (parking ulaz, pešačka/biciklistička prepreka) — realan nalaz, ne bag: fizičke visinske barijere (rampe, kapije) se u praksi retko postavljaju direktno na auto-put/magistralu. Mehanizam ekskluzije je dokazan sintetičkim testom (isti obrazac kao way-level ekskluzija); Novi Banovci slučaj i dalje nije reprodukovan ni sa node barijerama.
