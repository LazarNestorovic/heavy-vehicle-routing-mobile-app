# Kompletan vodič za ručno testiranje aplikacije

Ovaj vodič pokriva **sve** funkcionalnosti dodate u projekat (`documentations/features/`, hronološki 2026-07-21 → 2026-08-01), organizovane kao jedan test-prolaz kroz aplikaciju umesto feature-po-feature. Namena: da posle svake veće izmene (ili pred odbranu) možeš da prođeš kroz ceo sistem i budeš siguran da ništa nije pokvareno.

Za rundu 2 (Blokovi D-M: GPS, algoritam, reset lozinke, rest-stop, explainability, itd.) postoji i uži vodič [`test-scope-cut-closures.md`](test-scope-cut-closures.md) — ovaj dokument ga obuhvata i dopunjuje svime što je došlo pre i posle te runde, tako da pokriva ceo projekat na jednom mestu.

Za svaku stavku je naznačeno da li je proverljiva preko **curl-a** (bez telefona) ili zahteva **fizički uređaj** (GPS, gestovi, vizuelni izgled). Curl komande su spremne za copy-paste.

---

## 0. Priprema

### 0.1 Backend stack

```bash
docker compose ps   # svi servisi (postgres, rabbitmq, valhalla, backend) treba da budu "healthy"/"Up"
# ako nije pokrenuto:
docker compose up -d
```

### 0.2 Automatski test paketi (uvek prvi korak — jeftino, hvata regresije odmah)

```bash
cd backend
go build ./... && go vet ./... && go test ./...
```

Očekivano: sve prolazi, uključujući `internal/algorithm` (14 testova — bounded A*/Dijkstra, nema UI za ovo, vidi sekciju 8), `internal/auth`, `internal/scoring`, `internal/reststop`, `internal/ws` (gateway replay testovi), `internal/httpapi`.

```bash
cd ../mobile
flutter analyze   # očekivano: "No issues found!"
flutter test      # očekivano: svi testovi PASS (uključuje radial_fab_menu_test.dart)
```

Ako bilo koji od ova dva koraka padne, nema smisla nastavljati na uređaj — prvo to popraviti.

### 0.3 Pokretanje aplikacije na telefonu

```bash
cd mobile
flutter run
```

`mobile/lib/config.dart` već pokazuje na tvoju LAN IP (`192.168.1.13:8080`) — telefon mora biti na **istoj Wi-Fi mreži** kao računar. Detalji/alternative (emulator, iOS) u [`run-flutter-app.md`](run-flutter-app.md).

### 0.4 Nalozi koji su ti potrebni

Za pun obilazak treba ti **minimum 3 naloga** (mogu svi na istom telefonu, samo se odjavljuješ/prijavljuješ, ili bolje — jedan fizički telefon + jedan emulator da vidiš live sinhronizaciju):

| Nalog | Uloga | Za šta |
|---|---|---|
| A | Vozač (samostalan) | Sekcija 2 |
| B | Vozač (upravljan) | Sekcija 3, povezuje se sa C |
| C | Dispečer | Sekcija 3, upravlja B |

Registruj sve sa validnim email-om (obavezno polje) da testiraš i email-zavisne tokove (sekcija 1.3-1.5). SMTP je već podešen — mejlovi realno stižu na `lazarnestorovic003@gmail.com`, Google Client ID je popunjen u `config.dart`.

---

## 1. Autentifikacija i nalog

*(feature: `driver-preference-scoring` Faza 1, `google-maps-signin-email-verification`, `scope-cut-closures` Blok F)*

### 1.1 Registracija i login (korisničko ime/lozinka)

1. Registruj novi nalog (username + email + lozinka), izaberi ulogu Vozač.
2. Registracija **bez** email-a → treba da odbije ("Obavezno polje" na formi; backend `400` ako se zaobiđe validacija).
3. Login sa tačnim podacima → ulaz u app. Login sa pogrešnom lozinkom → greška NA login ekranu (ne globalna odjava).

### 1.2 Google prijava

1. Na Login ili Register ekranu, "Nastavi sa Google nalogom".
2. Prvi put sa datim Google nalogom → kreira nov nalog (`email_verified: true` odmah, bez potrebe za potvrdom).
3. Ako se Google email poklapa sa postojećim username/lozinka nalogom → nalozi se spajaju (login na isti nalog).
4. Otkazivanje Google dijaloga → vraća na login ekran bez greške.

### 1.3 Potvrda email adrese

1. Posle registracije sa email-om, na home ekranu treba da vidiš traku "Email nije potvrđen(a)".
2. Proveri inbox — mejl sa linkom za potvrdu. Otvori link (browser) → HTML stranica "uspešno potvrđeno".
3. Vrati se u app (traka treba **sama da nestane** čim se app vrati u foreground — ne treba re-login).
4. Probaj isti link ponovo → treba da javi "već iskorišćen"/nevažeći.
5. "Pošalji ponovo" na već potvrđenom nalogu → poruka da je već potvrđen.

curl provera (bez čekanja na UI):
```bash
TOKEN=$(curl -s -X POST http://192.168.1.13:8080/api/v1/auth/login -H "Content-Type: application/json" \
  -d '{"username":"testdriver","password":"test1234"}' | jq -r .token)
curl -s http://192.168.1.13:8080/api/v1/auth/me -H "Authorization: Bearer $TOKEN"
```

### 1.4 Zaboravljena lozinka

1. Login ekran → "Zaboravljena lozinka?" → unesi email.
2. Proveri inbox, otvori link, postavi novu lozinku (web forma).
3. Stara lozinka → odbijena; nova → radi.
4. Isti reset link ponovo → odbijen (400).

### 1.5 Odjava sa svih uređaja

1. Uloguj se na **dva** uređaja/emulatora istim nalogom.
2. Na uređaju 1: Profil → "Odjavi sve uređaje" → vraća na login (lokalno).
3. Na uređaju 2: uradi bilo koju akciju (otvori bilo koji ekran koji zove API) → treba da te **automatski** izbaci na login ekran (401 presretnut globalno, ne samo greška na ekranu).

### 1.6 Profil dostupan bez aktivne ture

Sa bilo kog od tri home ekrana (vozač samostalan/upravljan/dispečer) — ikonica "Profil" u AppBar-u treba da postoji i da radi, **bez obzira** da li postoji aktivna tura.

---

## 2. Samostalan vozač — osnovni tok

*(feature: `go-backend-skeleton`, `vehicle-trip-persistence`, `risk-scoring-layer`, `route-explainability`, `bounded-astar...` nije ovde vidi sek.8, `rest-stop-locations`, `rabbitmq-trip-worker`, `websocket-gateway`, `live-gps-tracking`, `start-proximity-gate`, `off-route-reroute`, `gps-origin-prefill...`, `single-active-trip...`, `trip-list-screen`)*

### 2.1 Vozila — CRUD

1. Prvi login → ekran "Moja vozila" (prazno ili sa postojećim).
2. "+" → forma (unapred popunjena standardnim profilom 4.0m/2.55m/16.5m/40000kg/11500kg) → sačuvaj → vraća na listu, novo vozilo se vidi.
3. Meni (tri tačke) na vozilu → **Izmeni** → forma popunjena postojećim vrednostima → promeni npr. težinu → sačuvaj → lista se osvežava.
4. **Obriši** vozilo bez ijedne ture → nestaje odmah.
5. **Obriši** vozilo koje IMA bar jednu turu (napravi turu s njim prvo) → crvena greška (409), vozilo ostaje.
6. Validacija: `axle_load_kg > weight_kg` na formi → `400`, forma javlja grešku.

### 2.2 Preference vozača

1. Ikonica Preference → 4 slajdera (gorivo/tovar/auto-put/brzina, 1-5), polje za preferirani brend pumpe, lista omiljenih lokacija.
2. Dodaj omiljenu lokaciju: pretraga adrese (novo polje, `Blok L`) ili dodir na mapu → sačuvaj → pojavljuje se u listi.
3. Obriši omiljenu lokaciju → nestaje.

**Provera da preference stvarno utiču na izbor rute** (poznat bug-fix, vredan pokazati na odbrani):
```bash
# Radalj -> Klisa, default preference (3/3/3/3): treba ~142.7km/113min (ne 209km/147min)
curl -s -X POST http://192.168.1.13:8080/api/v1/routes -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{
  "origin": {"lat": 44.7469, "lon": 19.4864},
  "destination": {"lat": 45.2018, "lon": 19.6497},
  "vehicle": {"height_m": 4.0, "width_m": 2.55, "length_m": 16.5, "weight_kg": 40000, "axle_load_kg": 11500, "hazmat": false}
}' | jq '{distance_km, duration_min}'
```
Zatim postavi u Preferencama `highway_priority=5, fuel_priority=1, time_priority=1` i ponovi — treba da izabere DRUGU (dužu, 209km) rutu. Ovo dokazuje da formula stvarno reaguje na preference, ne da je hardkodovana.

### 2.3 Planiranje rute — polja, mapa, pretraga

1. Ekran za rutu: polazna tačka podrazumevano "Trenutna pozicija" (dinamički prati GPS) — klik na polje otvara izbor: pretraga adrese / trenutna pozicija / tačka na mapi.
2. Odredište: zasebno polje, pretraga adrese ILI dodir na mapu.
3. Dodirni mapu za odredište → tačka se postavi, polje se automatski popuni čitljivom adresom (reverse geocoding).
4. Ukucaj mesto u pretragu (npr. "Novi Sad") → lista stvarnih rezultata → tap → mapa centrira, marker postavljen.
5. "Pregled rute" → plava linija + distanca/trajanje/risk score; **1-2 tanje sive linije** ispod nje = alternative koje Valhalla nije izabrala.
6. Plavi "Vaša pozicija" marker prikazan kontinuirano na mapi (i pre pokretanja ture).
7. Vozilo visine **4.7m**, ruta Beograd→Novi Sad → ispod sažetka treba kurzivna napomena tipa *"Ruta skreće kod [ulica] jer visina vozila (4.7m) ne zadovoljava ograničenje..."*. Promeni na 4.0m, ponovi → napomena nestaje.
8. Sačuvana omiljena lokacija TAČNO na ruti → napomena *"Ruta prolazi blizu vaše omiljene pumpe "[ime]"."* (samo ako Blok H napomena nije već zauzela to polje).

### 2.4 Start proximity gate

1. Isplaniraj rutu, NE pomeraj se fizički od trenutne pozicije van 500m od polazišta.
2. "Kreni na put" treba da bude **onemogućeno**, traka ispod pokazuje "Udaljeni ste X km od polazišta...".
3. Postavi polaznu tačku NA svoju stvarnu trenutnu poziciju (ili se fizički odvezi do nje) → traka prelazi u "Na polaznoj tački.", dugme se otključava.
4. Ako GPS dozvola nije data → traka kaže "Nije moguće proveriti lokaciju", dugme trajno zaključano dok se ne reši.

### 2.5 Aktivna tura — GPS uživo

1. Pritisni "Kreni na put" → `ActiveTripScreen`, traži dozvolu za lokaciju → odobri.
2. Marker treba da prati **tvoju stvarnu poziciju** (ne 60s skriptovanu animaciju — simulacija je potpuno uklonjena iz koda).
3. Ako GPS/dozvola nije dostupna → traka objašnjava tačno zašto (isključen GPS / dozvola odbijena / trajno odbijena) sa dugmetom za rešavanje (Podešavanja / Pokušaj ponovo) — nikad tih pad na "ništa se ne dešava" bez objašnjenja.
4. Dugme "Stigao sam" (zastavica u AppBar-u, vidljivo samo dok je GPS aktivan) → na kraju vožnje pritisni → dijalog potvrde, tura postaje `completed`.
5. **Auto-reconnect**: uključi pa isključi avionski režim na 5-10s dok je tura aktivna → mapa se sama oporavi bez potrebe da zatvoriš ekran.
6. **Off-route reroute**: skreni fizički (ili simuliraj lažnom GPS lokacijom preko developer opcija) >300m sa nacrtane rute → traka "Preračunavam rutu..." se pojavi na kratko, pa nova ruta do ISTOG odredišta zameni staru automatski (bez pitanja za potvrdu).
7. Dugačka ruta (Subotica→Vranje, >4.5h) → posle par sekundi (worker radi u pozadini) snackbar "Predlog pauze: [ime stvarne pumpe/parkinga]".

### 2.6 Jedna aktivna tura odjednom

1. Dok je tura u toku, vrati se na "Moja vozila" — treba da vidiš traku "Imate aktivnu turu u toku" sa dugmetom "Nastavi".
2. Dodir na VOZILO koje je na toj turi → ide pravo na `ActiveTripScreen` (ne na ekran za planiranje).
3. Pokušaj da napraviš/pokreneš DRUGU turu dok je prva aktivna → onemogućeno dugme + objašnjenje; backend takođe vraća `409` ako se probije (curl test ispod).
4. Izađi nazad sa aktivne ture (back dugme), pa se vrati na "Moja vozila" **bez restarta app-a** → traka i dalje ispravno prikazuje aktivnu turu (ne zastarelo stanje).

```bash
# regresija: druga POST /trips dok je prva aktivna -> 409
curl -s -X POST http://192.168.1.13:8080/api/v1/trips -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"vehicle_id": 1, "origin": {"lat":44.8,"lon":20.4}, "destination": {"lat":45.25,"lon":19.85}}'
```

### 2.7 "Moje ture" — istorija

1. RadialFabMenu → "Moje ture" → tri taba: Pokrenute / Predstojeće / Završene.
2. Pokrenute (najviše jedna) → dodir vodi na `ActiveTripScreen`.
3. Predstojeće — prazno za samostalnog vozača (očekivano, nema offered/accepted faze).
4. Završene — sortiranje po Datumu/Rastojanju (padajući meni), dodir → `TripLogScreen` (vremenska linija `departed`→`rest_stop_suggested`→`rerouted`→`arrived`).
5. Odbijena tura (ako je ima — samo relevantno za upravljanog vozača, sek. 3) treba da se pojavi u Završenim, ne da nestane.

### 2.8 Truck Status / Cargo / Trip Log (Nocturne ekrani)

1. RadialFabMenu → "Truck Status" → gorivo/servis (izmeni preko dijaloga, `PATCH`), radni sati (`since_last_break_min`/`driving_today_min`).
2. "Cargo" (ako je tura kreirana sa cargo poljima) → read-only prikaz tovara.
3. "Trip Log" → ista vremenska linija kao 2.7.4.

---

## 3. Dispečer + upravljani vozač

*(feature: `dispatcher-driver-roles`, `dispatcher-trip-review-flow` fix, `dispatcher-vehicle-list`, `dispatcher-create-trip-ui-parity`, `available-drivers-search`, `dispatcher-trip-list-screen`, `managed-driver-vehicle-access...` fix, `dispatcher-live-map-*` fixevi)*

### 3.1 Uspostavljanje veze

1. Registruj/uloguj se kao **Dispečer** (nalog C).
2. "Dostupni vozači" → pretraži po imenu/mejlu (polje za pretragu, debounce 300ms) → nađi vozača B.
3. Pošalji zahtev.
4. Uloguj se kao vozač B → "Zahtevi dispečera" → vidi zahtev (sa dispečerovim username-om) → Odobri.
5. B odmah (bez re-logina) prelazi na `OfferedTripsScreen` kao home ekran.
6. Vozač B nestaje iz C-ove liste "Dostupni vozači", pojavljuje se u "Moji vozači".

### 3.2 Napuštanje dispečera

1. Kao vozač B: "Zahtevi dispečera" ekran prikazuje "Trenutni dispečer: [ime]" + dugme "Napusti".
2. Napusti (uz potvrdu) → flotna vozila odmah nestaju iz "Moja vozila", lična ostaju.

### 3.3 Vozila — flota i hibridno vlasništvo

1. Kao dispečer: "Vozila" (u meniju) → lista flotnih vozila (prazna/postojeća), "+" dodaje novo, dodir na vozilo → izmena (ne planiranje rute — dispečer ne vozi).
2. Kao vozač B (upravljan): "Moja vozila" prikazuje UNIJU sopstvenih ličnih I dispečerove flote — flotna vozila imaju drugačiju ikonicu i " · Flota" oznaku.
3. Vozač B pokuša da **izmeni ili obriše** flotno vozilo → `403` (ne sme, samo pregled dozvoljen).
4. Vozač B **PATCH status** (gorivo/servis) na flotnom vozilu (na kom trenutno vozi) → dozvoljeno.

### 3.4 Kreiranje i dodela ture

1. Kao dispečer: "Kreiraj turu" — dva simetrična polja adrese (polazna I odredišna, BEZ GPS moda za polaznu — dispečerov telefon nije relevantan), pretraga adrese, dodir na mapu.
2. Padajući meni za vozilo → prikazuje I flotna vozila ("Flota") I vozačeva lična (imenom vozača) — zajedno.
3. Izaberi vozača B, vozilo, opciono cargo podatke → "Ponudi turu".
4. Tura koristi DISPEČEROV preference profil za scoring (ne vozačev) — proveri da promena dispečerovih preferenci menja izbor rute isto kao u 2.2.

### 3.5 Pregled i odluka vozača (accepted/rejected)

1. Kao vozač B: "Ponuđene ture" → tura se pojavljuje.
2. Dodir → `TripDetailScreen` — statična mapa cele rute, distanca/trajanje/risk score, tovar (ako postoji), vozilo. Dugmad [Odbij] [Prihvati].
3. **Odbij** jednu turu → status `rejected`, nestaje iz ponuđenih, dispečer vidi status "Odbijena" u svojoj listi.
4. **Prihvati** drugu → status `accepted`, dugme postaje [Kreni]. Proveri start proximity gate ovde takođe (traka se pojavljuje tek u "accepted" stanju, koristi prvu tačku rute kao polazište).
5. "Kreni" → `startTrip` → `ActiveTripScreen` (isto ponašanje kao 2.5).

### 3.6 Dispečerov uvid — "Sve ture"

1. Kao dispečer: "Sve ture" → tri taba: Pokrenute / Predstojeće / Završene, stavke prikazuju ime vozača.
2. Pokrenute (može biti VIŠE odjednom, po jedna po vozaču) → dodir vodi na `DispatcherLiveMapScreen` (ne `ActiveTripScreen`).
3. Završene: sadrži i `completed` I `rejected` ture — signal dispečeru da turu treba ponuditi drugom vozaču.

### 3.7 Dispečerova live mapa

1. Dok vozač B vozi (2.5), otvori kao dispečer "Vozila uživo" → marker vozača B vidljiv, prati stvarnu GPS poziciju (isti WS gateway).
2. Tura koja je TEK pokrenuta (status još `created`, worker nije obradio) → mora se ODMAH videti na mapi, ne tek kad pređe u `in_progress`.
3. Izađi sa ekrana live mape i vrati se dok je vozač i dalje aktivan, POGOTOVO ako vozač trenutno MIRUJE (nema novih GPS pingova) → marker treba da se odmah pojavi na poslednjoj poznatoj poziciji, ne da ostane prazan dok se vozilo ponovo ne pomeri.

---

## 4. Chat

*(feature: `nocturne-redesign` Blok A/B3, `chat-both-roles-contact-search-pinning`)*

1. Sa **sva tri** home ekrana (samostalan vozač, upravljani vozač, dispečer) — RadialFabMenu ima stalnu stavku "Poruke" (badge = broj nepročitanih), ne samo tokom aktivne ture.
2. Novi razgovor → pretraga kontakata po imenu/mejlu (debounce).
3. U listi kontakata, vozačev dispečer (ili dispečerovi vozači) su **na vrhu**, sa zvezdicom i oznakom "Dispečer"/"Vaš vozač".
4. Pošalji poruku A→B, otvori kao B (drugi telefon/nalog) → poruka stiže **uživo** preko WS-a bez refresh-a.
5. Lista razgovora i naslov razgovora prikazuju **username** sagovornika (ne "Vozač #id").
6. Broj nepročitanih se smanjuje na 0 čim se razgovor otvori.
7. Pošalji poruku i proveri da se NE duplira na strani pošiljaoca (poznat bug, ispravljen).

---

## 5. Google Maps prikaz

*(feature: `google-maps-signin-email-verification` Blok A)*

1. Svih 6 ekrana sa mapom (planiranje rute, dispečer kreiranje, aktivna tura, dispečerova live mapa, detalji ture, preference) treba da prikazuju **Google Maps** podlogu (ne OSM/flutter_map tajlove).
2. Markeri: zeleno=polazak, crveno=odredište, ljubičasto=kamion (vozilo), narandžasto=pumpa/odmorište, žuto=omiljena lokacija.
3. Ako mapa ne prikazuje ništa (siva/prazna) → proveri `MAPS_API_KEY` u `android/local.properties` i restrikcije u GCP konzoli ([`google-maps-setup.md`](google-maps-setup.md)).

---

## 6. Regresije preko curl-a (brzo, bez telefona)

Koristan sanity-check paket kad menjaš backend kod — svaka linija treba da vrati tačno naznačen status:

```bash
BASE=http://192.168.1.13:8080

# health
curl -s $BASE/healthz

# 422 - van pokrivenosti grafa
curl -s -o /dev/null -w "%{http_code}\n" -X POST $BASE/api/v1/routes -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{
  "origin": {"lat": 48.85, "lon": 2.35}, "destination": {"lat": 48.86, "lon": 2.36},
  "vehicle": {"height_m": 4.0, "width_m": 2.55, "length_m": 16.5, "weight_kg": 40000, "axle_load_kg": 11500, "hazmat": false}}'
# ocekivano: 422

# 400 - nevalidan JSON / axle > weight
curl -s -o /dev/null -w "%{http_code}\n" -X POST $BASE/api/v1/vehicles -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"height_m":4.0,"width_m":2.55,"length_m":16.5,"weight_kg":10000,"axle_load_kg":20000,"hazmat":false}'
# ocekivano: 400

# 404 - nepostojece vozilo
curl -s -o /dev/null -w "%{http_code}\n" $BASE/api/v1/vehicles/999999 -H "Authorization: Bearer $TOKEN"
# ocekivano: 404

# ownership - drugi vozac ne moze da vidi tudje vozilo/turu (zameni $TOKEN_B tokenom drugog naloga)
curl -s -o /dev/null -w "%{http_code}\n" $BASE/api/v1/vehicles/1 -H "Authorization: Bearer $TOKEN_B"
# ocekivano: 403 ako vozilo #1 ne pripada nalogu B

# bez tokena uopste
curl -s -o /dev/null -w "%{http_code}\n" $BASE/api/v1/vehicles
# ocekivano: 401
```

---

## 7. RabbitMQ / worker (opciono, uvid u infrastrukturu)

Management UI: `http://192.168.1.13:15672` (`hvr`/`hvr_dev_password`) — koristan za vizuelnu potvrdu da `trip.events`/`chat.events` exchange-evi postoje i primaju poruke dok testiraš sekcije 2/3/4 iznad. Nema šta posebno da se "testira" ovde odvojeno — ako sekcije 2.5 (rest-stop predlog) i 4 (chat uživo) rade, RabbitMQ tok radi.

---

## 8. Bounded A*/Dijkstra — algoritamski modul (BEZ UI-ja)

*(feature: `bounded-astar-dijkstra` + dopuna node-barijere/cost funkcija)*

Ovo je eksperimentalni modul za rad (poglavlje o algoritmu), **nije** deo `/routes` API-ja koji app koristi — nema šta da se klikne u aplikaciji.

```bash
cd backend
go test ./internal/algorithm/... -v
```

Očekivano: 14/14 testova PASS, uključujući `TestRealCorridor_HeightRestrictionExcludesRealTaggedEdge`, `TestDijkstra_NodeBarrierExcludesTallVehicle`, `TestRealCorridor_AStarMatchesDijkstra`.

---

## 9. Poznata ograničenja (ne prijavljuj kao bug)

- Simulacija kretanja je **potpuno uklonjena** — bez GPS dozvole, mapa aktivne ture jednostavno ne prikazuje ništa (očekivano, ne bug).
- Dispečer nema pretragu na "Moji vozači" ni na chat kontakt listi (samo na "Dostupni vozači").
- `TripDetailScreen` ne crta alternativne rute (samo `RouteRequestScreen`/`DispatcherCreateTripScreen`) — kandidati se ne perzistuju u bazi.
- Google prijava zahteva da telefon ima Google Play Services (ne radi na svim emulator image-ima).
- iOS nije testiran nijednom u ovom projektu — samo Android fizički uređaj.
- Reroute/off-route prag (300m), start-proximity prag (500m), preferred-stop radijus (3km/15km) su heuristike, nisu kalibrisane — ne očekuj "tačnu" granicu na terenu.
- Email/reset-lozinke linkovi otvaraju **browser HTML stranicu**, ne vraćaju se u app (nema deep-linking-a) — očekivano, ne bug.

---

## Ako nešto ne radi

Za svaki feature postoji odgovarajući fajl u `documentations/features/YYYY-MM-DD-*.md` sa tačnim opisom šta je implementirano, mehanizmom i kako je već potvrđeno preko curl-a — uporedi da vidiš da li je problem u UI-ju (očekivano, "ostaje na korisniku") ili je nešto stvarno pokvareno u pozadini (regresija, vredna `fixes/` zapisa).
