# Zatvaranje preostalih obim-cut-ova (runda 2, Blokovi F-M)

**Datum:** 2026-07-26
**Plan:** `/home/lazar/.claude/plans/lovely-cooking-puppy.md` (runda 2 — vidi i [live GPS](2026-07-26-live-gps-tracking.md) i [dopuna bounded A*/Dijkstra](2026-07-21-bounded-astar-dijkstra.md) za ostatak te iste runde)

## Zašto

Korisnik je pročitao svu projektnu dokumentaciju i zatražio da se zatvore preostale sekcije "Šta namerno NIJE urađeno"/"Namerni obim-cut-ovi" koje se ponavljaju kroz skoro svaki raniji feature dokument — stavke ostavljene zbog roka, ne zato što su pogrešne ideje. Ovaj dokument pokriva osam manjih, međusobno nezavisnih blokova (svaki verifikovan pojedinačno pre prelaska na sledeći).

## Blok F — Reset lozinke + odjava sa svih uređaja

Deli mailer infrastrukturu sa email verifikacijom (prošla runda) — reset lozinke je isti obrazac, ne nova infrastruktura.

- `password_reset_tokens` tabela (identična šema kao `email_verification_tokens`), `drivers.token_version INT` — migracija `00004_password_reset_and_logout_all.sql`.
- `POST /api/v1/auth/forgot-password {email}` — UVEK vraća generički `{"status":"sent"}` bez obzira da li email postoji (zaštita od enumeracije naloga). `GET`/`POST /api/v1/auth/reset-password` — ista "browser, ne deep-link" HTML forma kao email verifikacija. `DriverStore.SetPasswordHash` uz promenu lozinke i inkrementuje `token_version` (stara sesija odmah nevažeća).
- `auth.Manager.IssueToken`/`ParseToken` sad nose `token_version` claim; `RequireAuth`/`RequireAuthQuery` upoređuju sa trenutnom bazom vrednošću (jedan dodatan upit po zahtevu). `POST /api/v1/auth/logout-all` inkrementira — svi ranije izdati tokeni trenutno nevažeći.
- Flutter: `ForgotPasswordScreen` (samo email polje) sa `LoginScreen`-a preko "Zaboravljena lozinka?"; `ProfileScreen` dobija "Odjavi sve uređaje" dugme.

**Verifikovano uživo**: reset token iz baze → nova lozinka → stari JWT odmah `401`, stara lozinka `401`, nova lozinka radi; ponovna upotreba istog reset tokena → `400`. `logout-all` → isti token odmah `401`, sledeći login dobija nov `token_version`.

## Blok G — Rest-stop: provera da li je stanica stvarno na putu + hazmat-svesnost

`reststop.Finder.NearestPreferred` je birao geografski najbližu tačku ka JEDNOJ interpolisanoj tački rute, bez provere da li je uopšte na putu vozila. Novi `Finder.NearestOnRoute` dodaje: kandidat mora biti unutar `DefaultRouteCorridorRadiusM` (3km) od BAREM JEDNE tačke uzorkovane duž dekodirane rute (isti sampling princip kao `scoring.go`-ov `preferredStopDiscount`), inače pada nazad na plain `NearestPreferred`. Hazmat vozilo dodatno preferira `amenity=fuel` nad `parking` unutar tolerancije od 5km (`nearestPreferringAmenity`) — **heuristički proxy, ne prava bezbednosna garancija** (OSM ne taguje pouzdano "sme li hazmat vozilo da stane ovde"), eksplicitno dokumentovano u kodu.

**Verifikovano**: 3 nova sintetička testa (corridor filter, fallback kad je koridor prazan, hazmat preferenca) + živa provera (Subotica→Vranje, hazmat=true/false — obe vratile isti stvaran fuel stop na ovoj konkretnoj ruti, pošto je nearest već bio fuel; hazmat grananje samostalno dokazano testom).

## Blok H — Route-explainability: geometrijski presek + keširanje

`firstDivergentStreetName` je upoređivao imena ulica PO POZICIJI U LISTI manevara — radilo je za lokalna odstupanja, ali je za globalno različite rute (jako ograničeno vozilo) prijavljivalo lokaciju blizu POČETKA rute, ne stvarne prepreke (poznato ograničenje iz `2026-07-21-route-explainability.md`).

- Valhalla klijent sad parsira `begin_shape_index` po manevru → `RouteCandidate.ManeuverPoints` (paralelno sa `StreetNames`).
- Nov `divergenceStreetName`: dekodira OBA shape-a, hoda kroz izabranu rutu (`divergencePoint`, uzorkovanje na svaki 5. poen) tražeći prvu tačku čija je najbliža tačka na referentnoj ruti dalja od 200m — GEOMETRIJSKA tačka odstupanja, ne indeks u listi. Zatim `nearestManeuverStreetName` nalazi manevar izabrane rute najbliži toj tački.
- Referentna ruta (uvek ista za dati origin/destination, skup poziv Valhalla-i) sad se kešira 5 minuta (`Explainer.cachedReference`, ključ = zaokruženo origin/destination). **Dokumentovano pojednostavljenje**: keš ključ ne uključuje preference vozača — redak slučaj (dva različita vozača traže isti par koordinata u istom prozoru) dobija referentnu rutu rangiranu PRVOG pozivaoca; česti slučaj (isti vozač ponovo pregleda) profitira.

**Verifikovano**: 8 novih testova (uključujući end-to-end sa ručno enkodiranim polyline6 stringovima, i keš hit-count preko `httptest` Valhalla mock-a). Živa regresija na poznatim slučajevima iz stare dokumentacije: height=4.0 → `explanation: null`; height=4.7 → ISTA ulica (`Партизанске авијације`) kao pre izmene, potvrđujući da geometrijski pristup i dalje pogađa isti stvaran slučaj.

## Blok I — "Zašto" poruka za preferred-stop bonus

`preferredStopDiscount` (scoring.go) je oduzimao 20 poena bez ikakve driver-facing poruke zašto. Nov `scoring.NearestPreferredStopWithinRadius` vraća TAČNO koja sačuvana lokacija je pogodila; `httpapi.preferredStopMessage` je imenuje ("Ruta prolazi blizu vaše omiljene pumpe \"X\".") i postavlja u isto `explanation` polje koje `Explain()` koristi za ograničenja vozila — samo kad `Explain()` sam nije već našao nešto (dva se nikad ne stapaju u jednu poruku). Flutter ne treba izmenu — banner za `explanation` već postoji.

**Verifikovano uživo**: bez omiljene lokacije → `explanation: null`; sa lokacijom tačno na početku rute → `explanation: "Ruta prolazi blizu vaše omiljene pumpe \"Moja omiljena pumpa\"."`.

## Blok J — Alternativne rute na mapi

`candidateResponse` (backend) dobio `Shape` polje (ranije se slao samo za IZABRANU rutu) — `RouteCandidate` (Flutter model) i `Trip` model prošireni da ga parsiraju. `route_request_screen.dart`/`dispatcher_create_trip_screen.dart` crtaju ne-izabrane kandidate kao tanke sive linije (`zIndex: 0`) ispod izabrane rute (`zIndex: 1`).

**Namerno izostavljeno**: `trip_detail_screen.dart` — kandidati se NIKAD ne persistuju u bazi (samo izabrana ruta), pa `GET /trips/{id}` i liste tura uvek vraćaju prazan `candidates` niz; taj ekran nema šta da nacrta bez dodatnog, neopravdanog Valhalla poziva samo za redizajn već-odbačenih alternativa na statičnom pregled-ekranu.

**Verifikovano**: živo `POST /routes` sada vraća `shape` za sve kandidate (potvrđeno preko dužine stringa po kandidatu).

## Blok K — Dispečer: pristup vozačevim ličnim vozilima

Novi `GET /api/v1/dispatcher/drivers/{id}/vehicles` — vraća UNIJU dispečerove flote i vozačevih ličnih vozila (samo za vozača kojim dispečer stvarno upravlja, `403` inače). `vehicleResponse` dobio `is_fleet` polje (izvedeno iz `DispatcherID != nil`) da klijent može grupisati. `dispatcher_create_trip_screen.dart` padajući meni sad prikazuje oba, sa oznakom "Flota" ili imenom vozača.

**Verifikovano uživo**: dispečer + upravljani vozač, fleet vozilo + lično vozilo, `GET .../vehicles` vraća oba sa tačnim `is_fleet`; nepovezan dispečer dobija `403`.

## Blok L — Nominatim pretraga adresa

Novi `backend/internal/geocode` paket — tanak proxy ka Nominatim-u (`GET /search`), sa ugrađenim throttle-om (min. 1.1s između poziva, malo iznad Nominatim-ove politike od ~1 zahtev/sekund) i obaveznim `User-Agent` (`NOMINATIM_USER_AGENT` env var). `GET /api/v1/geocode?q=...&limit=5` (autentifikovano). Flutter: `AddressSearchField` (deljen widget) — pretraživačko polje IZNAD postojećeg tap-na-mapu birača na `preferences_screen.dart` (omiljena lokacija), `route_request_screen.dart`, `dispatcher_create_trip_screen.dart`; tap na rezultat postavlja tačku i pomera kameru, tap-na-mapu i dalje radi.

**Verifikovano**: 4 unit testa (parsiranje, throttle, malformirane koordinate, User-Agent) preko `httptest` mock-a; živ poziv protiv PRAVOG javnog Nominatim-a ("Beograd" → 3 stvarna rezultata sa ćiriličnim imenima).

## Blok M — Vozilo: izmena i brisanje profila

`PUT /api/v1/vehicles/{id}` (isti ownership/validacija obrazac kao `PATCH .../status`), `DELETE /api/v1/vehicles/{id}` — `409` ako vozilo ima ture koje ga referenciraju (Postgres FK violation prevedena u `store.ErrVehicleInUse`, ne proveravana unapred zasebno — izbegava race između provere i brisanja). `VehicleProfileScreen` sad radi dvostruko (kreiranje/izmena preko opcionog `existing` parametra); `VehicleListScreen` dobija meni (Izmeni/Obriši) po vozilu, sa potvrdom pre brisanja.

**Namerno isključeno**: brisanje TURA — istorijski/audit zapis, ostaje append-only.

**Verifikovano uživo**: izmena vozila → nove vrednosti vraćene; vozilo sa turom → `409` na brisanje; vozilo bez ture → `204`, naknadni `GET` → `404`; nevalidna izmena (osovina > težina) → `400`.

## Zajednička verifikacija

Svaki blok pojedinačno: `go build`/`vet`/`test` (backend) i `flutter analyze`/`flutter test` (mobile) čisti PRE prelaska na sledeći blok — ista disciplina kao prošla runda. Finalna kombinovana provera na kraju svih osam blokova: isto, čisto. Docker stack rebuild-ovan i restartovan više puta tokom rada, migracije primenjuju se bez greške (verzija 4).
