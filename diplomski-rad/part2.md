# 4 Arhitektura sistema

## 4.1 Pregled arhitekture i tok podataka

Sistem se sastoji od pet nezavisnih procesa: Flutter mobilne aplikacije (klijent, vozač ili dispečer), Go backend servisa (centralna tačka logike), Valhalla routing engine-a, PostgreSQL/PostGIS baze podataka i RabbitMQ posrednika poruka. Slika 4.1 prikazuje njihov međusobni odnos i osnovni tok podataka pri planiranju i praćenju jedne ture.

[DIAGRAM:architecture]

**Slika 4.1.** Arhitektura sistema — komponente i tok podataka

Tok pri planiranju rute je sledeći: mobilna aplikacija šalje `POST /api/v1/routes` sa profilom vozila, polazištem i odredištem; backend mapira profil vozila u Valhalla-in `truck` costing model (poglavlje 6.1) i traži do tri alternativne rute; svaka alternativa se rangira sopstvenom funkcijom rizika (poglavlje 6.2), a za izabranu (najbolje rangiranu) rutu se, ako odstupa od geometrijski najkraćeg puta, generiše tekstualno objašnjenje (poglavlje 6.3); odgovor sa geometrijom, rizikom i objašnjenjem se vraća klijentu. Kada vozač zaista pokrene turu (`POST /api/v1/trips`, zatim prelazak u status `in_progress`), backend objavljuje `trip.started` događaj na RabbitMQ, koji worker procès preuzima i računa predlog pauze (poglavlje 7.4); istovremeno se otvara WebSocket veza (`GET /ws/trips/{id}`) preko koje se, dok vozač šalje periodične GPS pozicije (`POST /api/v1/trips/{id}/position`), te pozicije prosleđuju svim zainteresovanim klijentima uživo — vozačevom sopstvenom ekranu i, ako postoji, njegovom dispečeru koji sistem preko sopstvene, paralelne WebSocket konekcije prati na jednoj mapi (poglavlje 7.5, 8.2).

Sistem sledi troslojnu podelu odgovornosti sličnu klasičnoj veb arhitekturi [TANENBAUM]: sloj prezentacije (Flutter aplikacija), sloj obrade (Go backend, uključujući scoring i explain module) i sloj perzistencije (PostgreSQL). Razlika u odnosu na uobičajenu troslojnu veb aplikaciju je što backend za svaki zahtev za rutu sam postaje klijent eksternog servisa (Valhalla), a deo obrade (predlog pauze) je pomeren u nezavisan, asinhron proces (worker), radi rasterećenja glavne HTTP niti obrade zahteva.

## 4.2 Kontejnerizacija i orkestracija servisa

Svi serverski delovi sistema definisani su u jednom `docker-compose.yml` fajlu i pokreću se komandom `docker compose up`. Listing 4.1 prikazuje definiciju PostgreSQL servisa, sa health-check-om koji ostali servisi (backend) koriste kao uslov za sopstveni start preko `depends_on: condition: service_healthy` — čime se garantuje da backend ne pokuša konekciju na bazu koja još nije spremna za prihvat konekcija.

```yaml
postgres:
  image: postgis/postgis:16-3.4-alpine
  environment:
    POSTGRES_USER: hvr
    POSTGRES_PASSWORD: hvr_dev_password
    POSTGRES_DB: hvr
  ports:
    - "5433:5432"
  volumes:
    - pgdata:/var/lib/postgresql/data
  healthcheck:
    test: ["CMD-SHELL", "pg_isready -U hvr -d hvr"]
    interval: 10s
    timeout: 5s
    retries: 5
```

**Listing 4.1.** Definicija PostgreSQL/PostGIS servisa u `docker-compose.yml`

Tabela 4.1 sumira sve servise definisane u `docker-compose.yml`, njihovu ulogu i portove koje izlažu na host mašini.

**Tabela 4.1.** Servisi definisani u `docker-compose.yml`

| Servis | Slika | Uloga | Port (host) |
|---|---|---|---|
| `postgres` | `postgis/postgis:16-3.4-alpine` | Relaciona baza + PostGIS ekstenzija | 5433 → 5432 |
| `rabbitmq` | `rabbitmq:3-management-alpine` | Posrednik poruka (AMQP) + web konzola | 5672, 15672 |
| `valhalla` | `ghcr.io/valhalla/valhalla:latest` | HTTP routing servis (`valhalla_service`) | 8002 |
| `valhalla-build` | `ghcr.io/valhalla/valhalla:latest` | Jednokratna izgradnja grafa iz OSM ekstrakta (profil `build`, ne pokreće se pri običnom `up`) | — |
| `backend` | sopstveni Go build | REST API, WebSocket gateway, worker | 8080 |

`valhalla-build` je definisan sa Docker Compose *profilom* `build`, što znači da se ne pokreće automatski pri `docker compose up` — pokreće se izričito (`docker compose --profile build up valhalla-build`) samo kada treba ponovo izgraditi graf (npr. nakon promene OSM ekstrakta, poglavlje 5), i radi sa `network_mode: none` jer mu za samu izgradnju grafa nije potrebna mrežna konekcija, čime se smanjuje površina za eventualni bezbednosni problem tog koraka. Mobilna aplikacija i sopstveni algoritamski modul (poglavlje 6.4) nisu deo `docker-compose.yml` — Flutter aplikacija se pokreće lokalno na razvojnoj mašini ili emulatoru, a algoritamski modul je čista Go biblioteka koja se pokreće isključivo kroz testove, bez sopstvenog servisa (obrazloženo u poglavlju 6.4).

## 4.3 Sigurnost i autentifikacija

Autentifikacija vozača i dispečera zasniva se na JWT tokenima (RFC 7519 [JWT]) potpisanim HMAC SHA-256 algoritmom (HS256). Nakon prijave (`POST /auth/login` sa korisničkim imenom i lozinkom, ili `POST /auth/google` sa Google ID tokenom), backend izdaje token koji, pored identifikatora vozača, nosi i celobrojno polje `token_version`. Ovo polje se upoređuje sa vrednošću `token_version` iz baze na svakom autentifikovanom zahtevu (`RequireAuth` middleware) — kada vozač zatraži odjavu sa svih uređaja (`POST /auth/logout-all`), backend samo inkrementira `token_version` u bazi, čime svi ranije izdati tokeni (koji nose stariju vrednost) trenutno postaju nevažeći, bez potrebe za posebnom, rastućom tabelom opozvanih tokena (*blocklist*) koja bi morala da se proverava na svakom zahtevu.

Lozinke se ne čuvaju u čistom tekstu, već kao bcrypt heš, algoritam koji ugrađuje nasumičnu "sol" po lozinci (čime dve identične lozinke dva različita korisnika dobijaju različit heš) i podesivi faktor rada (koji direktno određuje koliko dugo traje jedan pokušaj heširanja, pa time i koliko dugo traje jedan pokušaj brute-force napada). Nalozi kreirani preko Google prijave nemaju lozinku (`password_hash` je `NULL`) — identifikuju se preko stabilnog Google identifikatora naloga (`google_sub`, `sub` polje iz Google ID tokena), koji backend verifikuje samostalno preko Google-ovog javnog skupa ključeva (JWKS), bez uvođenja Firebase SDK-a kao zavisnosti.

Za obnovu zaboravljene lozinke (`POST /auth/forgot-password`) i verifikaciju email adrese (`GET /auth/verify-email`) koristi se model jednokratnog, vremenski ograničenog tokena zapisanog u posebnoj tabeli (`password_reset_tokens`, `email_verification_tokens`) sa poljima `expires_at` i `used_at` — link poslat emailom prestaje da važi i po isteku vremena i po jednokratnoj upotrebi, a backend razlikuje ta dva slučaja radi jasnije poruke korisniku. Slanje emaila (link za verifikaciju/reset) realizovano je preko `net/smtp` paketa iz Go standardne biblioteke, bez eksternog transakcionog email API-ja — servis je no-op (ne šalje ništa, samo zapisuje u log) ako promenljiva okruženja `SMTP_HOST` nije podešena, čime ostatak sistema radi i bez konfigurisanog email servisa (npr. u razvojnom okruženju).

Sistem u trenutnom obliku ima namerno ograničen bezbednosni obim, obrazložen u poglavlju 10 — WebSocket `CheckOrigin` provera je trenutno propustljiva (prihvata sve origin-e), pošto je aplikacija zamišljena kao rad sa jednim poznatim, poverljivim klijentom (sopstvena Flutter aplikacija), a ne kao javni, višezakupčani (multi-tenant) servis.

<!-- PAGEBREAK -->

# 5 Priprema geografskih podataka — OSM ekstrakcija za teretna vozila u Srbiji

## 5.1 Izvor podataka i osmium filter

Osnovni izvor geografskih podataka je Geofabrik-ov dnevno ažuriran ekstrakt OpenStreetMap podataka za Srbiju (`serbia-latest.osm.pbf`) [GEOFABRIK], preuzet automatizovanim `update-osm.sh` skriptom koja, pored preuzimanja, provera integritet fajla preko MD5 kontrolne sume koju Geofabrik objavljuje uz sam ekstrakt. Kompletan `.osm.pbf` fajl za Srbiju sadrži ogroman broj tagova nepotrebnih za rutiranje teretnih vozila (npr. sve poljoprivredne parcele, granice popisnih krugova, turističke oznake), pa se filtrira alatom `osmium-tools` [OSMIUM] pre nego što se prosledi Valhalla-i.

Listing 5.1 prikazuje tačnu `osmium tags-filter` komandu korišćenu u projektu.

```bash
osmium tags-filter \
  serbia-latest.osm.pbf \
  w/highway w/maxheight w/maxweight w/maxwidth \
  w/hgv w/hazmat w/bridge w/tunnel \
  w/surface w/maxspeed \
  n/amenity=fuel,parking n/highway=rest_area n/barrier \
  r/type=restriction \
  --output serbia-hvt.osm.pbf --overwrite
```

**Listing 5.1.** Osmium filter za ekstrakciju podataka relevantnih za teretni saobraćaj

Filter čuva way-eve (`w/`) sa tagovima relevantnim za geometriju i ograničenja puta (`highway`, `maxheight`, `maxweight`, `maxwidth`, `hgv`, `hazmat`, `bridge`, `tunnel`, `surface`, `maxspeed`), relacije (`r/`) tipa `restriction` (ograničenja skretanja) i, što je bilo kritično ispraviti tokom razvoja (videti dalje u ovom poglavlju), node-ove (`n/`) sa tagovima `amenity=fuel`, `amenity=parking`, `highway=rest_area` i `barrier`. Way i node su, kako je objašnjeno u poglavlju 2.4, dva različita nivoa na kojima OSM podaci nose ograničenja relevantna za teretna vozila — way-evi obično nose ograničenje za celu deonicu (npr. `maxheight` na mostu), a node-ovi tačkastu prepreku (npr. rampu ili stub na ulazu u parking).

**Napomena o razvoju filtera.** Prva verzija filtera korišćena u ranoj fazi razvoja sadržala je samo `w/` i `r/` pravila, bez ijednog `n/` pravila — što je značilo da su svi node-ovi sa `amenity=fuel`, `amenity=parking`, `highway=rest_area` i `barrier` tagovima bili odbačeni pri filtriranju, iako su neophodni za modul predloga pauze vozača (poglavlje 7.4). Ovaj propust je otkriven i ispravljen tokom pripreme podataka za rad — dodavanjem tri `n/` pravila filtrirani fajl je narastao sa 68 121 223 na 68 402 528 bajtova, a Valhalla graf je ponovo izgrađen sa ispravljenim skupom podataka. Ovaj slučaj ilustruje opštu lekciju vezanu za pripremu OSM podataka: filter koji izgleda kompletan gledano samo kroz way tagove tiho odbacuje čitavu kategoriju informacija zavedenih na node nivou, a greška se ne manifestuje kao pad sistema već kao *nedostajuća* funkcionalnost koja se lako previdi dok se posebno ne testira.

## 5.2 Izgradnja Valhalla grafa

Filtriran `.osm.pbf` fajl se prosleđuje alatu `valhalla_build_tiles`, koji ga parsira i izgrađuje hijerarhijski graf organizovan u pločice (*tiles*) — trajni format koji Valhalla HTTP servis (`valhalla_service`) učitava pri startu i koristi za sve naredne `/route` zahteve. Ovaj korak je u sistemu izdvojen u poseban Docker Compose servis (`valhalla-build`, poglavlje 4.2), koji se pokreće samo kada treba ponovo izgraditi graf, a ne pri svakom pokretanju sistema (jer izgradnja traje nekoliko minuta i, s obzirom na to da se OSM podaci ne menjaju u toku jedne demonstracije rada, nije potrebno da se ona izvršava pri svakom `docker compose up`).

Nakon ispravke filtera opisane u 5.1, ponovo izgrađen graf za teritoriju Republike Srbije sadrži 898 337 čvorova i 2 216 948 usmerenih ivica (grana). Ovaj graf koristi isključivo Valhalla, preko HTTP `/route` poziva (poglavlje 6.1) — nije direktno dostupan Go backend-u niti bilo kom drugom delu sistema, čime se čitava logika parsiranja OSM podataka i izgradnje efikasne rutne strukture prepušta Valhalla-i, u skladu sa odlukom (obrazloženom u poglavlju 6) da se produkcioni put ne oslanja na sopstvenu implementaciju pretrage nad grafom cele države.

## 5.3 Ekstrakcija podataka o odmaralištima

Za modul predloga pauze vozača (poglavlje 7.4) backend ne poziva Valhallu, već direktno učitava, pri pokretanju, sve node-ove sa tagovima `amenity=fuel`, `amenity=parking` i `highway=rest_area` iz filtriranog OSM ekstrakta (istog fajla korišćenog za izgradnju Valhalla grafa, poglavlje 5.1) u memoriju procesa — oko 2000 node-ova za teritoriju Srbije. Ova lista se pri svakom predlogu pauze pretražuje geometrijski: kandidat mora biti unutar zadatog radijusa od tačke na ruti u kojoj bi vozilo teorijski bilo nakon isteka praga vožnje (poglavlje 7.4), i mora zaista biti u koridoru trase (a ne, na primer, geometrijski najbliža tačka vazdušnom linijom koja se nalazi na potpuno drugom putu). Rezultat pretrage dodatno favorizuje omiljene ili brend-specifične lokacije vozača, a za vozila koja prevoze opasan teret preferira benzinske stanice nad parkinzima u granicama tolerancije rastojanja — detalji ove logike dati su u poglavlju 7.4.

<!-- PAGEBREAK -->

# 6 Algoritam rutiranja prilagođen vozilu

Ovo poglavlje je centralni algoritamski doprinos rada. Sistem koristi **dvoslojni pristup**: (1) u produkcionom putu, Valhalla generiše fizički dopustive alternativne rute preko svog `truck` costing modela (6.1), nad kojima sopstveni Go modul dodaje rangiranje po riziku (6.2) i objašnjenje odstupanja rute (6.3), pošto to Valhalla nativno ne radi; (2) kao odvojena, algoritamska celina namenjena evaluaciji i demonstraciji principa, implementirana je sopstvena Dijkstra/A* pretraga nad ograničenim podgrafom putne mreže, direktno nad OSM podacima, bez ikakve zavisnosti od Valhalla-e (6.4). Ovaj pristup je posledica realne procene obima rada: reimplementacija efikasnog nacionalnog routing engine-a nad grafom od skoro milion čvorova (poglavlje 5.2) nije realan cilj jednog diplomskog rada, dok je razumevanje i demonstracija principa na kojima takav engine počiva — kroz sopstvenu, testiranu implementaciju nad manjim, ali realnim podgrafom — sasvim ostvarivo i akademski relevantno.

## 6.1 Valhalla truck costing

Valhalla-in `truck` costing model prihvata profil vozila kao deo tela HTTP `/route` zahteva, bez potrebe da se graf iznova izgradi za svako vozilo. Listing 6.1 prikazuje tipičan zahtev korišćen u ovom sistemu.

```json
POST /route
{
  "locations": [
    {"lat": 44.8, "lon": 20.4},
    {"lat": 45.25, "lon": 19.85}
  ],
  "costing": "truck",
  "costing_options": {
    "truck": {
      "height": 4.0,
      "width": 2.55,
      "length": 16.5,
      "weight": 40,
      "axle_load": 11.5,
      "hazmat": false
    }
  },
  "alternates": 2
}
```

**Listing 6.1.** Zahtev za truck-costed rutu sa alternativama (visina/širina/dužina u metrima, masa/osovinsko opterećenje u tonama)

Backend mapira sopstveni `TruckProfile` (u SI jedinicama — metrima i kilogramima, radi konzistentnosti sa ostatkom sistema) u Valhalla-ine jedinice (metri za dimenzije, metrički toni za masu), i parsira odgovor u sopstvenu strukturu `RouteCandidate` koja, pored rastojanja i trajanja, izdvaja i broj manevara, udeo puta na magistralnim/autoputskim deonicama (`HighwayRatio`), prisustvo trajekta ili putarine, kao i nazive ulica i tačke svakog manevra — signale koje ne izlaže sam sažetak rute, ali su neophodni za sloj rangiranja (6.2) i objašnjenja (6.3). Listing 6.2 prikazuje deo implementacije koja šalje zahtev i mapira profil vozila.

```go
// TruckProfile mirrors Valhalla's truck costing_options, in SI units (meters, kilograms).
type TruckProfile struct {
	HeightM    float64
	WidthM     float64
	LengthM    float64
	WeightKg   float64
	AxleLoadKg float64
	Hazmat     bool
}

func (c *Client) RouteAlternates(ctx context.Context, origin, destination LatLon,
	profile TruckProfile, numAlternates int) ([]RouteCandidate, error) {
	body := routeRequest{
		Locations: []LatLon{origin, destination},
		Costing:   "truck",
		CostingOptions: map[string]truckCostingOpt{
			"truck": {
				Height:   profile.HeightM,
				Width:    profile.WidthM,
				Length:   profile.LengthM,
				Weight:   profile.WeightKg / 1000,   // Valhalla expects metric tons
				AxleLoad: profile.AxleLoadKg / 1000,
				Hazmat:   profile.Hazmat,
			},
		},
		Alternates: numAlternates,
	}
	// ... serijalizacija, HTTP POST na baseURL+"/route", parsiranje odgovora
}
```

**Listing 6.2.** `valhalla.Client.RouteAlternates` — mapiranje profila vozila u Valhalla `costing_options` (`backend/internal/valhalla/client.go`)

Ograničenje ovog pristupa je da odgovor običnog `/route` poziva ne izlaže *koja tačno* ivica grafa je isključena niti njenu tačnu vrednost ograničenja (npr. tačnu visinu nadvožnjaka) — ta informacija bi zahtevala poziv Valhalla-inog `/trace_attributes` endpoint-a nad već izračunatom rutom, ili direktan pristup OSM podacima nezavisno od Valhalla-e. Ovo ograničenje je direktno motivisalo modul opisan u 6.4, koji, radeći nad sopstvenim, manjim grafom, ima pristup tačnim OSM tagovima svake ivice.

## 6.2 Prilagođena funkcija rizika (scoring)

Valhalla po zahtevu vraća do tri alternativne rute, ali ih ne rangira po bilo kom kriterijumu specifičnom za teretni saobraćaj — sve tri su, sa njene tačke gledišta, fizički dopustive. Paket `scoring` dodaje taj nedostajući sloj: svaku alternativu bodira heurističkom funkcijom rizika koja kombinuje rastojanje, broj manevara, udeo puta van magistralnih deonica, prisustvo trajekta/putarine, procenu potrošnje goriva u odnosu na masu vozila, broj "oštrih" manevara (proxy za rizik pomeranja tereta) i blizinu omiljenih/brend benzinskih stanica vozača — a zatim vraća alternative sortirane od najboljeg (najmanji rizik) ka najgorem rezultatu.

Svaka dimenzija skalira se prema preferenci vozača, izraženoj kao ceo broj od 1 do 5 (gde je 3 neutralna, podrazumevana vrednost) preko faktora `priority/3` — vozač koji nikad ne podesi svoje preference dobija rezultat identičan fiksnoj, uravnoteženoj formuli. Listing 6.3 prikazuje centralnu funkciju bodovanja.

```go
func score(c valhalla.RouteCandidate, prefs Preferences, vehicleWeightKg,
	fastestDurationMin float64, preferredStops []valhalla.LatLon) float64 {
	var timeTerm float64
	if fastestDurationMin > 0 {
		timeTerm = (c.DurationMin - fastestDurationMin) / fastestDurationMin * timeScale
	}
	highwayTerm := (1 - c.HighwayRatio) * nonHighwayScale
	fuelTerm := fuelProxy(c, vehicleWeightKg)
	cargoTerm := float64(c.SharpManeuverCount) * sharpManeuverWt

	s := scalar(prefs.TimePriority)*timeTerm +
		scalar(prefs.HighwayPriority)*highwayTerm +
		scalar(prefs.FuelPriority)*fuelTerm +
		scalar(prefs.CargoPriority)*cargoTerm +
		float64(c.ManeuverCount)*maneuverWeight

	if c.HasFerry {
		s += ferryPenalty
	}
	if c.HasToll {
		s += tollPenalty
	}
	s += preferredStopDiscount(c.Shape, preferredStops)
	return s
}
```

**Listing 6.3.** Funkcija bodovanja rute (`backend/internal/scoring/scoring.go`) — manji rezultat je bolji

Vrednost `timeTerm` je relativno kašnjenje kandidata u odnosu na najbrži kandidat u istom skupu alternativa (0% ako je kandidat najbrži), a `fuelProxy` je relativna procena "potrošnje" izvedena iz rastojanja i mase vozila (nema pristupa podacima o nagibu puta, pa ovo nije stvarna potrošnja u litrima, već relativan signal za poređenje kandidata međusobno). Autor koda je u komentarima izričito naglasio da su bazne težine (`maneuverWeight`, `nonHighwayScale`, `ferryPenalty` itd.) prva heuristička procena, nekalibrisana prema stvarnim podacima o potrošnji ili nesrećama — što je pošteno i namerno ograničenje, razmatrano dalje u poglavlju 9.3 kroz konkretan primer greške koju je ova formula ispravila.

**Motivacija dodavanja vremenskog člana.** Rana verzija formule nije sadržala `timeTerm` — bodovala je isključivo na osnovu udela magistralne deonice (`highwayTerm`) i broja manevara. Testiranjem na konkretnoj ruti (Radalj, poglavlje 9.3) uočeno je da takva formula bira rutu koja je 47% duža i 30% sporija od alternative samo zato što ima nešto povoljniji odnos magistralne i lokalne deonice puta — očigledno pogrešan ishod sa stanovišta stvarnog transportnog troška. Dodavanje `timeTerm`-a, koji direktno kažnjava kandidata srazmerno tome koliko je sporiji od najbržeg u skupu, ispravilo je taj slučaj bez uklanjanja postojećih članova formule.

## 6.3 Objašnjenje predložene rute — slučaj Novi Banovci

Kada izabrana ruta odstupa od geometrijski najkraćeg puta, vozaču nije dovoljno prikazati *šta* je predloženo — potrebno mu je objašnjenje *zašto*. Paket `explain` implementira taj mehanizam prema principu binarne pretrage po dimenzijama profila vozila:

1. Zahteva se "referentna" ruta sa svim dimenzijama profila veštački opuštenim na nerealno velike vrednosti (npr. visina 100m, masa 900 000 kg) — ruta koju bi Valhalla predložila da fizička ograničenja vozila ne postoje.
2. Ako se izabrana ruta (sa stvarnim profilom vozila) po rastojanju ne razlikuje značajno od referentne (manje od 1km), nema odstupanja koje treba objasniti.
3. U suprotnom, dimenzije profila (visina, masa, osovinsko opterećenje, širina, prevoz opasnog tereta) se redom, jedna po jedna, "oslobađaju" na referentnu vrednost, i ruta se ponovo traži — dimenzija čije oslobađanje vrati rutu na rastojanje referentne rute je **vezujuće ograničenje** za taj segment puta.
4. Mesto odstupanja se određuje geometrijski: dekodirane geometrije (polyline) izabrane i referentne rute se upoređuju tačku po tačku, a prva tačka izabrane rute čija je udaljenost od svake tačke referentne rute veća od 200m proglašava se tačkom divergencije; njoj se pridružuje naziv ulice najbližeg manevra izabrane rute.
5. Generiše se poruka vozaču, npr.: *"Ruta skreće kod [ulica] jer visina vozila (4.7m) ne zadovoljava ograničenje na toj deonici."*

[DIAGRAM:explain]

**Slika 6.1.** Tok mehanizma objašnjenja rute (`explain.Explain`)

**Slučaj Novi Banovci.** Ovaj mehanizam je razvijen i potvrđen na konkretnom, stvarnom slučaju: ručnom binarnom pretragom (ponavljanjem `/route` poziva sa različitim vrednostima visine) utvrđeno je da na auto-putu A1, u okolini čvora Novi Banovci (izlaz 21/22), postoji ograničenje visine između 4.5m i 4.6m. Za vozilo profila korišćenog u testu (visina 4.7m), Valhalla je korektno birala detour preko lokalnog puta umesto direktnog nastavka auto-putem — a mehanizam opisan u ovom poglavlju automatizuje upravo tu istu binarnu pretragu, koju je autor prvo sproveo ručno da bi potvrdio da je ponašanje Valhalla-e (i, posledično, sistema) tačno.

Bitna napomena o ovom mehanizmu je jedna poznata, dokumentovana greška otkrivena tokom razvoja i potom ispravljena: rana verzija je poredila listu manevara (`street_names`) izabrane i referentne rute **po pozicionom indeksu u nizu** (manevar broj $i$ izabrane rute sa manevrom broj $i$ referentne rute). Ovo je radilo dobro kada su rute delile zajednički početni deo puta i lokalno se razdvajale, ali je davalo pogrešnu (preraniju) lokaciju za vozila sa toliko strogim ograničenjem da Valhalla za njih bira **globalno drugačiju strategiju rute od samog početka** — tada indeks $i$ ne odgovara istom mestu na putu kod obe rute. Zamena poređenja po indeksu geometrijskim poređenjem (korak 4 gore) rešila je ovaj problem, jer geometrija ne zavisi od toga kako su manevri indeksirani, već isključivo od toga gde se putevi fizički razilaze. Ovaj slučaj je vredan pomena u radu jer ilustruje opštu lekciju: kada se dva niza podataka (manevri dve različite rute) upoređuju da bi se pronašla tačka razlike, poređenje po **pozicionoj strukturi** (indeksu) je lažno pouzdano — tačno je samo dok se strukture podudaraju, a tiho daje pogrešan rezultat kada se globalno razlikuju; poređenje po **sadržaju** (u ovom slučaju, stvarnoj geografskoj geometriji) je robusnije jer ne pretpostavlja da strukture uopšte odgovaraju jedna drugoj.

## 6.4 Sopstvena implementacija pretrage puta nad ograničenim podgrafom

Kao samostalna algoritamska celina, odvojena od produkcionog puta (nije uvezana u `main.go` niti u REST API — koristi se isključivo kroz sopstvene testove), implementirana je sopstvena Dijkstra i A* pretraga najkraćeg puta koja radi direktno nad OSM XML podacima, bez ikakve zavisnosti od Valhalla-e. Svrha ovog modula nije da zameni Valhallu u produkciji (poglavlje 5.2 objašnjava zašto bi reimplementacija efikasnog routing engine-a nad grafom cele države bila nerealan cilj za obim ovog rada), već da (a) demonstrira, kroz sopstveni, testiran kod, principe na kojima Valhalla interno počiva (isključivanje fizički nedopustivih grana, pretraga najkraćeg puta uz heuristiku), i (b) da posluži kao alat za evaluaciju i poređenje Dijkstra i A* pristupa nad istim, realnim podacima (poglavlje 9.2), sa direktnim pristupom OSM tagovima koje obični Valhalla `/route` odgovor ne izlaže (poglavlje 6.1).

**Graf i model podataka.** Graf se učitava iz `.osm` (XML) ekstrakta funkcijom `LoadOSMXML`, koja parsira way-eve sa tagom `highway` u usmerene ivice (`Edge`), čuvajući po ivici dužinu, klasu puta (`RoadClass`), podlogu (`Surface`) i, ako postoje, ograničenja `maxheight`/`maxweight`/`hazmat` preuzeta sa way-a. Node-ovi koji imaju sopstveni `barrier` tag sa `maxheight`/`maxweight` (npr. rampa niže visine na inače prohodnom putu) čuvaju se kao ograničenje na samom čvoru (`Node.MaxHeightM`), odvojeno od ograničenja na ivici — jer, kako je objašnjeno u poglavlju 2.4, OSM te dve vrste ograničenja zavodi na dva različita nivoa.

**Funkcija cene.** Cena prelaska ivice je njena dužina u metrima, uvećana za faktor lošije podloge (`unpaved`, `gravel`, `dirt`, `sett` — 30% penal) i skalirana faktorom preferencije klase puta (Tabela 6.1) — magistralni putevi i auto-putevi imaju blagi popust u ceni (manje raskrsnica/prilaza po kilometru za veliko vozilo), dok sekundarni putevi imaju blagi penal. Na svaki prelaz sa ivice na ivicu (skretanje) dodaje se i penal za oštrinu ugla skretanja, izražen u istoj jedinici ("metar") kao i osnovna cena, radi jednostavnog sabiranja — svi ovi koeficijenti su, kao i kod paketa `scoring` (6.2), heuristička, nekalibrisana prva procena, što je u kodu izričito dokumentovano.

**Tabela 6.1.** Faktor preferencije klase puta (`roadClassMultiplier`)

| Klasa puta (OSM `highway`) | Faktor |
|---|---|
| `motorway` | 0.85 |
| `motorway_link` | 0.90 |
| `trunk` | 0.90 |
| `trunk_link` | 0.95 |
| `primary` | 1.00 |
| `primary_link` | 1.05 |
| `secondary` | 1.15 |
| `secondary_link` | 1.20 |

Listing 6.4 prikazuje isključivanje nedopustivih ivica i čvorova (tvrdo ograničenje, princip identičan Valhalla-inom `truck` costing-u, poglavlje 2.3) i funkciju cene ivice.

```go
func allowed(e Edge, p VehicleProfile) bool {
	if e.MaxHeightM > 0 && p.HeightM > e.MaxHeightM {
		return false
	}
	if e.MaxWeightT > 0 && p.WeightKg/1000 > e.MaxWeightT {
		return false
	}
	if e.Hazmat && p.Hazmat {
		return false
	}
	return true
}

func cost(e Edge) float64 {
	base := e.LengthM
	switch e.Surface {
	case "unpaved", "gravel", "dirt", "sett":
		base *= 1.3
	}
	if mult, ok := roadClassMultiplier[e.RoadClass]; ok {
		base *= mult
	}
	return base
}
```

**Listing 6.4.** Isključivanje nedopustivih ivica i funkcija cene (`backend/internal/algorithm/cost.go`)

**Pretraga.** Dijkstra i A* dele istu implementaciju pretrage (funkcija `search`), sa jedinom razlikom u heuristici prosleđenoj prioritetnom redu: Dijkstra koristi heuristiku koja je uvek nula (ekvivalentno standardnom Dijkstra algoritmu), a A* koristi haversine udaljenost od trenutnog čvora do cilja. Pretraga koristi binarnu gomilu (`container/heap`) kao prioritetni red i prekida se čim je ciljni čvor skinut sa reda, umesto da izračuna najkraći put do svih čvorova u grafu. Listing 6.5 prikazuje deljenu funkciju pretrage.

```go
func search(g *Graph, start, goal int64, profile VehicleProfile,
	heuristic func(node int64) float64) (Result, error) {
	dist := map[int64]float64{start: 0}
	prev := map[int64]int64{}
	pq := &priorityQueue{{node: start, priority: heuristic(start)}}
	heap.Init(pq)
	visited := map[int64]bool{}

	for pq.Len() > 0 {
		current := heap.Pop(pq).(pqItem).node
		if visited[current] {
			continue
		}
		visited[current] = true
		if current == goal {
			return Result{Path: reconstructPath(prev, start, goal), Cost: dist[goal]}, nil
		}
		for _, edge := range g.AdjList[current] {
			if !allowed(edge, profile) || !nodeAllowed(g.Nodes[edge.To], profile) || visited[edge.To] {
				continue
			}
			edgeCost := cost(edge)
			if p, ok := prev[current]; ok {
				edgeCost += turnPenaltyMeters(turnAngle(g, p, current, edge.To))
			}
			newDist := dist[current] + edgeCost
			if old, ok := dist[edge.To]; !ok || newDist < old {
				dist[edge.To] = newDist
				prev[edge.To] = current
				heap.Push(pq, pqItem{node: edge.To, priority: newDist + heuristic(edge.To)})
			}
		}
	}
	return Result{}, errNoPath
}
```

**Listing 6.5.** Deljena implementacija Dijkstra i A* pretrage (`backend/internal/algorithm/dijkstra.go`)

**Poznato, dokumentovano pojednostavljenje.** Penal za skretanje se računa na osnovu **jednog** prethodno fiksiranog puta do trenutnog čvora (`prev[current]`), a ne na osnovu para (čvor, dolazna ivica), koje bi bilo teorijski korektnije (jer bi omogućilo da se do istog čvora stigne različitim uglovima dolaska i time različitim penalom za naredno skretanje). Ovo je jeftinije i jednostavnije za implementaciju od praćenja stanja pretrage po paru (čvor, ivica), po cenu da pretraga, u retkim slučajevima, ne mora biti globalno optimalna po ukupnom penalu za skretanje — pojednostavljenje koje je u samom kodu izričito i pošteno dokumentovano, u istom duhu kao i nekalibrisani koeficijenti pomenuti ranije u ovom poglavlju. Evaluacija ovog modula nad stvarnim podacima data je u poglavlju 9.2.

<!-- PAGEBREAK -->

# 7 Backend servis i model podataka

## 7.1 REST API

Backend izlaže REST API organizovan oko pet grupa resursa: autentifikacija, vozila, ture (trips), dispečerski odnosi i chat. Tabela 7.1 sumira najvažnije endpoint-e; kompletan spisak obuhvata i dodatne varijante (npr. `PATCH`, `PUT`) koje ovde nisu posebno izdvojene.

**Tabela 7.1.** Pregled REST API endpoint-a (skraćeno)

| Grupa | Endpoint | Opis |
|---|---|---|
| Autentifikacija | `POST /auth/register`, `POST /auth/login`, `POST /auth/google` | Registracija i prijava (lozinka ili Google) |
| Autentifikacija | `GET /auth/verify-email`, `POST /auth/resend-verification` | Verifikacija email adrese |
| Autentifikacija | `POST /auth/forgot-password`, `GET`/`POST /auth/reset-password` | Obnova zaboravljene lozinke |
| Autentifikacija | `GET /auth/me`, `POST /auth/logout-all` | Podaci o naloгu, odjava sa svih uređaja |
| Vozila | `POST`/`GET`/`PUT`/`DELETE /vehicles` | CRUD nad profilom vozila |
| Vozila | `PATCH /vehicles/{id}/status`, `GET /vehicles/{id}/hours` | Nivo goriva/servis, sati vožnje |
| Rutiranje | `POST /api/v1/routes` | Generisanje, rangiranje i objašnjenje rute (bez čuvanja) |
| Ture | `POST`/`GET /trips`, `GET`/`PUT /trips/{id}` | Kreiranje i pregled tura |
| Ture | `POST /trips/{id}/accept` (i `reject`, `start`, `position`, `reroute`, `complete`) | Prelazi statusne mašine ture |
| Ture | `GET /trips/{id}/events` | Dnevnik događaja tokom vožnje |
| Dispečer | `GET /dispatcher/drivers`, `GET /dispatcher/available-drivers` | Pregled vozača dispečera / dostupnih vozača |
| Dispečer | `POST /dispatcher/requests`, `GET /dispatcher/requests`, `GET /driver/requests`, `POST /driver/requests/{id}/respond` | Zahtev za povezivanje dispečer↔vozač |
| Chat | `GET /chats`, `GET`/`POST /chats/{driverId}/messages` | Pregled i slanje poruka |
| Real-time | `GET /ws/trips/{id}`, `GET /ws/chats/{counterpartId}` | WebSocket veze (poglavlje 7.5) |
| Geokodiranje | `GET /geocode`, `GET /geocode/reverse` | Proxy ka Nominatim servisu |

Endpoint `POST /api/v1/routes` je *stateless* — samo generiše i vraća pregled rute (uz rizik i objašnjenje), bez upisa u bazu — dok `POST /trips` kreira stvarnu turu koja prolazi kroz statusnu mašinu opisanu u nastavku. Ovo razdvajanje omogućava vozaču da pregleda više varijanti rute (npr. sa različitim profilom vozila ili preferencama) bez posledica u bazi, pre nego što se konkretna ruta konvertuje u turu.

**Statusna mašina ture.** Tura prolazi kroz stanja: `offered` (ponuđena vozaču od strane dispečera, čeka odgovor) → `accepted` (vozač prihvatio ponudu) / `rejected` (vozač odbio, terminalno stanje) → `created` (samostalno kreirana od strane vozača, bez posrednog `offered` stanja) → `in_progress` (vozač pokrenuo turu) → `completed`. Backend, preko `HasActiveTrip`/`HasPendingOffer` provera, sprečava da vozač ima paralelno više aktivnih tura ili više neodgovorenih ponuda istovremeno.

## 7.2 Model podataka

Slika 7.1 prikazuje pregled entiteta baze podataka i njihovih veza; potpuna šema, uključujući sve kolone, definisana je preko četiri `goose` migracije opisane u nastavku.

[DIAGRAM:erd]

**Slika 7.1.** Pregled entiteta baze podataka (skraćeno, bez svih kolona)

Šema baze je organizovana u četiri numerisane, hronološke migracije, upravljane alatom `goose`, koji garantuje da se svaka migracija primeni tačno jednom, bez obzira na to koliko puta se sistem pokrene. Tabela 7.2 sumira sadržaj svake migracije.

**Tabela 7.2.** Migracije šeme baze podataka

| Migracija | Sadržaj |
|---|---|
| `00001_initial_schema.sql` | Osnovna šema: `drivers`, `vehicles`, `trips`, `driver_preferences`, `driver_favorite_stops`, `trip_events`, `chat_messages` |
| `00002_dispatcher_roles.sql` | `drivers.role`/`dispatcher_id`, `vehicles.dispatcher_id`, `trips.assigned_by_id`, tabela `dispatcher_requests` |
| `00003_google_and_email_auth.sql` | `drivers.google_sub`/`email`/`email_verified`, tabela `email_verification_tokens` |
| `00004_password_reset_and_logout_all.sql` | `drivers.token_version`, tabela `password_reset_tokens` |

Listing 7.1 prikazuje deo osnovne šeme — tabele `vehicles` i `trips`, sa CHECK ograničenjem koje garantuje da osovinsko opterećenje vozila ne može biti veće od njegove ukupne mase (integritetno ograničenje na nivou baze, ne samo na nivou aplikacije).

```sql
CREATE TABLE IF NOT EXISTS vehicles (
    id SERIAL PRIMARY KEY,
    height_m DOUBLE PRECISION NOT NULL,
    width_m DOUBLE PRECISION NOT NULL,
    length_m DOUBLE PRECISION NOT NULL,
    weight_kg DOUBLE PRECISION NOT NULL,
    axle_load_kg DOUBLE PRECISION NOT NULL,
    hazmat BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (axle_load_kg <= weight_kg)
);

CREATE TABLE IF NOT EXISTS trips (
    id SERIAL PRIMARY KEY,
    vehicle_id INTEGER NOT NULL REFERENCES vehicles(id),
    origin_lat DOUBLE PRECISION NOT NULL,
    origin_lon DOUBLE PRECISION NOT NULL,
    destination_lat DOUBLE PRECISION NOT NULL,
    destination_lon DOUBLE PRECISION NOT NULL,
    distance_km DOUBLE PRECISION NOT NULL,
    duration_min DOUBLE PRECISION NOT NULL,
    risk_score DOUBLE PRECISION NOT NULL,
    shape TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'created',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

**Listing 7.1.** Osnovna šema tabela `vehicles` i `trips` (`00001_initial_schema.sql`)

Geometrija rute (`trips.shape`) čuva se kao enkodovan tekstualni polyline (isti format koji vraća Valhalla), a ne kao PostGIS `geometry` kolona — pragmatičan izbor koji izbegava dekodiranje/enkodiranje geometrije pri svakom čitanju iz baze, po cenu da prostorni upiti nad geometrijom rute (npr. "sve ture koje prolaze kroz ovaj region") nisu mogući direktno u bazi, već bi zahtevali dekodiranje u aplikacionom sloju — razmatrano kao mogući pravac budućeg rada u poglavlju 10.

## 7.3 Uloge vozač/dispečer

Sistem razlikuje dve uloge vozača: **vozač** (`driver`) i **dispečer** (`dispatcher`), zapisane u koloni `drivers.role`. Veza dispečer↔vozač se, namerno, ne uspostavlja pri registraciji, već isključivo preko mehanizma zahteva i odobrenja: dispečer šalje zahtev (`POST /dispatcher/requests`) izabranom vozaču, a vozač ga prihvata ili odbija (`POST /driver/requests/{id}/respond`); prihvatanje postavlja `drivers.dispatcher_id` na identifikator dispečera. Vozač u svakom trenutku može napustiti dispečera (`POST /driver/leave-dispatcher`), čime se ta veza raskida.

Vozilo (`vehicles`) pripada tačno jednom od dva vlasnika — `driver_id` ili `dispatcher_id` — nikad oboma. Ovim se modeluje realan scenario iz transportne prakse: vozač koji radi za dispečera može voziti vozilo koje je u vlasništvu firme (dispečera), ali isto tako može, paralelno, imati i sopstveno vozilo kojim upravlja samostalno — provera koje vozilo vozač trenutno može koristiti (`vehicleAccessible`/`vehicleMutable`) implementirana je na nivou Go handler-a, a ne kao ograničenje baze, jer zavisi i od trenutne dispečer-vozač veze, koja se može promeniti tokom vremena.

Kada dispečer kreira turu za vozača (`assigned_by_id` postavljen na dispečera), ruta se generiše i rangira prema **dispečerovom** profilu preferenci, ne vozačevom — s obzirom na to da je dispečer taj koji donosi odluku o ruti u ime firme, a preference u ovom sistemu modeluju stil odlučivanja o ruti, ne lične navike vozača u vožnji.

## 7.4 Asinhrona obrada — RabbitMQ i modul pauza vozača

Kada vozač pokrene turu, backend objavljuje `trip.started` poruku na `trip.events` topic exchange RabbitMQ-a. Nezavisan worker proces, pokrenut kao goroutine unutar istog backend binarnog fajla (ne kao poseban kontejner), konzumira tu poruku i računa jednostavan predlog pauze: ako je planirano trajanje ture veće od praga od 270 minuta (4.5h — pojednostavljena zamena za stvarno AETR pravilo o obaveznoj pauzi nakon određenog vremena vožnje [AETR], obrazloženo dalje u ovom poglavlju), worker pronalazi tačku na ruti gde bi se vozilo teorijski nalazilo nakon isteka tog vremena (pod pretpostavkom konstantne brzine duž rute), i traži najbliže odmaralište iz skupa OSM node-ova učitanih u poglavlju 5.3 koje je stvarno u koridoru trase. Listing 7.2 prikazuje deo implementacije koja računa predlog.

```go
const restThresholdMin = 270 // 4.5h - stand-in za AETR pravilo

func (w *TripWorker) computeRestStop(ctx context.Context, trip store.Trip) store.RestStopSuggestion {
	if trip.DurationMin <= restThresholdMin {
		return store.RestStopSuggestion{}
	}
	afterMin := float64(restThresholdMin)
	suggestion := store.RestStopSuggestion{AfterMinutes: &afterMin}

	points := valhalla.DecodePolyline6(trip.Shape)
	fraction := restThresholdMin / trip.DurationMin
	at := valhalla.PointAtFraction(points, fraction)

	// ... učitavanje preferenci vozača i omiljenih stanica ...
	stop, _, found := w.RestStops.NearestOnRoute(at.Lat, at.Lon, brand, favorites,
		reststop.DefaultPreferredRadiusM, routePoints, reststop.DefaultRouteCorridorRadiusM, hazmat)
	if !found {
		return suggestion
	}
	suggestion.Lat, suggestion.Lon = &stop.Lat, &stop.Lon
	suggestion.Amenity = &stop.Amenity
	return suggestion
}
```

**Listing 7.2.** Proračun predloga pauze u `TripWorker` (`backend/internal/worker/trip_worker.go`)

Rezultat se upisuje u samu turu (`trips.rest_stop_*` kolone) i u dnevnik događaja (`trip_events`, tip `rest_stop_suggested`), koji mobilna aplikacija prikazuje vozaču kao vremensku liniju (poglavlje 8.1). Za vozila koja prevoze opasan teret, pretraga preferira benzinske stanice nad parkinzima (u granicama tolerancije rastojanja), a rezultat u svim slučajevima favorizuje vozačevu omiljenu ili brend-specifičnu stanicu, ako se takva nalazi u koridoru trase.

**Namerno pojednostavljenje.** Prag od 270 minuta je jednostavna, fiksna zamena za stvarnu evropsku AETR regulativu o radnom vremenu i obaveznim pauzama vozača teretnih vozila [AETR], koja u stvarnosti razlikuje dnevno i nedeljno vreme vožnje, obavezne dnevne i nedeljne odmore, i dozvoljava određena izuzeća. Puna implementacija te regulative izlazi iz obima ovog rada (obrazloženo u poglavlju 10) — cilj modula je da demonstrira **mehanizam** (asinhrono računanje predloga na osnovu proteklog vremena vožnje i geografske pretrage najbliže odgovarajuće lokacije), a ne da zameni stvarnu, zakonski obavezujuću logiku.

**Napomena o evoluciji sistema.** U ranoj fazi razvoja bio je planiran i drugi tok poruka — `trip.eta_updated`, koji bi worker objavljivao nakon računanja ETA, a WebSocket gateway (7.5) konzumirao radi guranja ka klijentu. Taj tok je napušten kada je uvedeno praćenje stvarne GPS pozicije vozila (umesto simulacije), pošto je ETA tada postalo jednostavnije računati direktno u WebSocket gateway-u, iz svake nove GPS pozicije (poglavlje 7.5) — čime je `trip.eta_updated` poruka postala "mrtav kod" (niko je nije konzumirao) i uklonjena je iz sistema. Ovo je uobičajena i zdrava evolucija u razvoju asinhronog sistema: mehanizam koji je u jednom trenutku bio neophodan postaje suvišan kada se promeni pretpostavka na kojoj je zasnovan, i treba ga ukloniti umesto ostaviti kao neaktivan kod.

## 7.5 Real-time komunikacija — WebSocket gateway

Paket `ws` implementira dva WebSocket gateway-a nad istom osnovnom idejom: relej poruka svim zainteresovanim klijentima jedne "teme" (ture ili konverzacije), bez čuvanja stanja konverzacije unutar samog WebSocket sloja.

**Praćenje pozicije uživo.** Za svaku turu koja je u toku, gateway drži strukturu `liveTrip` sa skupom pretplatnika (otvorenih WebSocket konekcija) i poslednjom emitovanom pozicijom. Kada vozačev telefon prijavi novu GPS poziciju (`POST /api/v1/trips/{id}/position`), gateway izračunava preostalo rastojanje do odredišta (haversine formula) i procenjeno vreme dolaska na osnovu prosečnog planiranog tempa ture, i tu poruku emituje svim trenutno povezanim pretplatnicima — vozačevom sopstvenom ekranu i, ako postoji, njegovom dispečeru koji istu turu prati na svojoj live mapi (poglavlje 8.2). Novi pretplatnik koji se poveže (ili ponovo poveže) odmah prima poslednju poznatu poziciju, umesto da čeka narednu GPS prijavu — bez ovoga, dispečer koji zatvori i ponovo otvori live mapu ne bi video ništa do naredne GPS prijave vozila, koja može kasniti minutima ako se vozilo ne kreće (GPS prijave se šalju na pomak od određenog broja metara, ne na fiksni interval).

**Napomena o evoluciji sistema.** Prva verzija ovog gateway-a je, kada još nije stigla nijedna stvarna GPS prijava, simulirala kretanje vozila "hodanjem" duž geometrije rute fiksnim korakom, kao zamenu za stvarni GPS signal tokom razvoja bez pristupa fizičkom vozilu. Kada je testiranje sa stvarnim telefonom postalo dostupno, simulacija je počela da pravi vidljiv "skok" na ekranu u trenutku kada bi stigla prva stvarna GPS prijava (jer se stvarna pozicija po pravilu nije poklapala sa simuliranom) — problem koji je rešen potpunim uklanjanjem simulacije, pošto je do tog trenutka mobilna aplikacija već zahtevala stvaran GPS fix pre nego što dozvoli pokretanje ture (provera blizine polazištu, poglavlje 8.1), čime je simulacija izgubila i svoju originalnu svrhu.

**Chat.** Chat poruke se, za razliku od pozicije, trajno čuvaju u bazi (`chat_messages`) preko REST endpoint-a (`POST /chats/{driverId}/messages`) — REST je "izvor istine" za istoriju razgovora. WebSocket veza (`GET /ws/chats/{counterpartId}`) služi isključivo za isporuku uživo dok je druga strana trenutno povezana: backend objavljuje poslatu poruku na RabbitMQ (`chat.events` exchange), a WS gateway je samo prosleđuje pretplatniku ako je u tom trenutku povezan — ako nije, poruka je već sačuvana u bazi preko REST poziva i biće učitana pri narednom otvaranju razgovora, bez potrebe za posebnim mehanizmom "neisporučenih" poruka.

<!-- PAGEBREAK -->
