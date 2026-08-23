Heavy Vehicle Routing System (OSM-based)
This thesis focuses on developing a service-oriented architecture for the routing of heavy-duty vehicles under load. The system leverages OpenStreetMap (OSM) data and applies specific vehicle constraints such as maximum height, weight, and bridge load capacities.

Key Features:
* Hybrid Offline Mode: Enables downloading and using maps and routes for specific regions without an internet connection.
* Custom Routing Engine: Valhalla with per-request truck costing (height/weight/width/length/axle_load/hazmat), plus a custom Go scoring/algorithm layer for vehicle-specific route risk.
* Mobile Application: Built with Flutter for real-time navigation.

See [SPECIFIKACIJA.md](SPECIFIKACIJA.md) for the full architecture, data model, and work plan.

## Running the app

Prerequisites: Docker + Docker Compose, Flutter SDK (`flutter doctor`), and an Android emulator or a physical device.

1. **Backend stack** — from the repository root:

   ```bash
   cp .env.example .env   # first run only; defaults work for a local demo
   docker compose up -d   # Postgres, RabbitMQ, Valhalla, Go backend on :8080
   ```

   Routing tiles are already built in `valhalla/tiles/`. If they are missing or the
   OSM extract changed, rebuild them once with
   `docker compose --profile build up valhalla-build` (5-15 min).

2. **Mobile app** — from `mobile/`:

   ```bash
   cd mobile
   flutter pub get
   flutter run   # pick the emulator/device from the list
   ```

   The app targets `http://10.0.2.2:8080` (the Android emulator's alias for the host
   machine) by default. For an iOS simulator or a physical device, change
   `apiBaseUrl` / `wsBaseUrl` in [mobile/lib/config.dart](mobile/lib/config.dart).

Stop everything with `docker compose down` (add `-v` to also drop the database volume).

More detail: [documentations/guides/run-flutter-app.md](documentations/guides/run-flutter-app.md)
and [documentations/guides/rebuild-valhalla-graph.md](documentations/guides/rebuild-valhalla-graph.md).
