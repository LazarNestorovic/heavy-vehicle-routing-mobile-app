# Chat dostupan i dispečeru i vozaču + pretraga kontakata + isticanje dispečera/vozača + username umesto ID-ja

**Datum:** 2026-08-01
**Fajlovi:** `backend/internal/store/driver.go`, `backend/internal/store/chat_message.go`, `backend/internal/httpapi/chat.go`, `mobile/lib/services/api_client.dart`, `mobile/lib/models/chat_message.dart`, `mobile/lib/screens/chat_list_screen.dart`, `mobile/lib/screens/offered_trips_screen.dart`, `mobile/lib/screens/vehicle_list_screen.dart`, `mobile/lib/screens/dispatcher_home_screen.dart`

## Zašto

Chat ("Poruke") je do sada bio dostupan samo vozaču, i to jedino preko `ActiveTripScreen`-a (dakle samo dok je tura u toku). Korisnik je tražio da chat bude dostupan i vozaču i dispečeru kao stalna stavka RadialFabMenu-a (ne samo tokom aktivne ture), da se kontakti za novi razgovor mogu pretraživati po imenu/mejlu, da se u toj listi kontakata istakne (na vrhu) vozačev dispečer odnosno dispečerovi vozači, i naknadno - da se u listi razgovora i unutar samog razgovora prikazuje sagovornikov username umesto generičkog "Vozač #id".

## Chat na RadialFabMenu-u za oba pola

`radial_fab_menu.dart` je ostao nepromenjen (već je generički - lista `RadialFabMenuItem`); dodata je nova stavka "Poruke" (`Icons.chat_bubble_outline`, badge = broj nepročitanih) na sva tri preostala mesta gde postoji "home" RadialFabMenu, po istom obrascu koji je `ActiveTripScreen` već koristio (`_chatUnreadTotal` + `_loadChatUnreadTotal()` pozvano na `initState`/povratak sa `ChatListScreen`-a):

- `offered_trips_screen.dart` (upravljani vozač).
- `vehicle_list_screen.dart` (samostalni vozač bez dispečera - jedini slučaj gde se tamošnji meni uopšte prikazuje).
- `dispatcher_home_screen.dart` (dispečer).

`ActiveTripScreen` je već imao chat, nije diran.

## Backend: pretraga kontakata

`DriverStore.List` (`backend/internal/store/driver.go`) sad prima `query string` (isti obrazac kao već postojeći `ListAvailable`, vidi [available-drivers-search.md](2026-08-01-available-drivers-search.md)) i vraća i `email` kolonu; filtriranje `LOWER(username) LIKE $2 OR LOWER(COALESCE(email,'')) LIKE $2`, case-insensitive substring, prazan query = svi. `GET /api/v1/drivers` (`handleListDrivers` u `chat.go`) prosleđuje `?q=` iz query-stringa i sad koristi zajednički `toDriverResponses` (već postojao u `dispatcher.go` za available/managed liste) umesto ručnog mapiranja bez mejla.

Napomena: `drivers` tabela je zajednička za obe uloge (vidi [dispatcher-driver-roles.md](2026-07-25-dispatcher-driver-roles.md)), pa je "svi ostali registrovani nalozi" već uključivalo i dispečere i vozače i pre ove izmene - `List` samo nije imao pretragu ni mejl u odgovoru.

## Flutter: pretraga + isticanje u `chat_list_screen.dart`

- `services/api_client.dart` — `listDrivers({String? query})`, isti `Uri.replace(queryParameters:)` obrazac kao `listAvailableDrivers`.
- `chat_list_screen.dart` — `_DriverPickerScreen` (bio `StatelessWidget`, samo gola lista) zamenjen sa `_ContactPickerScreen` (`StatefulWidget`): `TextField` za pretragu sa 300ms debounce-om (isti obrazac kao `dispatcher_available_drivers_screen.dart`), `subtitle` prikazuje mejl kad postoji.
  - Isticanje je urađeno **klijentski**, bez novog backend poziva samo za to: `ApiClient.dispatcherId` (vozačev sopstveni dispečer, već se osvežava preko `refreshAccountStatus`) i `ApiClient.listManagedDrivers()` (dispečerovi vozači, već postojeći poziv) su dovoljni da se odredi skup ID-jeva za isticanje. Ti ID-jevi se učitaju JEDNOM u `initState` (`_pinnedIdsFuture`, ne ponovo na svaki debounce), a filtrirana lista kontakata se na svaki reload deli na "istaknuti" + "ostali" i spaja u tom redosledu (`_sorted`) - redosled UNUTAR istaknute grupe ostaje kakav ga je backend vratio (alfabetski po username-u), pošto redosled dispečerovih vozača namerno nije bitan (traženo od korisnika).
  - Istaknuta stavka dobija zvezdicu (`Icons.star`, accent boja) i trailing oznaku "Dispečer" (vozačev slučaj) ili "Vaš vozač" (dispečerov slučaj).

## Username umesto "Vozač #id" u listi razgovora i u samom razgovoru

`GET /api/v1/chats` je prikazivao samo `counterpart_id`, pa je `chat_list_screen.dart` prikazivao "Vozač #5"/"Vozač #15" i u listi razgovora i (dalje, preko `_openThread(c.counterpartId, null)`) kao naslov `ChatThreadScreen`-a. `ChatMessageStore.ListConversations` (`chat_message.go`) sad JOIN-uje `drivers` na `counterpart_id` i vraća `CounterpartUsername`; `chatConversationResponse`/`ChatConversation` (Go i Dart model) dobijaju `counterpart_username`. Lista razgovora sad prikazuje username (i njegovo prvo slovo u avataru umesto `#id`) i taj username se prosleđuje dalje kao `counterpartName` u `ChatThreadScreen`, pa i naslov razgovora sad ispravno prikazuje username. `ChatThreadScreen`-ov `widget.counterpartName ?? 'Vozač #${widget.counterpartId}'` fallback je ostavljen kao odbrambena vrednost (oba trenutna poziva ga uvek prosleđuju), ne bi trebalo da se ikad vidi u praksi.

## Testirano

`go build ./...`, `go vet ./...`, `go test ./...` i `flutter analyze` prolaze čisto. Ručno testiranje toka (chat sa novih home ekrana, pretraga, isticanje) ostaje na korisniku.

## Namerni obim-cut-ovi

- Nema posebne oznake/grupisanja (naslov sekcije) za istaknute kontakte u UI-u - samo su na vrhu liste i vizuelno obeleženi (zvezdica + trailing tekst), bez dodatnog razdvajača, radi jednostavnosti.
- Ako vozač ima VIŠE dispečera (nije moguće u trenutnom modelu - `dispatcher_id` je jedno polje) ili dispečer nema nijednog vozača, lista se prikazuje bez posebnog slučaja - prosto nema šta da se istakne.
- Pretraga ostaje "sadrži" (substring), ne fuzzy.
