Heavy Vehicle Routing System (OSM-based)
This thesis focuses on developing a service-oriented architecture for the routing of heavy-duty vehicles under load. The system leverages OpenStreetMap (OSM) data and applies specific vehicle constraints such as maximum height, weight, and bridge load capacities.

Key Features:
* Hybrid Offline Mode: Enables downloading and using maps and routes for specific regions without an internet connection.
* Custom Routing Engine: Valhalla with per-request truck costing (height/weight/width/length/axle_load/hazmat), plus a custom Go scoring/algorithm layer for vehicle-specific route risk.
* Mobile Application: Built with Flutter for real-time navigation.

See [SPECIFIKACIJA.md](SPECIFIKACIJA.md) for the full architecture, data model, and work plan.