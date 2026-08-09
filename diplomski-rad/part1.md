<div class="coverpage">
<div class="letterhead">УНИВЕРЗИТЕТ У НОВОМ САДУ<br/>FAKULTET TEHNIČKIH NAUKA U NOVOM SADU</div>
<div class="coverauthor">[IME PREZIME]</div>
<div class="covertitle">Razvoj aplikacije za rutiranje teretnih vozila prilagođene fizičkim ograničenjima vozila i terena</div>
<div class="coverkind">Diplomski rad<br/>- Osnovne akademske studije -</div>
<div class="coveryear">Novi Sad, 2026.</div>
</div>

<!-- PAGEBREAK -->

## KLJUČNA DOKUMENTACIJSKA INFORMACIJA

| | |
|---|---|
| **Redni broj, RBR:** | |
| **Identifikacioni broj, IBR:** | |
| **Tip dokumentacije, TD:** | Monografska dokumentacija |
| **Tip zapisa, TZ:** | Tekstualni štampani materijal |
| **Vrsta rada, VR:** | Diplomski rad |
| **Autor, AU:** | [IME PREZIME] |
| **Mentor, MN:** | [TITULA I IME MENTORA] |
| **Naslov rada, NR:** | Razvoj aplikacije za rutiranje teretnih vozila prilagođene fizičkim ograničenjima vozila i terena |
| **Jezik publikacije, JP:** | Srpski / latinica |
| **Jezik izvoda, JI:** | Srpski |
| **Zemlja publikovanja, ZP:** | Republika Srbija |
| **Uže geografsko područje, UGP:** | Vojvodina |
| **Godina, GO:** | 2026 |
| **Izdavač, IZ:** | Autorski reprint |
| **Mesto i adresa, MA:** | Novi Sad, Trg Dositeja Obradovića 6 |
| **Fizički opis rada, FO:** (poglavlja/strana/citata/tabela/slika/grafika/priloga) | 10/[BROJ]/17/[BROJ]/[BROJ]/3/0 |
| **Naučna oblast, NO:** | Elektrotehnika i računarstvo |
| **Naučna disciplina, ND:** | Primenjeno softversko inženjerstvo |
| **Predmetna odrednica/Ključne reči, PO:** | Rutiranje teretnih vozila, OpenStreetMap, Valhalla, distribuirani sistemi, mobilna aplikacija, prilagođena funkcija cene rute |
| **UDK** | |
| **Čuva se, ČU:** | U biblioteci Fakulteta tehničkih nauka, Novi Sad |
| **Važna napomena, VN:** | |
| **Izvod, IZ:** | U ovom radu opisan je sistem za rutiranje teretnih vozila koji uzima u obzir fizička ograničenja vozila (visina, širina, dužina, masa, osovinsko opterećenje, prevoz opasnog tereta) i podatke o putnoj mreži preuzete iz OpenStreetMap projekta. Sistem se sastoji od Go backend servisa, Valhalla routing engine-a sa truck costing modelom, PostgreSQL/PostGIS baze podataka, RabbitMQ mehanizma za asinhronu obradu i Flutter mobilne aplikacije za vozače i dispečere. Nad Valhalla alternativama implementiran je sopstveni modul za rangiranje ruta prema riziku i preferencama vozača, kao i mehanizam koji vozaču objašnjava zašto predložena ruta odstupa od geometrijski najkraćeg puta. Za potrebe algoritamske evaluacije, nezavisno od produkcionog puta, implementirana je sopstvena Dijkstra/A* pretraga nad ograničenim podgrafom putne mreže, direktno učitanim iz OSM podataka. Rad opisuje arhitekturu sistema, pripremu geografskih podataka, algoritamski deo, backend servis i mobilnu aplikaciju, a zaključuje se evaluacijom implementiranog rešenja i pregledom mogućih pravaca budućeg razvoja. |
| **Datum prihvatanja teme, DP:** | |
| **Datum odbrane, DO:** | |
| **Članovi komisije, KO:** | Predsednik: |
| | Član: |
| | Član, mentor: [TITULA I IME MENTORA] |

## KEY WORDS DOCUMENTATION

| | |
|---|---|
| **Accession number, ANO:** | |
| **Identification number, INO:** | |
| **Document type, DT:** | Monographic publication |
| **Type of record, TR:** | Textual printed material |
| **Contents code, CC:** | Bachelor thesis |
| **Author, AU:** | [NAME SURNAME] |
| **Mentor, MN:** | [MENTOR TITLE AND NAME] |
| **Title, TI:** | Development of a heavy vehicle routing application adapted to the physical constraints of the vehicle and terrain |
| **Language of text, LT:** | Serbian |
| **Language of abstract, LA:** | Serbian |
| **Country of publication, CP:** | Republic of Serbia |
| **Locality of publication, LP:** | Vojvodina |
| **Publication year, PY:** | 2026 |
| **Publisher, PB:** | Author's reprint |
| **Publication place, PP:** | Novi Sad, Trg Dositeja Obradovića 6 |
| **Physical description, PD:** | 10/[NUM]/17/[NUM]/[NUM]/3/0 |
| **Scientific field, SF:** | Electrical and Computer Engineering |
| **Scientific discipline, SD:** | Applied Software Engineering |
| **Subject/Key words, S/KW:** | Heavy vehicle routing, OpenStreetMap, Valhalla, distributed systems, mobile application, custom route cost function |
| **UC** | |
| **Holding data, HD:** | Library of the Faculty of Technical Sciences, Novi Sad |
| **Note, N:** | |
| **Abstract, AB:** | This thesis describes a heavy vehicle routing system that accounts for the vehicle's physical constraints (height, width, length, weight, axle load, hazardous cargo) using road network data from OpenStreetMap. The system consists of a Go backend service, the Valhalla routing engine with a truck costing model, a PostgreSQL/PostGIS database, RabbitMQ-based asynchronous processing, and a Flutter mobile application for drivers and dispatchers. On top of Valhalla's alternative routes, a custom module ranks candidates by risk/driver preference and explains to the driver why the suggested route deviates from the geometrically shortest path. For algorithmic evaluation, independent of the production path, a custom Dijkstra/A* search over a bounded subgraph loaded directly from OSM data was implemented. The thesis covers system architecture, geographic data preparation, the routing algorithm, the backend service, and the mobile application, and concludes with an evaluation of the implemented solution and directions for future work. |
| **Accepted by the Scientific Board on, ASB:** | |
| **Defended on, DE:** | |
| **Defended Board, DB:** | President: |
| | Member: |
| | Member, Mentor: [MENTOR TITLE AND NAME] |

<!-- PAGEBREAK -->

## ZADATAK ZA IZRADU DIPLOMSKOG (BACHELOR) RADA

**Vrsta studija:** Osnovne akademske studije
**Studijski program:** [STUDIJSKI PROGRAM]
**Rukovodilac studijskog programa:** [IME RUKOVODIOCA]

**Student:** [IME PREZIME]  **Broj indeksa:** [BROJ INDEKSA]
**Oblast:** Elektrotehničko i računarsko inženjerstvo
**Mentor:** [TITULA I IME MENTORA]

**NASLOV DIPLOMSKOG (BACHELOR) RADA:**

RAZVOJ APLIKACIJE ZA RUTIRANJE TERETNIH VOZILA PRILAGOĐENE FIZIČKIM OGRANIČENJIMA VOZILA I TERENA

**TEKST ZADATKA:**

- Proučiti postojeća komercijalna i otvorena rešenja za rutiranje teretnih vozila i principe rada distribuiranih sistema za obradu i prenos podataka u realnom vremenu.
- Proučiti podatkovni model OpenStreetMap projekta u delu koji se odnosi na ograničenja za teretni saobraćaj (visina, masa, širina, osovinsko opterećenje, opasan teret) i pripremiti geografske podatke za teritoriju Republike Srbije.
- Implementirati backend servis koji na osnovu profila vozila i Valhalla routing engine-a generiše rutu, dodatno je rangira sopstvenom funkcijom rizika prilagođenom preferencama vozača i vozilu, te vozaču objašnjava odstupanje predložene rute od geometrijski najkraćeg puta.
- Implementirati, isključivo u svrhu algoritamske evaluacije, sopstvenu pretragu najkraćeg puta (Dijkstra/A*) nad ograničenim podgrafom putne mreže učitanim direktno iz OSM podataka, sa sopstvenom funkcijom cene.
- Implementirati distribuiranu komunikaciju sistema (REST API, asinhrona obrada porukama, WebSocket komunikacija u realnom vremenu) i mobilnu aplikaciju za vozače i dispečere.
- Testirati i evaluirati implementirano rešenje i izvesti najvažnije zaključke.

**Rukovodilac studijskog programa:** [IME RUKOVODIOCA]
**Mentor rada:** [TITULA I IME MENTORA]

<!-- PAGEBREAK -->

## IZJAVA O NEPOSTOJANJU SUKOBA INTERESA

Izjavljujem da nisam u sukobu interesa u odnosu mentor – kandidat i da nisam član porodice (supružnik ili vanbračni partner, roditelj ili usvojitelj, dete ili usvojenik), povezano lice (krvni srodnik mentora/kandidata u pravoj liniji, odnosno u pobočnoj liniji zaključno sa drugim stepenom srodstva, kao ni fizičko lice koje se prema drugim osnovama i okolnostima može opravdano smatrati interesno povezanim sa mentorom ili kandidatom), odnosno da nisam zavisan/na od mentora/kandidata, da ne postoje okolnosti koje bi mogle da utiču na moju nepristrasnost, niti da stičem bilo kakve koristi ili pogodnosti za sebe ili drugo lice bilo pozitivnim ili negativnim ishodom, kao i da nemam privatni interes koji utiče, može da utiče ili izgleda kao da utiče na odnos mentor-kandidat.

U Novom Sadu, dana \_\_\_\_\_\_\_\_\_\_\_\_\_\_

Mentor

\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_

Kandidat

\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_\_

<!-- PAGEBREAK -->

<!-- QUOTE_PAGE -->

<!-- PAGEBREAK -->

## Sadržaj

- 1 Uvod
- 2 Pregled postojećih rešenja i teorijske osnove
  - 2.1 Komercijalna i otvorena rešenja za rutiranje teretnih vozila
  - 2.2 Distribuirani sistemi — osnovni pojmovi
  - 2.3 Grafovi i algoritmi najkraćeg puta
  - 2.4 OpenStreetMap podatkovni model i oznake za teretni saobraćaj
- 3 Korišćene tehnologije
  - 3.1 Go
  - 3.2 Flutter i Dart
  - 3.3 Valhalla
  - 3.4 PostgreSQL i PostGIS
  - 3.5 RabbitMQ
  - 3.6 WebSocket
  - 3.7 Docker i docker-compose
  - 3.8 Ostale biblioteke i servisi
- 4 Arhitektura sistema
  - 4.1 Pregled arhitekture i tok podataka
  - 4.2 Kontejnerizacija i orkestracija servisa
  - 4.3 Sigurnost i autentifikacija
- 5 Priprema geografskih podataka — OSM ekstrakcija za teretna vozila u Srbiji
  - 5.1 Izvor podataka i osmium filter
  - 5.2 Izgradnja Valhalla grafa
  - 5.3 Ekstrakcija podataka o odmaralištima
- 6 Algoritam rutiranja prilagođen vozilu
  - 6.1 Valhalla truck costing
  - 6.2 Prilagođena funkcija rizika (scoring)
  - 6.3 Objašnjenje predložene rute — slučaj Novi Banovci
  - 6.4 Sopstvena implementacija pretrage puta nad ograničenim podgrafom
- 7 Backend servis i model podataka
  - 7.1 REST API
  - 7.2 Model podataka
  - 7.3 Uloge vozač/dispečer
  - 7.4 Asinhrona obrada — RabbitMQ i modul pauza vozača
  - 7.5 Real-time komunikacija — WebSocket gateway
- 8 Mobilna aplikacija
  - 8.1 Tok korisnika — vozač
  - 8.2 Tok korisnika — dispečer
  - 8.3 Arhitektura mobilne aplikacije
- 9 Testiranje i evaluacija
  - 9.1 Automatizovano testiranje backend-a
  - 9.2 Evaluacija sopstvenog algoritma nad realnim podacima
  - 9.3 Uticaj prilagođene funkcije rizika — studija slučaja
  - 9.4 Ograničenja identifikovana testiranjem
- 10 Zaključak i budući rad
- Literatura
- Spisak skraćenica
- Spisak slika
- Spisak tabela
- Biografija

<!-- PAGEBREAK -->

# 1 Uvod

Transport robe teškim teretnim vozilima podleže ograničenjima koja ne postoje kod putničkih automobila: visina vozila mora biti manja od visine svakog nadvožnjaka i tunela na ruti, ukupna masa i osovinsko opterećenje moraju biti u skladu sa nosivošću mostova, širina i dužina vozila ograničavaju kretanje kroz uža naselja i oštre krivine, a vozila koja prevoze opasan teret moraju zaobilaziti određene deonice (tunele, naseljena mesta) po propisu. Standardne aplikacije za navigaciju namenjene putničkim vozilima (Google Maps, Waze i slične) te podatke ne uzimaju u obzir, pa ruta koju predlažu vozaču teretnog vozila može biti fizički neprohodna ili čak zakonski nedozvoljena za dato vozilo. Sa druge strane, postojeća komercijalna rešenja specijalizovana za teretni saobraćaj (npr. PTV navigator, Google Maps za dostavu, TomTom Truck) su zatvorena, plaćena i ne omogućavaju uvid u to *zašto* je određena ruta izabrana, što otežava vozaču da proceni da li je predlog razuman.

Ovaj rad opisuje razvoj sistema za rutiranje teretnih vozila koji rešava taj problem korišćenjem otvorenih geografskih podataka projekta OpenStreetMap (OSM) i otvorenog routing engine-a Valhalla, koji već ima ugrađen model cene rute specifičan za teretna vozila (*truck costing*), sposoban da isključi deonice koje ne zadovoljavaju ograničenja visine, mase, širine ili prevoza opasnog tereta datog vozila. Na tu osnovu je dodat sopstveni sloj koji:

1. rangira alternativne rute koje Valhalla vrati prema riziku i preferencama vozača (stil vožnje, osetljivost tereta, potrošnja goriva, favorizovane benzinske stanice), pošto sam Valhalla to ne radi;
2. vozaču objašnjava **zašto** je predložena ruta odstupila od geometrijski najkraćeg puta, umesto da mu samo prikaže krajnji rezultat;
3. omogućava, kao samostalnu algoritamsku celinu odvojenu od produkcionog puta, sopstvenu implementaciju Dijkstra i A* pretrage najkraćeg puta nad ograničenim podgrafom putne mreže, radi demonstracije i evaluacije principa na kojima Valhalla interno počiva.

Sistem je zamišljen ne samo kao alat za planiranje jedne rute, već kao kompletna, iako po obimu ograničena, distribuirana aplikacija za rad voznog parka: pored planiranja i praćenja pojedinačne ture, uveden je i model dispečera koji upravlja više vozača i vozila, mehanizam ponude i prihvatanja ture između dispečera i vozača, chat u realnom vremenu, praćenje pozicije vozila uživo preko WebSocket veze, dnevnik događaja tokom vožnje i jednostavan mehanizam predloga pauze vozača zasnovan na proteklom vremenu vožnje. Sistem se sastoji od Go backend servisa, PostgreSQL/PostGIS baze podataka, RabbitMQ posrednika poruka, Valhalla routing engine-a i Flutter mobilne aplikacije, a svi servisi su kontejnerizovani i pokreću se preko `docker-compose`.

## 1.1 Motivacija

Motivacija za rad proizlazi iz konkretnog, ponovljivog problema: standardna navigacija ne razlikuje teretno vozilo od putničkog automobila, dok specijalizovana komercijalna rešenja skrivaju logiku odabira rute od korisnika i po pravilu zahtevaju plaćenu licencu i vlasnički skup podataka o putnoj mreži. Cilj ovog rada je da pokaže da se prihvatljivo tačan i **objašnjiv** sistem za rutiranje teretnih vozila može izgraditi nad potpuno otvorenim podacima (OSM) i otvorenim routing engine-om (Valhalla), uz sopstveni algoritamski doprinos koji tim alatima nedostaje: procenu rizika rute specifičnu za teretni saobraćaj i automatsko objašnjenje odstupanja rute od očekivanog, najkraćeg puta.

## 1.2 Ciljevi i obim rada

Ciljevi rada su:

- proučiti postojeća rešenja za rutiranje teretnih vozila i teorijske osnove distribuiranih sistema i algoritama pretrage najkraćeg puta u grafu;
- pripremiti geografske podatke za teritoriju Republike Srbije iz OSM ekstrakta, sa oznakama relevantnim za teretni saobraćaj;
- implementirati backend servis koji generiše, rangira i objašnjava rutu za dato vozilo, koristeći Valhallu kao routing engine;
- implementirati, kao odvojenu algoritamsku celinu, sopstvenu Dijkstra/A* pretragu nad ograničenim podgrafom, radi evaluacije principa rutiranja bez zavisnosti od Valhalla-e;
- implementirati distribuiranu komunikaciju sistema (REST, asinhrona obrada, WebSocket) i mobilnu aplikaciju za vozače i dispečere;
- testirati implementirano rešenje i evaluirati doprinos sopstvene funkcije rizika u odnosu na "sirov" izbor rute.

Rad se svesno ne bavi punom implementacijom evropske AETR regulative o radnom vremenu i pauzama vozača, elevacionim (nagibnim) rizikom rute, offline režimom rada mobilne aplikacije, podrškom za više država niti bezbednosnim hardeningom na produkcionom nivou — ta ograničenja su namerna i obrazložena u poglavlju 10.

## 1.3 Struktura rada

Poglavlje 2 daje pregled postojećih komercijalnih i otvorenih rešenja za rutiranje teretnih vozila i teorijske osnove (distribuirani sistemi, algoritmi pretrage najkraćeg puta, OSM podatkovni model). Poglavlje 3 opisuje korišćene tehnologije. Poglavlje 4 opisuje arhitekturu sistema. Poglavlje 5 opisuje pripremu geografskih podataka. Poglavlje 6 detaljno opisuje algoritamski deo rada — Valhalla truck costing, sopstvenu funkciju rizika, mehanizam objašnjenja rute i sopstvenu Dijkstra/A* implementaciju. Poglavlje 7 opisuje backend servis i model podataka. Poglavlje 8 opisuje mobilnu aplikaciju. Poglavlje 9 opisuje testiranje i evaluaciju. Poglavlje 10 sadrži zaključak i pravce budućeg rada.

<!-- PAGEBREAK -->

# 2 Pregled postojećih rešenja i teorijske osnove

## 2.1 Komercijalna i otvorena rešenja za rutiranje teretnih vozila

Rutiranje teretnih vozila je specijalizovana varijanta opšteg problema pronalaženja puta u grafu, sa dodatnim skupom ograničenja koja zavise od fizičkih karakteristika vozila i tereta. Postojeća rešenja mogu se podeliti u tri grupe.

**Komercijalna rešenja sa vlasničkim podacima.** PTV navigator i PTV xServer (nemački proizvođač PTV Group) su industrijski standard u logistici teretnog saobraćaja u Evropi; koriste vlasnički skup podataka o putnoj mreži sa ručno unetim ograničenjima za kamione (visina mostova, zabrane za HGV — *Heavy Goods Vehicle* — u naseljima i sl.) i naplaćuju se po licenci. TomTom Truck i pojedini moduli Google Maps Platform-a (Routes API sa parametrima vozila) rade na sličnom principu — visok nivo tačnosti podataka, ali zatvoren izvor, zatvorena cena rute i bez mogućnosti da se razumeju ili prilagode kriterijumi po kojima je ruta izabrana.

**Otvoreni routing engine-i.** OSRM (*Open Source Routing Machine*), GraphHopper i Valhalla su tri najpoznatija otvorena routing engine-a koji rade nad OpenStreetMap podacima. OSRM je najbrži za rutiranje putničkih vozila (koristi *contraction hierarchies*), ali njegova podrška za teretna vozila je ograničena i uglavnom se svodi na jednostavne profile bez finog podešavanja po osovinskom opterećenju. GraphHopper ima ugrađen `truck` profil sličan Valhalla-inom i dobru Java/JVM integraciju, ali slabiju podršku za alternativne rute i manje detaljan mehanizam maneuver-a u odgovoru. Valhalla, koji je razvio Mapzen (danas ga održava zajednica okupljena oko Valhalla projekta), izabran je za ovaj rad zbog tri razloga: (1) ima *per-request* `truck` costing model koji prihvata visinu, širinu, dužinu, masu, osovinsko opterećenje i prevoz opasnog tereta kao parametre HTTP zahteva, bez potrebe da se engine iznova kompajlira ili graf iznova gradi za svako vozilo; (2) podržava vraćanje više alternativnih ruta (`alternates`) u jednom zahtevu, što je preduslov za sopstveni sloj rangiranja opisan u poglavlju 6; (3) odgovor sadrži dovoljno detalja o maneuver-ima (tip, naziv ulice, početni indeks u geometriji rute) da se nad njim može izgraditi mehanizam objašnjenja rute (poglavlje 6.3) bez potrebe za dodatnim pozivima ka engine-u.

**Akademski i istraživački radovi.** Problem rutiranja vozila sa ograničenjima (*constrained vehicle routing*) je dobro proučen u okviru šire oblasti *Vehicle Routing Problem* (VRP) [VRP] i njenih varijanti, gde se ograničenja tipično modeluju kao tvrda isključenja grana grafa (vozilo ne može da koristi granu) ili kao dodatni članovi u funkciji cene puta. Ovaj rad se ne bavi klasičnim VRP problemom (raspoređivanjem više vozila na više tura radi minimizacije ukupnog troška flote), već užim problemom rutiranja jednog vozila između dve tačke uz fizička ograničenja i sopstvenu procenu rizika — najbliži akademski okvir je *constrained shortest path* problem, koji je u poglavlju 2.3 i 6.4 rešen sopstvenom implementacijom Dijkstra i A* algoritma sa isključivanjem grana koje vozilo ne može fizički da koristi.

Tabela 2.1 sumira poređenje.

**Tabela 2.1.** Poređenje pristupa rutiranju teretnih vozila

| Rešenje | Izvor podataka | Truck costing | Alternative rute | Objašnjenje rute | Cena |
|---|---|---|---|---|---|
| PTV navigator / xServer | Vlasnički | Da, detaljan | Da | Ne (crna kutija) | Komercijalna licenca |
| Google Routes API (vozilo) | Vlasnički | Delimično | Da | Ne | Po pozivu (plaćeno) |
| OSRM | OSM | Ograničen | Da | Ne | Besplatno (open source) |
| GraphHopper | OSM | Da | Ograničeno | Ne | Besplatno / komercijalna podrška |
| Valhalla (bez izmena) | OSM | Da, detaljan | Da | Ne | Besplatno (open source) |
| **Ovaj rad (Valhalla + sopstveni sloj)** | OSM | Da (Valhalla) | Da (Valhalla) | **Da (sopstveni modul)** | Besplatno |

## 2.2 Distribuirani sistemi — osnovni pojmovi

Sistem opisan u ovom radu je, u smislu klasifikacije koju daju Tanenbaum i van Steen [TANENBAUM], **distribuiran sistem**: skup nezavisnih računarskih procesa (Go backend, PostgreSQL, RabbitMQ, Valhalla, Flutter klijenti) koji korisniku djeluju kao jedinstvena celina, iako fizički i procesno rade odvojeno, najčešće u sopstvenim kontejnerima. U nastavku su ukratko definisani pojmovi neophodni za razumevanje arhitekture opisane u poglavlju 4 i 7.

**Model klijent-server.** Osnovni obrazac komunikacije u sistemu: mobilna aplikacija (klijent) šalje HTTP zahtev Go backend servisu (server), koji obrađuje zahtev i vraća odgovor. Sam backend je, u odnosu na Valhallu i bazu podataka, istovremeno klijent — što je uobičajena situacija u slojevitim sistemima, gde jedan proces može biti server u jednoj interakciji i klijent u drugoj.

**Sinhrona i asinhrona komunikacija.** REST API poziv je *sinhron*: klijent čeka odgovor servera pre nego što nastavi. Za operacije koje ne moraju biti završene odmah da bi klijent nastavio sa radom (npr. proračun predloga pauze vozača nakon što je tura započeta), koristi se *asinhroni* model preko posrednika poruka (RabbitMQ) — proizvođač (backend) objavljuje poruku o događaju i odmah nastavlja dalje, a potrošač (worker proces) je obrađuje kad je slobodan, nezavisno od toga da li je klijent koji je pokrenuo turu još povezan.

**Model izdavač-pretplatnik (publish–subscribe).** RabbitMQ u ovom sistemu radi kao *topic exchange*: proizvođači objavljuju poruke sa određenim *routing key*-em (npr. `trip.started`), a potrošači se pretplaćuju na red vezan za taj routing key. Ovaj model referencijalno razdvaja proizvođača od potrošača — backend koji objavljuje `trip.started` ne mora znati ništa o tome ko (ili da li uopšte neko) sluša taj događaj, što olakšava dodavanje novih potrošača bez izmene postojećeg koda (detaljnije u poglavlju 7.4).

**Real-time komunikacija (WebSocket).** REST model zahteva da klijent inicira svaki zahtev, što ga čini neprikladnim za slanje podataka od servera ka klijentu bez zahteva (npr. pozicija vozila koja se menja svake sekunde). WebSocket protokol (RFC 6455 [WEBSOCKET]) rešava taj problem uspostavljanjem trajne, dvosmerne veze preko jednog TCP soketa nakon početnog HTTP *upgrade* zahteva — server u svakom trenutku može poslati poruku klijentu bez čekanja na njegov zahtev. U ovom sistemu se koristi za praćenje pozicije vozila uživo i za dopisivanje (chat) u realnom vremenu (poglavlje 7.5).

**Transparentnost i pouzdanost.** Od četiri klasična cilja distribuiranog sistema — povezanost, transparentnost, otvorenost i skalabilnost [TANENBAUM] — u ovom radu je najrelevantnija *transparentnost otkaza* (*fault transparency*): sistem je dizajniran da otkaz jedne komponente (npr. privremeni gubitak RabbitMQ konekcije od strane worker procesa) ne obori ostatak sistema, već da se poruka ponovo isporuči (*message redelivery*, videti poglavlje 7.4) kada se komponenta ponovo poveže. S obzirom na obim rada (jedan backend, jedna baza, jedan message broker — bez replikacije ijedne komponente), sistem ne postiže skalabilnost po veličini u produkcionom smislu, što je namerno ograničenje obrazloženo u poglavlju 10.

## 2.3 Grafovi i algoritmi najkraćeg puta

Putna mreža se prirodno modeluje kao usmereni, težinski graf $G = (V, E)$, gde skup čvorova $V$ predstavlja tačke na putu (raskrsnice, prelome geometrije), a skup grana $E$ predstavlja segmente puta između susednih čvorova, sa težinom koja predstavlja cenu prelaska te grane (najčešće rastojanje u metrima, vreme u sekundama ili neka izvedena veličina). Problem rutiranja se svodi na pronalaženje puta $P$ od početnog čvora $s$ do ciljnog čvora $t$ koji minimizuje zbir težina grana na putu, uz eventualna dodatna ograničenja (izuzete grane).

**Dijkstra algoritam.** Dijkstrin algoritam [DIJKSTRA] pronalazi najkraći put od jednog čvora do svih ostalih čvorova u grafu sa nenegativnim težinama grana, koristeći greedy pristup: u svakom koraku bira neposećeni čvor sa najmanjom trenutno poznatom cenom od početka, "fiksira" tu cenu kao konačnu i ažurira cene njegovih suseda (*relaxation*). Algoritam se efikasno implementira uz prioritetni red (u ovom radu binarna gomila, `container/heap` iz Go standardne biblioteke), sa vremenskom složenošću $O((|V| + |E|)\log|V|)$. Kada je poznat samo jedan ciljni čvor (a ne svi čvorovi grafa), pretraga se može prekinuti čim je ciljni čvor skinut sa reda — ta optimizacija je korišćena u implementaciji opisanoj u poglavlju 6.4.

**A\* algoritam.** A* algoritam [ASTAR] je uopštenje Dijkstre koje uvodi *heuristiku* $h(n)$ — procenu preostale cene od čvora $n$ do cilja — i usmerava pretragu prioritetno ka čvorovima za koje je zbir $f(n) = g(n) + h(n)$ (stvarna cena do sada plus procena preostale cene) najmanji. Ako je heuristika *dopustiva* (nikad ne preceni stvarnu preostalu cenu), A* garantovano pronalazi optimalan put, uz manji broj posećenih čvorova od Dijkstre u praksi. U ovom radu je kao heuristika korišćena haversine udaljenost (linija vazdušne udaljenosti po sferi Zemlje) od trenutnog čvora do cilja, podeljena maksimalnom mogućom brzinom kretanja po grani — dopustiva heuristika jer nikad ne potcenjuje stvarni put duž putne mreže, koji je uvek duži ili jednak vazdušnoj liniji.

**Ograničena (constrained) pretraga.** Fizička ograničenja vozila (visina, masa, prevoz opasnog tereta) se u ovom radu modeluju kao *tvrda isključenja* grane iz pretrage: grana koja ne zadovoljava profil vozila jednostavno se ne razmatra kao prelaz, čime pretraga po definiciji nikad ne predloži fizički neprohodnu rutu. Ovaj princip je identičan principu na kome interno počiva Valhalla-in `truck` costing model — razlika je u tome da Valhalla ta isključenja primenjuje nad grafom cele putne mreže, obogaćenim brojnim heuristikama (vreme prolaska, tipovi puta, historijski podaci o saobraćaju), dok sopstvena implementacija u ovom radu (poglavlje 6.4) to radi nad znatno manjim, ograničenim podgrafom, isključivo radi transparentne demonstracije mehanizma.

## 2.4 OpenStreetMap podatkovni model i oznake za teretni saobraćaj

OpenStreetMap (OSM) je projekat slobodne, uređivačke geografske baze podataka sveta, u kojoj se putna mreža, objekti i granice modeluju preko tri osnovna geometrijska primitiva: **node** (tačka, definisana geografskom širinom i dužinom), **way** (uređena lista node-ova, predstavlja liniju ili poligon — npr. jedan segment puta) i **relation** (grupisanje node-ova, way-eva i drugih relation-a — npr. skup segmenata koji zajedno predstavljaju jednu magistralnu rutu ili ograničenje skretanja). Svaki od ova tri primitiva može imati proizvoljan skup **tag**-ova — parova ključ-vrednost koji opisuju njegova svojstva.

Za teretni saobraćaj relevantni su, između ostalih, sledeći tagovi (dokumentovani na OSM Wiki-ju [OSMWIKI]):

- `maxheight`, `maxweight`, `maxwidth` — na way-u, ograničenje visine/mase/širine za tu deonicu (npr. nadvožnjak, tunel);
- `hgv` — da li je way dozvoljen za teška teretna vozila (*Heavy Goods Vehicle*), sa vrednostima poput `yes`, `no`, `destination`;
- `hazmat` — ograničenje za vozila koja prevoze opasan teret;
- `maxaxleload` — ograničenje osovinskog opterećenja, relevantno za mostove;
- `surface` — vrsta podloge (asfalt, kolovoz, tucanik i sl.), indirektno relevantno za rizik (loša podloga + veliko opterećenje = veći rizik);
- `barrier` na node-u (npr. `barrier=height_restrictor`, `barrier=lift_gate`) — tačkasta prepreka koja može imati sopstveni `maxheight` tag, različit od tag-a way-a na kome se nalazi (npr. rampa niske visine na inače visokom putu);
- `amenity=fuel`, `amenity=parking`, `highway=rest_area` — node-ovi koji predstavljaju benzinske stanice, parkinge i odmarališta, relevantni za modul predloga pauze vozača (poglavlje 7.4).

Bitna karakteristika OSM podataka, relevantna za poglavlje 5, je da su ograničenja poput `maxheight` i `barrier` gotovo uvek zavedena na **way** nivou (za deonicu puta), dok su tačkaste prepreke (rampe, stubovi) zavedene na **node** nivou — što znači da alat koji filtrira OSM ekstrakt mora eksplicitno čuvati oba tipa tagova (way i node), jer se u suprotnom gubi tačno ona informacija koja je najrelevantnija za bezbednost teretnog vozila.

<!-- PAGEBREAK -->

# 3 Korišćene tehnologije

## 3.1 Go

Backend servis je implementiran u programskom jeziku Go (Golang) [GO], koji je razvio Google 2009. godine. Go je statički tipiziran, kompajliran jezik sa automatskim upravljanjem memorijom (*garbage collection*) i ugrađenom podrškom za konkurentno programiranje preko *goroutine*-a (lakih, korisnički upravljanih niti) i *channel*-a (tipiziranih kanala za komunikaciju između goroutine-a). Za ovaj rad je Go izabran iz tri razloga: (1) standardna biblioteka već sadrži kompletan HTTP server (`net/http`, uz Go 1.22+ rutiranje po putanji i HTTP metodu preko `http.ServeMux`, korišćeno u `httpapi` paketu), bez potrebe za eksternim veb okvirom; (2) goroutine model konkurentnosti je pogodan za istovremeno opsluživanje mnogo WebSocket konekcija (poglavlje 7.5) i pozadinskog RabbitMQ potrošača (poglavlje 7.4) unutar istog procesa, bez složene infrastrukture za konkurentnost kakva bi bila potrebna u jednonitskim okruženjima; (3) statička tipizacija i kompajliranje u jedan izvršni fajl pogoduju jednostavnom kontejnerizovanju (poglavlje 3.7).

## 3.2 Flutter i Dart

Mobilna aplikacija je implementirana u Flutter okviru [FLUTTER], koji koristi programski jezik Dart (oba razvijena od strane Google-a). Flutter kompajlira aplikaciju u nativni kod za Android i iOS iz jedinstvene baze izvornog koda, a korisnički interfejs se ne oslanja na nativne widgete platforme već na sopstveni *rendering engine* (Skia/Impeller), čime se postiže vizuelna konzistentnost između platformi. Za potrebe ovog rada korišćene su, između ostalih, biblioteke `geolocator` (pristup GPS senzoru uređaja), `flutter_map` odnosno OSM/vector tile prikaz mape, `web_socket_channel` za WebSocket konekcije i `google_sign_in` za prijavu preko Google naloga.

## 3.3 Valhalla

Valhalla [VALHALLA] je otvoreni (open source) routing engine specijalizovan za rad nad OpenStreetMap podacima, originalno razvijen u kompaniji Mapzen. Za razliku od routing engine-a koji graf i cenu puta fiksiraju u trenutku izgradnje (tzv. *preprocessing*, kakav koristi npr. OSRM sa *contraction hierarchies*), Valhalla dozvoljava da se većina parametara cene puta (costing) prosledi **po zahtevu** (*per-request*), uključujući ceo `truck` profil vozila — što znači da nije potrebno iznova graditi graf niti pokretati novu instancu servisa za svako vozilo ili promenu profila. Valhalla graf se organizuje u hijerarhijske "pločice" (*tiles*), izgrađene alatom `valhalla_build_tiles` iz `.osm.pbf` ekstrakta (poglavlje 5.2), a HTTP API (`valhalla_service`) prihvata zahteve na endpoint-u `/route` sa JSON telom koje uključuje lokacije, tip costing modela (`truck`) i njegove opcije. Detaljna upotreba Valhalla-inog truck costing modela u ovom radu opisana je u poglavlju 6.1.

## 3.4 PostgreSQL i PostGIS

PostgreSQL je korišćen kao relaciona baza podataka za skladištenje vozila, vozača, tura i pratećih entiteta (poglavlje 7.2). PostGIS [POSTGIS] je ekstenzija za PostgreSQL koja dodaje geografske tipove podataka i prostorne upite (npr. "sve tačke u krugu od 3km oko date koordinate"). Iako trenutna implementacija čuva geometriju rute kao enkodovan tekstualni *polyline* (a ne PostGIS `geometry` kolonu — videti poglavlje 7.2 i ograničenja u poglavlju 10), PostGIS ekstenzija je uključena u korišćenu Docker sliku (`postgis/postgis:16-3.4-alpine`) kao osnova za prostorne upite koje bi budući razvoj (npr. pretraga "najbliže odmaralište" direktno u bazi umesto u memoriji, poglavlje 7.4) mogao iskoristiti bez promene infrastrukture.

## 3.5 RabbitMQ

RabbitMQ [RABBITMQ] je posrednik poruka (*message broker*) koji implementira AMQP 0-9-1 protokol. U ovom sistemu se koristi *topic exchange* (razmenjivač po temi) nazvan `trip.events`: proizvođač (backend) objavljuje poruku sa *routing key*-em `trip.started` kada vozač pokrene turu, a worker proces (poglavlje 7.4) je preuzima iz reda vezanog za taj routing key, obrađuje je i upisuje rezultat (predlog pauze) u bazu. RabbitMQ je izabran zbog ugrađene podrške za potvrdu obrade poruke (*acknowledgment*) i automatsko ponovno slanje neuspešno obrađene poruke (*requeue*), što sistemu daje otpornost na privremeni pad worker procesa bez gubitka podataka o događaju.

## 3.6 WebSocket

Za real-time komunikaciju (praćenje pozicije vozila uživo i chat) korišćena je biblioteka `gorilla/websocket` na backend strani i `web_socket_channel` na Flutter strani, obe implementacije WebSocket protokola (RFC 6455 [WEBSOCKET]) nad HTTP/1.1 *upgrade* mehanizmom. Detalji implementacije dati su u poglavlju 7.5.

## 3.7 Docker i docker-compose

Svi serverski delovi sistema (Go backend, PostgreSQL, RabbitMQ, Valhalla) su kontejnerizovani Docker slikama [DOCKER] i orkestrisani preko `docker-compose`, alata koji na osnovu deklarativnog YAML fajla pokreće, povezuje u zajedničku mrežu i redosledom health-check zavisnosti startuje više kontejnera jednom komandom (`docker compose up`). Ovo omogućava da se ceo backend deo sistema pokrene na bilo kojoj mašini sa Docker-om instaliranim, bez ručne instalacije PostgreSQL-a, RabbitMQ-a ili Valhalla-e — što je posebno važno za predvidljivost demonstracije rada (poglavlje 9). Detalji konfiguracije dati su u poglavlju 4.2.

## 3.8 Ostale biblioteke i servisi

- **JWT (JSON Web Token, RFC 7519 [JWT])** — format za autentifikacione tokene korišćen preko biblioteke `golang-jwt/jwt`, sa HS256 potpisom; nosi identitet vozača i broj verzije tokena (`token_version`), koji se koristi za "odjavu sa svih uređaja" bez potrebe za posebnom listom opozvanih tokena (poglavlje 4.3).
- **bcrypt** — algoritam za heširanje lozinki sa ugrađenom "solju" i podesivim faktorom rada, korišćen umesto čuvanja lozinki u čistom tekstu ili prostog heša.
- **Google Sign-In / JWKS** — prijava preko Google naloga verifikuje se ručnom proverom Google-ovog ID token-a nad javnim skupom ključeva (JWKS) preko biblioteke `MicahParks/keyfunc`, bez uvođenja celog Firebase SDK-a kao zavisnosti.
- **Nominatim [NOMINATIM]** — javni servis za geokodiranje (pretvaranje adrese u koordinate i obrnuto) projekta OpenStreetMap, korišćen za pretragu adresa u mobilnoj aplikaciji (poglavlje 7.1), sa ograničenjem brzine poziva (throttling) u skladu sa politikom korišćenja servisa.
- **goose [GOOSE]** — alat za upravljanje migracijama šeme baze podataka preko numerisanih SQL fajlova (poglavlje 7.2), koji garantuje da se svaka migracija primeni tačno jednom.
- **osmium-tool** — komandolinijski alat za filtriranje i obradu `.osm`/`.osm.pbf` fajlova, korišćen u pripremi geografskih podataka (poglavlje 5.1).

<!-- PAGEBREAK -->
