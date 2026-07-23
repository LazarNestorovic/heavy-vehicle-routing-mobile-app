# Kako pokrenuti Flutter aplikaciju

Kod je napisan bez pristupa Flutter/Dart SDK-u (nije instaliran u razvojnom okruženju gde je pisan — vidi [feature dokument](../features/2026-07-21-flutter-mobile-app.md) za detalje), pa **nije pokrenut niti vizuelno proveren**. Ovo uputstvo je za pokretanje na mašini gde Flutter SDK i emulator/uređaj postoje.

## Preduslovi

- Flutter SDK instaliran (`flutter doctor` bez kritičnih grešaka).
- Android emulator, iOS simulator (samo macOS), ili fizički uređaj.
- Backend stack pokrenut: `docker compose up -d` u root-u projekta (Valhalla, Postgres, RabbitMQ, backend na portu 8080).

## Prvo pokretanje

```bash
cd mobile

# Popuni android/ios/web platform foldere - ne postoje jos, pisani su samo
# lib/ i pubspec.yaml. Moderne verzije Flutter-a (3.x+) podrzavaju ovo u
# direktorijumu koji vec ima lib/pubspec.yaml - dopunice samo ono sto fali.
flutter create --project-name hvr_mobile --org com.example --platforms=android,ios,web .

flutter pub get
flutter run   # izaberi emulator/uredjaj sa liste
```

Ako `flutter create .` odbije da radi u direktorijumu sa postojećim `pubspec.yaml` (zavisi od verzije SDK-a): pokreni `flutter create --project-name hvr_mobile --org com.example --platforms=android,ios /tmp/hvr_scaffold` u praznom folderu, pa iz njega ručno kopiraj `android/` i `ios/` foldere u `mobile/` (ne diraj `lib/` ni `pubspec.yaml` — oni već postoje i ispravni su).

## Podešavanje adrese backend-a

`mobile/lib/config.dart` podrazumevano cilja `http://10.0.2.2:8080` (Android emulator alias za host mašinu). Izmeni po potrebi:

| Cilj | `apiBaseUrl` / `wsBaseUrl` |
|---|---|
| Android emulator | `http://10.0.2.2:8080` / `ws://10.0.2.2:8080` (podrazumevano) |
| iOS simulator | `http://127.0.0.1:8080` / `ws://127.0.0.1:8080` |
| Fizički uređaj | `http://<LAN-IP-masine>:8080` / `ws://<LAN-IP-masine>:8080` — uređaj i backend moraju biti na istoj mreži |

## Šta proveriti pri prvom pokretanju

1. **Ekran profila vozila** — polja su unapred popunjena standardnim profilom (4.0m/2.55m/16.5m/40000kg/11500kg). "Sačuvaj i nastavi" treba da napravi vozilo preko `POST /api/v1/vehicles` i pređe na mapu.
2. **Ekran mape** — dodirni mapu dvaput (polazak, pa odredište) negde u Srbiji (npr. Beograd pa Novi Sad). "Pregled rute" treba da nacrta plavu liniju i prikaže distancu/trajanje/risk score (i `explanation` ako postoji ograničenje — vidi [route-explainability.md](../features/2026-07-21-route-explainability.md)).
3. **"Kreni na put"** — treba da pređe na ekran aktivnog putovanja i odmah otvori WebSocket konekciju; ikonica kamiona treba da počne da se pomera duž rute za ~60 sekundi (simulacija, ne prava GPS pozicija — vidi [websocket-gateway.md](../features/2026-07-21-websocket-gateway.md)).
4. Za dugu rutu (npr. Subotica → Vranje) treba da se pojavi narandžasti alert sa predlogom pauze (stvarna benzinska pumpa/parking iz OSM-a).

## Poznati rizici pri prvom pokretanju (nisu mogli da se provere bez SDK-a)

- Verzije paketa u `pubspec.yaml` su tačne najnovije stabilne verzije sa pub.dev u trenutku pisanja (proverene preko pub.dev API-ja), ali `flutter pub get` može zahtevati manje prilagođavanje zavisno od instalirane Flutter SDK verzije.
- `flutter_map` API (`Marker.child`, `MapController.camera.zoom`) je proveren protiv CHANGELOG-a biblioteke za verzije 7.x→8.x (nema breaking promena koje utiču na ovaj kod), ali nije uživo testiran.
- Android/iOS permisije (internet pristup) — `INTERNET` permisija je podrazumevano uključena u Android template-u koji generiše `flutter create`, ne treba dodatno podešavanje za HTTP/WS pozive ka backend-u.
