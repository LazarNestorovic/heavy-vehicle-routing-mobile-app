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
- ~~Node-level barijere se ne čitaju (samo way tagovi)~~ — zatvoreno 2026-07-26, vidi ispod.
- **`NearestNode` je linearna pretraga** (35K čvorova, dovoljno brzo za bounded graf) — ne bi skalirala na nacionalni nivo bez prostornog indeksa (grid/k-d tree); nije problem jer ovaj modul namerno ostaje bounded.
- ~~Cost funkcija je namerno prosta (nema turn penalty, road-class preferencu)~~ — delimično zatvoreno 2026-07-26 (road-class + turn penalty dodati), vidi ispod. Hazmat-proximity i dalje nedostaje.

## Dopuna 2026-07-26 — node-level barijere + bogatija cost funkcija

Korisnik je posle prve odbrane/pregleda dokumentacije zatražio da se zatvore preostali "namerno NIJE urađeno" stavovi kroz projekat, uz eksplicitnu potvrdu da menjanje algoritamske priče u radu (uključujući ponovnu evaluaciju Novi Banovci slučaja) nije problem.

### Node-level barijere

`LoadOSMXML` (`loader.go`) sada čita i node tagove. Kada node ima `barrier` tag ZAJEDNO sa `maxheight`/`maxweight` (standardna OSM konvencija za `barrier=height_restrictor`/`lift_gate`), ta ograničenja se upisuju u `Node.MaxHeightM`/`MaxWeightT` i primenjuju preko novog `nodeAllowed()` (cost.go) — vozilo se sada može isključiti i zbog prepreke NA čvoru, ne samo na ivici.

**Ekstrakcija je regenerisana** (`documentations/guides/extract-osm-corridor.md` korak 2 dobio `n/barrier` pravilo) — `osm-data/filtered/serbia-hvt.osm.pbf` je već imao node barrier tagove (iz [osmium fix-a](../fixes/2026-07-21-osmium-filter-node-tags.md)), ali koridorska ekstrakcija ih je do sada odbacivala u drugom filter prolazu.

**Pošten nalaz nad regenerisanim podacima**: od 2808 `barrier`-tagovanih node-ova u koridoru, samo 21 je zaista povezano na way ivicu koja je zadržana ovim filterom (motorway/trunk/primary/secondary) — i svih 21 su `toll_booth`/`jersey_barrier`, nijedan sa `maxheight`/`maxweight`. Tri barrier+maxheight node-a postoje u širem bbox-u, ali sva tri su na way-ovima van ove klase puteva (parking ulaz, pešačka prepreka). **Novi Banovci slučaj i dalje nije reprodukovan** — realan, dokumentovan nalaz (fizičke visinske barijere se u praksi retko postavljaju direktno na magistralu), ne bag. Mehanizam ekskluzije je ipak dokazan sintetičkim testom (`TestDijkstra_NodeBarrierExcludesTallVehicle`) i parsiranje potvrđeno nad stvarnim podatkom (`TestRealCorridor_ParsesNodeBarrierHeightTag`, node `11742525355` kod centra Beograda, `maxheight=2.2`).

### Bogatija cost funkcija

- **Road-class preferenca** — `Edge.RoadClass` (iz way `highway` taga, ranije parsiran ali odbačen) sad ulazi u `cost()` preko `roadClassMultiplier` mape (motorway 0.85× ... secondary_link 1.2×) — direktna paralela Valhalla-inom `highway_ratio` signalu, samo kao multiplikator umesto post-hoc score člana.
- **Turn penalty** — aproksimacija preko već postojećeg `prev` pokazivača u Dijkstra/A* (`search()`, dijkstra.go): ugao skretanja između `bearing(prev[current]→current)` i `bearing(current→edge.To)` (nov `bearing()` helper, graph.go) dodaje penalty (0/15/60/120 "metara-ekvivalenta", zavisno od oštrine ugla). **Svesno pojednostavljenje**: koristi trenutno najbolji poznat put do `current`, ne pun edge-based graf — redak slučaj gde bi suboptimalan put do `current` dao bolji ukupan trošak skretanja nije pokriven; isti stil iskrenog kompromisa kao `PointAtFraction`-ova pretpostavka konstantne brzine.

**Verifikacija**: 14 testova u `internal/algorithm` (9 postojećih + 5 novih: node barijera dozvoljava/isključuje, road-class preferenca, turn penalty, real-data node tag parsing), svi prolaze. `go build`/`vet`/`test` čisto za ceo backend.
