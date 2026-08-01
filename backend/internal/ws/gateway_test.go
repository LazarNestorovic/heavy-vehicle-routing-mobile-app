package ws

import "testing"

// A subscriber that joins AFTER a position was already broadcast (e.g. a
// dispatcher reopening the live map) must immediately receive that last
// known position, not wait for the next GPS fix - see
// documentations/fixes/2026-08-01-dispatcher-live-map-loses-position-on-reopen.md.
func TestLiveTrip_ReplaysLastPositionToNewSubscriber(t *testing.T) {
	lt := newLiveTrip()
	lt.broadcast(positionUpdate{Lat: 44.8, Lon: 20.4, Status: "in_progress"})

	ch := lt.subscribe()
	select {
	case got := <-ch:
		if got.Lat != 44.8 || got.Lon != 20.4 {
			t.Fatalf("replayed update = %+v, want lat=44.8 lon=20.4", got)
		}
	default:
		t.Fatal("expected the last known position to be replayed immediately, got nothing")
	}
}

// A subscriber joining before anything has been reported must not get a
// spurious empty/zero-value update.
func TestLiveTrip_NoReplayWhenNothingReportedYet(t *testing.T) {
	lt := newLiveTrip()
	ch := lt.subscribe()
	select {
	case got := <-ch:
		t.Fatalf("expected no replay, got %+v", got)
	default:
	}
}

// Broadcasts after subscribing still reach the subscriber as normal.
func TestLiveTrip_BroadcastAfterSubscribeStillDelivered(t *testing.T) {
	lt := newLiveTrip()
	ch := lt.subscribe()
	lt.broadcast(positionUpdate{Lat: 45.0, Lon: 19.0, Status: "in_progress"})

	select {
	case got := <-ch:
		if got.Lat != 45.0 || got.Lon != 19.0 {
			t.Fatalf("update = %+v, want lat=45.0 lon=19.0", got)
		}
	default:
		t.Fatal("expected the broadcast update, got nothing")
	}
}

// Each new subscriber gets its own replay - a second dispatcher opening the
// map later shouldn't consume the first one's replayed message.
func TestLiveTrip_ReplayIsPerSubscriber(t *testing.T) {
	lt := newLiveTrip()
	lt.broadcast(positionUpdate{Lat: 1, Lon: 2, Status: "in_progress"})

	ch1 := lt.subscribe()
	ch2 := lt.subscribe()

	for i, ch := range []chan positionUpdate{ch1, ch2} {
		select {
		case got := <-ch:
			if got.Lat != 1 || got.Lon != 2 {
				t.Fatalf("subscriber %d: update = %+v, want lat=1 lon=2", i, got)
			}
		default:
			t.Fatalf("subscriber %d: expected a replayed update, got nothing", i)
		}
	}
}
