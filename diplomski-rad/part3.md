# 8 Mobilna aplikacija

Mobilna aplikacija je jedinstvena Flutter kodna baza koja se, u zavisnosti od uloge prijavljenog korisnika (poglavlje 7.3), grana na dva različita korisnička toka: vozača i dispečera. Grananje se vrši na jednom mestu (`entry_router`), odmah nakon prijave, na osnovu polja `role` vraćenog sa backend-a.

## 8.1 Tok korisnika — vozač

Nakon prijave (`login_screen`, `register_screen` sa izborom uloge vozač/dispečer preko `SegmentedButton`-a, ili `forgot_password_screen` za obnovu lozinke), samostalni vozač ili vozač koji radi za dispečera dolazi na svoj početni ekran. Ako je trenutno pod dispečerom i čeka na ponuđenu turu, prikazuje mu se `offered_trips_screen` sa ponudama koje treba prihvatiti ili odbiti; u suprotnom, vozač sam pokreće planiranje preko `route_request_screen`.

**Planiranje rute.** `route_request_screen` omogućava izbor polazišta (trenutna GPS pozicija ili pretraga adrese preko `address_search_field`, koja poziva backend `GET /geocode` proxy ka Nominatim-u) i odredišta, izbor vozila (preko `vehicle_list_screen`/`vehicle_profile_screen`, gde vozač unosi ili menja visinu, širinu, dužinu, masu, osovinsko opterećenje i prevoz opasnog tereta), i prikaz vraćenih alternativnih ruta na mapi, sortiranih po riziku (poglavlje 6.2), sa mogućim objašnjenjem odstupanja rute (poglavlje 6.3) prikazanim kao tekstualna napomena.

**Pokretanje i praćenje ture.** Pre pokretanja ture, `start_proximity_status` widget provera da je vozač stvarno u blizini (unutar 500m) prijavljenog polazišta preko GPS-a — namerna provera koja sprečava pokretanje ture "iz kancelarije" i garantuje da je prva pozicija koju backend primi zaista stvaran GPS fix, ne proizvoljna vrednost (relevantno za odluku da se ukloni simulacija pozicije, poglavlje 7.5). Tokom vožnje, `active_trip_screen` prikazuje trenutnu poziciju, procenjeno vreme dolaska i alert za predlog pauze (poglavlje 7.4) primljene preko WebSocket veze (`trip_socket.dart`, sa automatskim ponovnim povezivanjem po rastućem intervalu — 1, 2, 5, potom 10 sekundi — u slučaju privremenog prekida konekcije), a `active_trip_banner` ostaje vidljiv i na drugim ekranima aplikacije dok je tura aktivna, kao podsetnik i prečica nazad na `active_trip_screen`. Ako vozač napusti planiranu rutu za više od 300m, backend automatski preračunava novu rutu od trenutne pozicije do istog odredišta (`POST /trips/{id}/reroute`).

**Dnevnik i istorija.** `trip_log_screen` prikazuje dnevnik događaja jedne ture (`trip_events` — polazak, predlog pauze, dolazak) kao vremensku liniju, dok `trip_list_screen` organizuje sve vozačeve ture u tri kategorije (u toku, predstojeće, završene). `truck_status_screen` prikazuje nivo goriva, preostali kilometre do sledećeg servisa i ukupne sate vožnje vozila; `preferences_screen` omogućava podešavanje stila vožnje (klizači za prioritet potrošnje goriva, osetljivosti tereta, korišćenja auto-puta i brzine, opisani u poglavlju 6.2) i omiljene benzinske stanice; `chat_list_screen`/`chat_thread_screen` omogućavaju dopisivanje sa dispečerom u realnom vremenu (poglavlje 7.5).

## 8.2 Tok korisnika — dispečer

Dispečer upravlja flotom vozača i vozila kroz odvojen skup ekrana. `dispatcher_home_screen` je početna tačka; `dispatcher_available_drivers_screen` prikazuje vozače koji trenutno nisu vezani za nijednog dispečera i omogućava slanje zahteva za povezivanje (poglavlje 7.3), dok `dispatcher_requests_screen` prikazuje status poslatih zahteva. `dispatcher_create_trip_screen` omogućava kreiranje i dodelu ture jednom od svojih vozača, birajući između vozačevog sopstvenog vozila ili vozila u vlasništvu dispečera (poglavlje 7.3).

Najkompleksniji ekran dispečerskog toka je `dispatcher_live_map_screen`: prikazuje jednu mapu sa **N paralelnih WebSocket konekcija**, po jednu za svaku aktivnu turu svojih vozača, i ažurira poziciju svakog vozila na mapi čim gateway (poglavlje 7.5) emituje novu poziciju za odgovarajuću turu. `dispatcher_trip_list_screen` prikazuje istoriju svih tura koje je dispečer dodelio.

## 8.3 Arhitektura mobilne aplikacije

Aplikacija je organizovana u standardnu Flutter strukturu: `screens/` (ekrani navedeni u 8.1/8.2), `models/` (podatkovne klase koje odgovaraju backend JSON odgovorima — `driver`, `vehicle_profile`, `trip`, `trip_event`, `route_result`/`route_candidate`, `chat_message`, `dispatcher_request`, `driver_preferences`, `favorite_stop`, `geocode_result`, `position_update`, `rest_stop`, `account_status`, `auth_result`), `services/` i `widgets/` (deljene, ponovo iskoristive komponente).

Sloj servisa uključuje: `api_client.dart` (centralizovan REST klijent koji, pored slanja zahteva, globalno presreće HTTP 401 odgovor i automatski prebacuje korisnika na ekran za prijavu, umesto da svaki ekran posebno rukuje isticanjem sesije), `auth_storage.dart` (lokalno, trajno čuvanje JWT tokena na uređaju), `chat_socket.dart` i `trip_socket.dart` (WebSocket klijenti sa automatskim ponovnim povezivanjem), `location_service.dart` (očitavanje GPS pozicije preko `geolocator` biblioteke), `google_auth.dart` (Google prijava na klijentskoj strani), `polyline.dart` (dekodiranje enkodovane geometrije rute za prikaz na mapi) i `route_observer.dart` (Flutter `RouteAware` mehanizam koji osvežava podatke na ekranu kada se korisnik na njega vrati nakon zatvaranja nekog drugog ekrana — npr. povratak na listu vozila nakon uređivanja profila vozila).

Među deljenim widgetima ističe se `radial_fab_menu.dart` — radijalni, povlačivi (*draggable*) meni pokrenut iz plutajućeg dugmeta (*floating action button*), korišćen kao glavna navigacija umesto klasične donje trake sa karticama (*bottom navigation bar*), kao i `email_verification_banner` (podsetnik da nalog nije verifikovan) i `start_proximity_status` (opisan u 8.1).

<!-- PAGEBREAK -->

# 9 Testiranje i evaluacija

## 9.1 Automatizovano testiranje backend-a

Backend je testiran automatizovanim, tabelarno-vođenim (*table-driven*) Go testovima, standardnim pristupom u Go ekosistemu gde se više varijacija istog scenarija testira kroz jednu test funkciju sa nizom ulaznih/izlaznih parova. Tabela 9.1 prikazuje broj test funkcija po paketu, na dan pisanja rada; svi navedeni testovi prolaze (`go test ./...` vraća `ok` za svaki paket koji sadrži testove).

**Tabela 9.1.** Broj automatizovanih testova po Go paketu

| Paket | Broj testova | Fokus testiranja |
|---|---|---|
| `internal/algorithm` | 15 | Isključivanje nedopustivih ivica/čvorova, poklapanje Dijkstra/A*, penal za skretanje, ponašanje nad realnim OSM podacima |
| `internal/scoring` | 6 | Funkcija bodovanja rute, preferenca korisnika, popust za omiljenu stanicu |
| `internal/explain` | 8 | Detekcija odstupanja rute, identifikacija vezujuće dimenzije, geometrijska divergencija |
| `internal/reststop` | 9 | Pretraga najbližeg odmarališta u koridoru rute, hazmat preferenca, brend/omiljena lokacija |
| `internal/auth` | 5 | Izdavanje/parsiranje JWT tokena, provera lozinke, verzija tokena |
| `internal/ws` | 4 | Relej pozicije pretplatnicima, replay poslednje pozicije novom pretplatniku |
| `internal/httpapi` | 4 | HTTP handleri (integracioni testovi nad REST rutama) |
| `internal/geocode` | 6 | Nominatim proxy, throttling |
| **Ukupno** | **57** | |

Paket `internal/algorithm` koristi dve vrste test podataka: (a) male, ručno konstruisane sintetičke grafove (npr. "dijamant" od četiri čvora sa kontrolisanim ograničenjima), pogodne za preciznu proveru jedne izolovane osobine algoritma (npr. "grana sa `maxheight=3.5` se isključuje za vozilo visine 4.0m"), i (b) stvaran OSM ekstrakt koridora Beograd–Novi Sad (poglavlje 9.2), koji dodatno potvrđuje da implementacija korektno parsira i koristi *stvarne* OSM tagove, ne samo tagove koje je autor sam smislio za potrebe testa.

## 9.2 Evaluacija sopstvenog algoritma nad realnim podacima

Sopstvena Dijkstra/A* implementacija (poglavlje 6.4) je evaluirana nad stvarnim OSM ekstraktom koridora Beograd–Novi Sad (major-roads-only ekstrakt, `testdata/beograd-novisad-corridor.osm`), učitanim istim `LoadOSMXML` mehanizmom koji koristi i produkcioni kod. Ekstrakt se učitava u graf od **38 265 čvorova i 50 705 usmerenih ivica** — dva reda veličine manje od kompletnog nacionalnog Valhalla grafa (898 337 čvorova, poglavlje 5.2), u skladu sa namerom da ovaj modul bude ograničen, transparentan alat za evaluaciju, ne zamena za Valhallu.

**Plauzibilnost rute.** Za par tačaka Beograd (44.80, 20.40) – Novi Sad (45.25, 19.85), Dijkstra pretraga (vozilo visine 4.0m, mase 40 000 kg) pronalazi put od **81,0 km kroz 1899 čvorova**. Za poređenje, Valhalla, nad kompletnim nacionalnim grafom sa punim skupom puteva (ne samo magistralnim), za isti par tačaka vraća rute u rasponu od 84,8 do 92,7 km (u zavisnosti od izabrane alternative), zabeleženo tokom rada na Valhalla integraciji (poglavlje 6.1). Rezultat sopstvenog modula je unutar plauzibilnog opsega — blago kraći, što je i očekivano jer ograničeni ekstrakt sadrži samo magistralne puteve i ne modeluje sve raskrsnice/petlje koje Valhalla uzima u obzir sa kompletnim grafom, a start/cilj pretrage su najbliži čvorovi grafa vazdušnoj liniji do date koordinate, ne tačno iste ulazno-izlazne rampe koje bi Valhalla izabrala.

**Isključivanje na osnovu stvarnog OSM tag-a.** Nad istim ekstraktom pronađena je stvarna deonica sa tagom `maxheight=4.3` u centralnoj zoni Beograda. Za vozilo visine 4.0m, Dijkstra pronalazi put od 4036m preko te deonice; za vozilo visine 4.5m (koje premašuje 4,3m ograničenje), pretraga korektno ne pronalazi nijedan put između istih dveju tačaka — u ovom ograničenom, samo-magistralnom ekstraktu ne postoji alternativna deonica koja bi vozilo moglo koristiti umesto isključene, pa je ishod potpuno odsustvo puta, a ne duži zaobilazni put. Ovaj test potvrđuje da mehanizam isključivanja radi korektno na **stvarnim**, ne samo ručno konstruisanim OSM podacima.

**Poklapanje Dijkstra i A\*.** Za isti par (Beograd, Novi Sad) i profil, Dijkstra i A* implementacija vraćaju identičnu cenu puta, što je i teorijski očekivano za dopustivu heuristiku (poglavlje 2.3) — A* ne pronalazi drugačiji (a kamoli lošiji) put, samo ga pronalazi pregledavajući manji broj čvorova. Vremenska analiza (Tabela 9.2) potvrđuje da je pretraga, nad grafom ove veličine, brza u odnosu na jednokratno parsiranje ulaznog XML fajla — 11,3 MB OSM XML-a.

**Tabela 9.2.** Vreme izvršavanja nad koridorom Beograd–Novi Sad (38 265 čvorova, 50 705 ivica)

| Korak | Vreme |
|---|---|
| Učitavanje i parsiranje OSM XML ekstrakta (`LoadOSMXML`) | ≈ 405 ms |
| Dijkstra pretraga (Beograd → Novi Sad) | ≈ 19,3 ms |
| A* pretraga (Beograd → Novi Sad) | ≈ 17,1 ms |

Rezultat u Tabeli 9.2 potvrđuje očekivanje iz poglavlja 2.3 i 6.4: sama pretraga, jednom kada je graf učitan u memoriju, izvršava se za desetine milisekundi i dominantno je ograničena parsiranjem ulaznih podataka, a ne samim algoritmom pretrage — što je i razlog zbog kojeg je u produkcionom putu ovog sistema izgradnja i održavanje grafa u celosti prepušteno Valhalla-i (poglavlje 5.2, 6.1), koja tu istu vrstu posla radi jednom, unapred, za ceo nacionalni graf, umesto pri svakom zahtevu.

## 9.3 Uticaj prilagođene funkcije rizika — studija slučaja

Doprinos sopstvenog sloja rangiranja (poglavlje 6.2) u odnosu na "sirov" izbor prve Valhalla alternative ilustrovan je konkretnim slučajem otkrivenim tokom testiranja, na ruti u okolini mesta Radalj. Rana verzija formule bodovanja, koja nije sadržala vremenski član (`timeTerm`), birala je kao "najbolju" alternativu rutu koja je bila **47% duža i 30% sporija** od druge ponuđene alternative, isključivo zato što je imala povoljniji odnos magistralne i lokalne deonice puta (`highwayTerm`). Sa stanovišta stvarnog transportnog troška (vreme, gorivo, habanje vozila), taj izbor je bio pogrešan — kraća/brža alternativa je bila objektivno bolji izbor uprkos manje povoljnom `highwayRatio`.

Dodavanjem `timeTerm`-a, koji direktno kažnjava kandidata srazmerno njegovom relativnom kašnjenju u odnosu na najbrži kandidat u istom skupu alternativa (poglavlje 6.2, Listing 6.3), formula je počela korektno da favorizuje bržu/kraću rutu u ovom slučaju, bez uklanjanja postojećih članova formule (broj manevara, udeo magistralne deonice, potrošnja, oštri manevri) koji ostaju relevantni u slučajevima kada razlika u trajanju nije toliko izražena. Ovaj slučaj je dobar primer opšte lekcije o dizajnu heurističkih funkcija cene sa više članova: dodavanje novog signala treba motivisati **konkretnim, uočenim lošim ishodom** postojeće formule, a ne unapred pretpostavljenom kompletnošću — isti princip po kome je i ova formula, i funkcija cene u poglavlju 6.4, iz istog razloga ostala eksplicitno obeležena kao nekalibrisana, prva heuristička procena, a ne konačan, empirijski potvrđen model.

## 9.4 Ograničenja identifikovana testiranjem

Testiranje je otkrilo i granice sistema koje nisu greške u uobičajenom smislu, već posledice svesno odabranog obima rada:

- Sopstveni Dijkstra/A* modul (poglavlje 6.4), primenjen na isti par koordinata za koji je poznat slučaj Novi Banovci (poglavlje 6.3), **ne reprodukuje** taj slučaj — u ograničenom, samo-magistralnom ekstraktu korišćenom za ovaj modul, u okolini Novih Banovaca ne postoji `maxheight` tag na way nivou. Najverovatnije objašnjenje, dokumentovano tokom razvoja, je da je stvarno ograničenje u tom slučaju zavedeno kao tačkasta (node) prepreka, ne kao way tag, i/ili da geometrijski najkraći put u ograničenom grafu jednostavno ne prolazi kroz tačno tu deonicu koju bi Valhalla, sa svojim potpuno drugačijim, kompletnim grafom i sopstvenom heuristikom, izabrala. Ovo je pošteno prijavljen negativan nalaz, ne prikriven — dodatno naglašava razliku u nameni dva sloja rutiranja u ovom sistemu: Valhalla (sa Explain mehanizmom, poglavlje 6.3) je taj koji stvarno rešava slučaj Novi Banovci u produkciji; sopstveni modul (6.4) demonstrira princip mehanizma isključivanja nad drugim, ali i dalje stvarnim, primerom istog tipa OSM ograničenja (maxheight=4,3 u centralnom Beogradu, poglavlje 9.2).
- Penal za skretanje u sopstvenoj pretrazi (6.4) je, kako je i dokumentovano u samom kodu, teorijsko pojednostavljenje koje u retkim slučajevima ne garantuje globalno optimalan put po ukupnom penalu za skretanje.
- Funkcija rizika (6.2) i funkcija cene sopstvene pretrage (6.4) koriste bazne koeficijente koji su prva heuristička procena, ne kalibrisani prema stvarnim podacima o potrošnji goriva, habanju vozila ili statistici saobraćajnih nezgoda — razlika u odnosu na produkcioni sistem bi bila prikupljanje takvih podataka i kalibracija koeficijenata regresijom ili sličnim metodom, što izlazi iz obima ovog rada.

<!-- PAGEBREAK -->

# 10 Zaključak i budući rad

Ovaj rad je pokazao da se prihvatljivo tačan, transparentan i **objašnjiv** sistem za rutiranje teretnih vozila može izgraditi nad potpuno otvorenim geografskim podacima (OpenStreetMap) i otvorenim routing engine-om (Valhalla), uz sopstveni algoritamski doprinos koji tim alatima nedostaje. Implementiran je kompletan sistem koji obuhvata: pripremu geografskih podataka specifičnih za teretni saobraćaj (poglavlje 5), backend servis koji generiše, rangira sopstvenom funkcijom rizika i objašnjava predloženu rutu (poglavlje 6 i 7), sopstvenu, nezavisno testiranu Dijkstra/A* implementaciju nad ograničenim, ali stvarnim podacima (poglavlje 6.4, evaluirana u poglavlju 9.2), distribuiranu komunikaciju preko REST API-ja, RabbitMQ posrednika poruka i WebSocket veza (poglavlje 7), i mobilnu aplikaciju za dve različite korisničke uloge — vozača i dispečera (poglavlje 8). Konkretan slučaj (visinsko ograničenje kod Novih Banovaca, poglavlje 6.3) je korišćen kao stvaran, provaljen dokaz da mehanizam objašnjenja rute radi na pravom, a ne samo hipotetičkom problemu, a slučaj Radalj (poglavlje 9.3) kao konkretan primer u kome je sopstvena funkcija rizika ispravila objektivno pogrešan izbor rute.

## 10.1 Doprinosi rada

Glavni doprinosi rada su:

1. Sopstveni sloj rangiranja alternativnih ruta prema riziku specifičnom za teretni saobraćaj i preferencama vozača, nad Valhalla alternativama koje engine sam ne rangira (poglavlje 6.2).
2. Automatizovan mehanizam koji vozaču objašnjava zašto predložena ruta odstupa od geometrijski najkraćeg puta, sa geometrijskom (ne pozicionom) detekcijom mesta odstupanja, potvrđen na stvarnom slučaju (poglavlje 6.3).
3. Samostalna, testirana Dijkstra/A* implementacija nad ograničenim podgrafom OSM podataka, sa sopstvenom funkcijom cene i isključivanjem ivica/čvorova na osnovu fizičkih ograničenja vozila, evaluirana nad stvarnim, a ne samo sintetičkim podacima (poglavlje 6.4, 9.2).
4. Kompletan, iako po obimu ograničen, distribuiran sistem — REST API, asinhrona obrada porukama, WebSocket komunikacija u realnom vremenu — sa modelom uloga dispečer/vozač i mehanizmom ponude/prihvatanja ture (poglavlje 7).

## 10.2 Ograničenja rada

U skladu sa realno postavljenim obimom (obrazloženim već u poglavlju 1.2), rad svesno ne obuhvata:

- **Punu AETR regulativu** o radnom vremenu i pauzama vozača (poglavlje 7.4) — trenutni prag od 270 minuta je pojednostavljena zamena, ne zakonski obavezujuća logika.
- **Elevacioni (nagibni) rizik rute** — sistem nema pristup podacima o nagibu puta, pa je procena potrošnje goriva (poglavlje 6.2) relativna, ne apsolutna veličina.
- **Offline režim rada mobilne aplikacije** — aplikacija zahteva internet konekciju za svaku operaciju; rad bez signala/na lokalnoj mreži nije podržan.
- **Podršku za više država** — geografski podaci i pripremljen graf pokrivaju isključivo teritoriju Republike Srbije.
- **Produkcioni nivo bezbednosti** — WebSocket `CheckOrigin` je propustljiv (poglavlje 4.3), a RabbitMQ topologija je minimalna (jedan topic exchange, bez retry/DLQ mehanizma), prihvatljivo za sistem sa jednim poznatim klijentom, ne za javni, višezakupčani servis.
- **Kalibraciju koeficijenata funkcije rizika i funkcije cene** prema stvarnim empirijskim podacima (poglavlje 9.4) — koeficijenti su prva heuristička procena.
- **Prostorne upite nad geometrijom rute direktno u bazi** — geometrija se čuva kao tekstualni polyline, ne PostGIS `geometry` kolona (poglavlje 7.2), iako je PostGIS ekstenzija već deo infrastrukture.

## 10.3 Pravci budućeg rada

Na osnovu ograničenja iz 10.2, prirodni pravci budućeg razvoja su: (1) implementacija pune AETR logike, uključujući razliku između dnevnog i nedeljnog vremena vožnje i zakonski propisanih izuzetaka; (2) uvođenje elevacionih podataka (npr. iz SRTM ili sličnog izvora) u funkciju rizika, radi realnije procene potrošnje goriva na deonicama sa izraženim nagibom; (3) offline režim mobilne aplikacije sa lokalnim kešom poslednje poznate rute i odloženim slanjem GPS pozicija po povratku konekcije; (4) proširenje geografskog obuhvata na region ili celu Evropu, što bi zahtevalo i preispitivanje da li ograničeni, u memoriji učitan skup odmarališta (poglavlje 5.3) i dalje skalira, ili treba preći na PostGIS prostorni upit nad već postojećom ekstenzijom; (5) prikupljanje stvarnih podataka (npr. iz simulacije ili pilot upotrebe) radi kalibracije koeficijenata funkcije rizika (6.2) i funkcije cene sopstvenog algoritma (6.4) regresijom ili sličnim metodom, umesto ručno pretpostavljenih vrednosti; (6) bezbednosni hardening WebSocket sloja (stroža `CheckOrigin` provera, autentifikacija na nivou pojedinačne poruke) i proizvodna RabbitMQ topologija (retry, dead-letter queue) ako bi sistem prešao iz demonstracionog u produkcioni kontekst sa nepoznatim, nepoverljivim klijentima.

<!-- PAGEBREAK -->

# Literatura

<!-- LITERATURE_LIST -->

<!-- PAGEBREAK -->

# Spisak skraćenica

| Skraćenica | Značenje |
|---|---|
| AETR | Evropski sporazum o radu posade vozila koja obavljaju međunarodne drumske prevoze (*Accord européen sur les Transports Routiers*) |
| AMQP | *Advanced Message Queuing Protocol* |
| API | *Application Programming Interface* |
| CRUD | *Create, Read, Update, Delete* |
| ETA | *Estimated Time of Arrival* — procenjeno vreme dolaska |
| GPS | *Global Positioning System* |
| HGV | *Heavy Goods Vehicle* — teško teretno vozilo |
| HTTP | *Hypertext Transfer Protocol* |
| JSON | *JavaScript Object Notation* |
| JWT | *JSON Web Token* |
| JWKS | *JSON Web Key Set* |
| MVP | *Minimum Viable Product* |
| OSM | *OpenStreetMap* |
| REST | *Representational State Transfer* |
| SMTP | *Simple Mail Transfer Protocol* |
| SQL | *Structured Query Language* |
| VRP | *Vehicle Routing Problem* |
| WS | WebSocket |

<!-- PAGEBREAK -->

# Spisak slika

- Slika 4.1. Arhitektura sistema — komponente i tok podataka
- Slika 6.1. Tok mehanizma objašnjenja rute (`explain.Explain`)
- Slika 7.1. Pregled entiteta baze podataka (skraćeno, bez svih kolona)

# Spisak tabela

- Tabela 2.1. Poređenje pristupa rutiranju teretnih vozila
- Tabela 4.1. Servisi definisani u `docker-compose.yml`
- Tabela 6.1. Faktor preferencije klase puta (`roadClassMultiplier`)
- Tabela 7.1. Pregled REST API endpoint-a (skraćeno)
- Tabela 7.2. Migracije šeme baze podataka
- Tabela 9.1. Broj automatizovanih testova po Go paketu
- Tabela 9.2. Vreme izvršavanja nad koridorom Beograd–Novi Sad

<!-- PAGEBREAK -->

# Biografija

[IME PREZIME] rođen/a je [DATUM ROĐENJA] godine u [MESTO ROĐENJA]. Osnovne akademske studije na Fakultetu tehničkih nauka Univerziteta u Novom Sadu, studijski program [STUDIJSKI PROGRAM], upisao/la je [GODINA UPISA] godine. Tokom studija se posebno interesovao/la za oblasti [OBLASTI INTERESOVANJA — npr. distribuirani sistemi, mobilne aplikacije, geografski informacioni sistemi]. Ovaj diplomski rad predstavlja rezultat rada na projektu razvoja sistema za rutiranje teretnih vozila, izrađenog pod mentorstvom [TITULA I IME MENTORA].

*(Ovaj odeljak popunjava kandidat/kinja ličnim podacima pre predaje rada.)*
