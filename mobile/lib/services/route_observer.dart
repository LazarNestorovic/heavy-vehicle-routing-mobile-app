import 'package:flutter/material.dart';

/// App-wide RouteObserver, registered on MaterialApp in main.dart. Lets a
/// screen mix in RouteAware and override didPopNext() to refresh itself when
/// a route pushed on top of it is popped and it becomes visible again -
/// Navigator does NOT call initState() again for an existing screen that's
/// merely uncovered, so without this, screens kept showing whatever trip
/// state they had when they were first pushed (e.g. "no active trip") even
/// after one was started/completed while a screen on top of them was open -
/// only a full app restart picked up the change. See e.g.
/// screens/route_request_screen.dart, screens/vehicle_list_screen.dart,
/// widgets/active_trip_banner.dart.
final routeObserver = RouteObserver<PageRoute<dynamic>>();
