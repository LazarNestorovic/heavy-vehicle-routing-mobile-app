# Bounded A*/Dijkstra modul (sopstveni algoritam nad OSM podacima)

**Datum:** 2026-07-21
**Fajlovi:** [`backend/internal/algorithm/`](../../backend/internal/algorithm/)

## Šta je dodato

Poslednja stavka nedelje 2 iz `SPECIFIKACIJA.md` (sekcija 3.3.2) — jezgro algoritamskog doprinosa za rad: sopstvena implementacija A* i Dijkstra nad grafom učitanim **direktno iz OSM-a** (ne preko Valhalla-e), sa custom cost funkcijom svesnom fizičkih ograničenja vozila. **Nije** deo produkcionog `/routes` puta (Valhalla ostaje za to) — služi za evaluaciju i demonstraciju algoritma u radu.

```
backend/internal/algorithm/
  graph.go          — Node, Edge, Graph, NearestNode (linear scan), haversineMeters
  loader.go         — LoadOSMXML: parsira OSM XML (encoding/xml, bez eksterne PBF biblioteke)
  cost.go           — VehicleProfile, allowed() (hard exclusion po visini/težini/hazmat), cost() (dužina + surface penalty)
  dijkstra.go       — Dijkstra + deljena search() funkcija (container/heap)
  astar.go          — A* sa haversine-do-cilja heuristikom (ista search() funkcija)
  algorithm_test.go — sintetički testovi (determinstički) + testovi nad stvarnim koridor podacima
  testdata/beograd-novisad-corridor.osm — fiksni fixture, vidi guide za regeneraciju
```

## Zašto OSM XML, ne PBF direktno

Nema Go PBF parsing biblioteke u zavisnostima (izbegnuto namerno). Umesto toga: `osmium extract` + `osmium tags-filter` + `osmium cat -f osm` (isti alat koji već koristimo u `update-osm.sh`) pretvara ekstrakt u OSM XML, koji Go parsira sa `encoding/xml` iz standardne biblioteke — nula novih zavisnosti. Postupak je u [`documentations/guides/extract-osm-corridor.md`](../guides/extract-osm-corridor.md).

## Bitan nalaz tokom testiranja: A* na veštačkim koordinatama može biti neadmisibilan

Prva verzija sintetičkog testa je postavila čvorove na koordinate razmaknute na kilometarskoj skali dok su cost vrednosti ivica bile u metrima (10m, 5m) — haversine heuristika je time enormno precenjivala stvarnu preostalu cenu, što je A* učinilo neadmisibilnim za taj veštački graf i test je **stvarno pukao** (A* je vratio 20 umesto optimalnih 10). Ispravka: sintetički graf koristi identične koordinate za sve čvorove (heuristika = 0, trivijalno admisibilna), dok se prava admisibilnost heuristike dokazuje na stvarnim OSM koordinatama u `TestRealCorridor_AStarMatchesDijkstra`. Ovo je vredan nalaz za poglavlje o algoritmu u radu — direktna demonstracija razumevanja uslova admisibilnosti heuristike kod A*.

## Verifikacija — 9 testova, svi prolaze

Sintetički (determinstički, `buildDiamondGraph`):
- `TestDijkstra_TakesShortestUnrestrictedPath` — niže vozilo koristi kraći, visinski ograničen prečac
- `TestDijkstra_ExcludesEdgeAboveHeightLimit` — više vozilo primorano na duži zaobilazak
- `TestDijkstra_NoPathWhenAllEdgesExcluded` — graf bez validne rute vraća grešku, ne pogrešan rezultat
- `TestAStar_MatchesDijkstra` — A* i Dijkstra daju isti optimalni cost na oba profila
- `TestParseMeters`, `TestHaversineMeters_*` — pomoćne funkcije

Nad stvarnim koridor podacima (`beograd-novisad-corridor.osm`, 35 478 čvorova, ~50 700 ivica, sve motorway/trunk/primary/secondary klase u bbox-u Beograd-Novi Sad):
- `TestRealCorridor_DijkstraFindsPlausibleRoute` — 80.6km, 1899 čvorova na putanji (Valhalla je za istu tačku-do-tačke rutu davala 84.8-92.7km — u istom redu veličine, razlika očekivana jer ovaj graf ima samo glavne klase puteva i prostiju cost funkciju)
- `TestRealCorridor_HeightRestrictionExcludesRealTaggedEdge` — koristi **stvaran** `maxheight=4.3` way kod centra Beograda (44.808, 20.429): vozilo od 4.0m nalazi rutu (4352m), vozilo od 4.5m dobija `no path found` jer u ovom bounded grafu nema alternative preko glavnih puteva
- `TestRealCorridor_AStarMatchesDijkstra` — potvrđuje admisibilnost heuristike i na stvarnim koordinatama/cost vrednostima

## Zašto ovaj modul NE reprodukuje tačno Novi Banovci slučaj

Namerno pokušano i **nije uspelo** — vredan nalaz, ne sakriven:
- Sa istim koordinatama (Beograd → Novi Sad) i profilima (height 4.0 vs 4.7) kao u [risk-scoring-layer.md](2026-07-21-risk-scoring-layer.md), ovaj graf vraća **istu** distancu (75.89km) za oba — nema detour-a.
- Provereno direktno: u bbox-u oko Novih Banovaca (44.85-44.95, 20.2-20.35) **nema nijedne** `maxheight`-tagovane way ivice u ovom ekstraktu.
- Dva verovatna razloga (nijedan nije potvrđen sa sigurnošću, oba vredna future work): (1) stvarna prepreka je možda tagovana kao node-level `barrier` (npr. `barrier=height_restrictor`), a `LoadOSMXML` trenutno čita **samo way tagove** — node barijere se ignorišu; (2) Valhalla-in trošak uzima u obzir tip puta/skretanja/tolls, pa njena "optimalna" ruta nije nužno geometrijski najkraća — naša prosta cost funkcija (dužina + surface penalty) možda bira potpuno drugu putanju koja tu konkretnu ivicu i ne dodiruje.

Zbog ovoga je test za ekskluziju prebačen na drugi, stvaran `maxheight=4.3` slučaj kod Beograda gde postoji jasna, izolovana veza — dokazuje mehanizam ekskluzije na pravim OSM podacima bez oslanjanja na to da baš ta specifična Banovci ivica bude na izračunatoj putanji.

## Šta namerno NIJE urađeno

- **Nije deo produkcionog `/routes` puta.** Ostaje eksperimentalni/evaluacioni modul za rad, kako je i planirano u `SPECIFIKACIJA.md` 3.3.2.
- **Node-level barijere se ne čitaju** (samo way tagovi) — vidi nalaz iznad.
- **`NearestNode` je linearna pretraga** (35K čvorova, dovoljno brzo za bounded graf) — ne bi skalirala na nacionalni nivo bez prostornog indeksa (grid/k-d tree); nije problem jer ovaj modul namerno ostaje bounded.
- **Cost funkcija je namerno prosta** (dužina + surface penalty) — nema turn penalty, road-class preferencu, hazmat-proximity itd.; poređenje sa Valhalla-inim bogatijim modelom je deo priče u radu, ne nedostatak koji treba ispraviti ovde.
