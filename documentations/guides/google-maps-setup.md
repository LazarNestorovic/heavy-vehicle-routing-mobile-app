# Podešavanje Google Maps SDK i Google prijave (Google Cloud konzola)

Ovo su koraci koje **korisnik mora sam da uradi** u [Google Cloud Console](https://console.cloud.google.com/) — ne mogu se automatizovati iz koda. Pokrivaju i Maps SDK (prikaz mape, `documentations/features/`) i Google Sign-In (deli isti GCP projekat).

## 1. Google Cloud projekat

Kreiraj novi projekat (ili izaberi postojeći) na [console.cloud.google.com](https://console.cloud.google.com/projectcreate).

## 2. Omogući billing

**Settings → Billing** — poveži nalog za naplatu na projekat. Google ovo zahteva čak i za korišćenje unutar besplatnog tier-a Maps SDK-a (nema naplate dok se ne pređe besplatni limit, ali projekat bez povezanog billing-a ne može da koristi Maps API uopšte). Ovo iznenadi većinu ljudi koji prvi put podešavaju Maps SDK.

## 3. Omogući potrebne API-je

**APIs & Services → Library**, omogući:
- **Maps SDK for Android**

(iOS je namerno preskočen — `mobile/ios/` nema još generisan `Podfile`, projekat se dosad testirao isključivo na fizičkom Android uređaju.)

## 4. Dobavi SHA-1 fingerprint aplikacije

Potreban je za Maps API ključ (Android restrikcija) i za Google Sign-In OAuth klijent:

```bash
cd mobile/android
./gradlew signingReport
```

Traži `SHA1` pod `Variant: debug` (za razvoj — release fingerprint se dodaje odvojeno kad/ako se pravi potpisana verzija za odbranu).

## 5. Maps API ključ

**APIs & Services → Credentials → Create Credentials → API key.**

Odmah posle kreiranja, klikni na ključ i ograniči ga (**ne ostavljaj neograničen** — neko drugi može da ga zloupotrebi i napuni ti billing):
- **Application restrictions → Android apps** → dodaj `com.example.hvr_mobile` + SHA-1 iz koraka 4.
- **API restrictions** → samo "Maps SDK for Android".

Kopiraj ključ — ide u `mobile/android/local.properties` (fajl koji Flutter/Gradle već ignoriše u git-u):

```properties
MAPS_API_KEY=tvoj-kljuc-ovde
```

## 6. OAuth consent screen (potrebno za Google Sign-In, korak 7)

**APIs & Services → OAuth consent screen** — popuni ime aplikacije i support email. Mora biti podešeno pre nego što bilo koji OAuth klijent (korak 7) proradi, čak i u test režimu sa ograničenim brojem test korisnika.

## 7. OAuth 2.0 klijenti za Google Sign-In

**APIs & Services → Credentials → Create Credentials → OAuth client ID**, dva klijenta:

- **Web application** — bez dodatnih polja za sada. Ovo je "server client ID" koji koriste i Flutter (`serverClientId` parametar) i backend (za proveru `aud` u primljenom ID token-u). **Ovo je najčešća greška**: ako se ovaj client ID ne prosledi Flutter strani kao `serverClientId`, ID token koji backend dobije neće imati odgovarajuću publiku i verifikacija će odbiti token.
- **Android** — package `com.example.hvr_mobile` + SHA-1 iz koraka 4. Bez ovoga se Google Sign-In dijalog na Android-u uopšte ne otvara (ali audience ID tokena i dalje dolazi od Web klijenta gore).

Kopiraj Web client ID — ide u `mobile/lib/config.dart` (`googleServerClientId`) i u backend konfiguraciju (`GOOGLE_CLIENT_ID` env promenljiva, `docker-compose.yml`).

## 8. SMTP za potvrdu email adrese (nezavisno od Google Cloud-a)

Izaberi jedno:

- **Gmail + App Password** — Google nalog → Security → 2-Step Verification (mora biti uključeno) → App passwords → generiši lozinku samo za ovu aplikaciju. Šalje na prave inbox-ove, dobro za finalnu odbranu/demo.
- **Mailtrap** (ili sličan sandbox servis) — besplatan nalog, ne šalje na prave adrese (sve završava u test inbox-u koji servis prikazuje), dobro za razvoj bez "spamovanja" sebe. Zameniti pravim SMTP-om pre odbrane ako želiš da se demo email stvarno primi.

Kredencijali idu u `SMTP_HOST`/`SMTP_PORT`/`SMTP_USERNAME`/`SMTP_PASSWORD`/`SMTP_FROM` env promenljive (`docker-compose.yml`).
