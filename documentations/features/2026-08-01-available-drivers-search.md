# Pretraga dostupnih vozača po imenu ili mejlu

**Datum:** 2026-08-01
**Fajlovi:** `backend/internal/store/driver.go`, `backend/internal/httpapi/{dispatcher,chat}.go`, `mobile/lib/models/driver.dart`, `mobile/lib/services/api_client.dart`, `mobile/lib/screens/dispatcher_available_drivers_screen.dart`

## Zašto

Ekran "Dostupni vozači" (gde dispečer bira kome da pošalje zahtev za pridruživanje floti — vidi [dispatcher-driver-roles.md](2026-07-25-dispatcher-driver-roles.md)) do sada je prikazivao SVE neupravljane vozače u jednoj neprekidnoj listi, bez pretrage. Korisnik je tražio da može da filtrira po imenu ili mejlu vozača.

## Backend

`DriverStore.ListAvailable` (`backend/internal/store/driver.go`) sada prima `query string` parametar. Filtriranje je urađeno u samom SQL upitu (`LOWER(username) LIKE $2 OR LOWER(COALESCE(email, '')) LIKE $2`, pattern `%query%`), case-insensitive substring match; prazan `query` daje `%%` što odgovara svima, pa nema potrebe za posebnom granom za "bez filtera".

`GET /api/v1/dispatcher/available-drivers` prihvata opcioni query-string parametar `q` (`handleListAvailableDrivers` u `dispatcher.go` prosleđuje `r.URL.Query().Get("q")` direktno u store).

`driverResponse` (definisan u `chat.go`, deljen sa `handleListManagedDrivers`/`handleListAvailableDrivers` preko `toDriverResponses`) dobija novo `email` polje (`*string`, `omitempty`) — potrebno da se mejl prikaže u UI-u kao razlikovni podatak kad pretraga pogodi po mejlu a ne po imenu.

## Flutter

- `models/driver.dart` — `Driver` dobija opciono `email` polje.
- `services/api_client.dart` — `listAvailableDrivers({String? query})`; query se šalje kao `?q=...` preko `Uri.replace(queryParameters: ...)` (izostavljen ako je prazan/null).
- `dispatcher_available_drivers_screen.dart` — dodato `TextField` za pretragu iznad liste. Otkucani tekst se debounce-uje (300ms `Timer`) pre nego što se ponovo pozove `listAvailableDrivers`, da se ne šalje zahtev na svaki taster; `suffixIcon` (X za brisanje) se ažurira odmah (poseban `setState` van debounce-a) da UI ne deluje "zaglavljeno". `ListTile.subtitle` sada prikazuje mejl vozača kad postoji. Prazna lista razlikuje dva teksta — "nema dostupnih vozača" vs. "nema vozača koji odgovaraju pretrazi" — u zavisnosti da li je pretraga aktivna.

**Testirano:** `go build ./...`, `go vet ./...`, `go test ./internal/httpapi/...` i `flutter analyze` prolaze čisto. Ručno testiranje pretrage kroz UI ostaje na korisniku.

## Namerni obim-cut-ovi

- Nema pretrage na `GET /dispatcher/drivers` (već upravljani vozači) ni na generičkoj chat kontakt listi (`GET /drivers`) — traženo je samo za ekran slanja zahteva za pridruživanje floti.
- Pretraga je "sadrži" (substring), ne fuzzy/tipo-tolerantna.
