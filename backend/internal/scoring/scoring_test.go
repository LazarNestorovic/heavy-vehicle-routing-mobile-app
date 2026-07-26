package scoring

import (
	"testing"

	"heavy-vehicle-routing/backend/internal/valhalla"
)

// radaljCandidates reproduces the real Radalj->Klisa case (documentations/features/
// 2026-07-21-driver-preference-scoring.md): a direct route and a 47%-longer,
// 30%-slower one that only wins on highway ratio. Real numbers from that
// investigation, ManeuverCount/SharpManeuverCount are representative not exact.
func radaljCandidates() []valhalla.RouteCandidate {
	return []valhalla.RouteCandidate{
		{ // the direct, sensible route (Valhalla's own top pick)
			DistanceKm: 142.7, DurationMin: 113.1, ManeuverCount: 27, SharpManeuverCount: 10, HighwayRatio: 0.21,
		},
		{ // the longer, more-highway route the old fixed-weight formula picked
			DistanceKm: 209.4, DurationMin: 146.9, ManeuverCount: 28, SharpManeuverCount: 2, HighwayRatio: 0.64,
		},
	}
}

func TestRank_DefaultPreferences_PicksDirectRouteOverRadaljDetour(t *testing.T) {
	ranked := Rank(radaljCandidates(), Preferences{FuelPriority: 3, CargoPriority: 3, HighwayPriority: 3, TimePriority: 3}, 40000, nil)

	best := ranked[0]
	if best.DistanceKm != 142.7 {
		t.Errorf("expected the direct 142.7km route to win with default (neutral) preferences, got %.1fkm (risk=%.1f)",
			best.DistanceKm, best.RiskScore)
	}
}

// TestRank_HighwayLover_CanStillPreferTheDetour proves the formula is genuinely
// tunable, not just patched to always avoid detours: a driver who explicitly
// doesn't care about time or the extra fuel/distance, and heavily prioritizes
// highway, can still end up with the longer route - it's a preference, not a
// hidden hard rule. (fuel_priority is lowered too, not just time/highway - a
// driver who's fine with 47% more distance is, consistently, also fine with
// the fuel that costs; leaving fuel_priority neutral would let the fuel term's
// distance sensitivity override the highway preference on its own.)
func TestRank_HighwayLover_CanStillPreferTheDetour(t *testing.T) {
	ranked := Rank(radaljCandidates(), Preferences{FuelPriority: 1, CargoPriority: 3, HighwayPriority: 5, TimePriority: 1}, 40000, nil)

	best := ranked[0]
	if best.DistanceKm != 209.4 {
		t.Errorf("expected the highway-heavy route to win when time/fuel_priority=1 and highway_priority=5, got %.1fkm (risk=%.1f)",
			best.DistanceKm, best.RiskScore)
	}
}

func TestRank_HigherCargoPriority_PenalizesSharpManeuversMore(t *testing.T) {
	candidates := []valhalla.RouteCandidate{
		{DistanceKm: 100, DurationMin: 100, ManeuverCount: 10, SharpManeuverCount: 0, HighwayRatio: 0.5},
		{DistanceKm: 100, DurationMin: 100, ManeuverCount: 10, SharpManeuverCount: 8, HighwayRatio: 0.5},
	}

	lowCargo := Rank(candidates, Preferences{FuelPriority: 3, CargoPriority: 1, HighwayPriority: 3, TimePriority: 3}, 40000, nil)
	highCargo := Rank(candidates, Preferences{FuelPriority: 3, CargoPriority: 5, HighwayPriority: 3, TimePriority: 3}, 40000, nil)

	// The smooth candidate (index 0) should win in both cases, but the margin
	// (risk score gap) should widen as cargo_priority increases.
	lowGap := lowCargo[1].RiskScore - lowCargo[0].RiskScore
	highGap := highCargo[1].RiskScore - highCargo[0].RiskScore
	if highGap <= lowGap {
		t.Errorf("expected higher cargo_priority to widen the smooth-vs-bumpy score gap: low=%.2f high=%.2f", lowGap, highGap)
	}
}

func TestRank_HeavierVehicle_PenalizedMoreByFuelTerm(t *testing.T) {
	candidates := []valhalla.RouteCandidate{
		{DistanceKm: 200, DurationMin: 100, ManeuverCount: 10, HighwayRatio: 0.5},
	}
	prefs := Preferences{FuelPriority: 5, CargoPriority: 3, HighwayPriority: 3, TimePriority: 3}

	light := Rank(candidates, prefs, 10000, nil)
	heavy := Rank(candidates, prefs, 60000, nil)

	if heavy[0].RiskScore <= light[0].RiskScore {
		t.Errorf("expected a 60000kg vehicle to score worse (higher) than a 10000kg one on the same route, got light=%.2f heavy=%.2f",
			light[0].RiskScore, heavy[0].RiskScore)
	}
}

// beogradNoviSadShape is a real polyline6-encoded shape fetched live from this
// project's own Valhalla instance (Belgrade, 44.8/20.4 -> a nearby point,
// 44.85/20.45 - decodes to 756 points, first one at 44.799996,20.399933).
// Used here just as "some real shape with real coordinates along it" to test
// preferredStopDiscount against genuine data, not synthetic points - an earlier
// version of this constant was a hand-truncated substring that cut a polyline6
// token in half and made DecodePolyline6 panic with an index-out-of-range.
const beogradNoviSadShape = "w~jmtAyrb|e@kJl@a@BWB_@BYB}@Fm@D}BPiEVi@DgRjAsDVoAHqAHWBsIj@{F\\oRhAySnAa@B}@Fu@DiAD_H]oBFaO`@qOb@iADuFXoGd@u@Fuf@xCkIf@yNl@mLr@gDPqDTeAFaDRgEVoQdA}]`CiMx@wHp@sIt@{Fb@qKp@qEPeENaEDiCF_DBiFDoJ?cEImHQqF]kFm@{IeA}B]wLkBiUiFmD}@yE_BmA]gA]mHuCkHoD{FwCqVoNkJyHqBcBwBeBwIiH_DiCcO_Ma`@i\\}AsA[W{S{QwHcHoHgH{RaP_@[qD}CsO}MiJeI_MuKcD{CaGkFaDsCeDgCqGqFmc@a`@{PyN_IgHkE{DiQeP_LeKaBcBuJsIwCeC_M}Jm@i@i{@su@kBaBeA_A{I}H}FqFcC}B}BgBqOyNeHmG{AmAcFqE{OiN}JaJg^{ZiKkJaAwDu@wEUeDCuD^kCr@mCdAqBpC}D~AcCdIyH`D_CzBy@rASlBCzB^vAlA`ArAp@rBt@pE@tDM`D{@bDaDpNs@`DuCpO}Pnf@_GlPyGfRyEzMerArtDsAvDqHzSgL|[iV~q@uFlOaUbn@{Yjy@kZ`{@cUho@wHrTwItVsHpSqNn`@sNxa@ci@jzAuAdEgJhXmJvXsIxUgH|SuKrZ_IjUkKvYaGpQmKr[mKlXaRbi@mPhf@mSzm@}GfUwDhMwV|{@wI~[cGhUyJr_@yBbJwB|IqJna@oHh\\wFrWgJxc@}Jzg@cJxf@gFhZoEjWiFr[mH`f@oGjc@iFt_@iGzf@eFvb@oCzUuGdo@kDv_@aGtp@cLvrAePpqBwFzs@{BrYaDrc@eC|]sEdq@iBhWoDve@qC~`@]hFcGtw@mDxUqDn[mCjR_BhIkBdIsFtRsA`E_BtDkBxDkB|CeCpDiDzDmCtC}C~BsCrB_FvBiD|@cEp@cDf@qEZiTFoHoAij@kL{]yHi\\aIeI{BkLgDgEiAyH_CuPeFkJ_D_F{AmCu@kEyAwLcEmc@mQ}d@kRm]gP_RgJiQuI{EcCoQsJyu@kd@eMyHgQuKiRsLcHgEw@e@wCiB{ByAgBmAuN}IgNaJei@c]gj@g^im@s_@cOwJmDwBkPiKyfAcs@_JwFeKwG}IuF_Q_LyGaE}NmJqAy@uAy@kMeIwCuByHaFkIaF{j@u^my@gi@cT}M}P{J{HoEmGkDqAo@uEmBuDeAyEiAo@IsDs@gEs@}Ca@aEa@mHSiHRuGZ_BLkHfAoCl@eS|Eyj@nPiT~FwT|DaUrBiKVeQP{Se@}RyAkMiBwVaFmW_IqIiDwI_E{FwCiDiBkHiEyHsFmToOsNmKeHsFiNkKuJcIm@e@wGgFgPeN{GkGcCyBsK{J}JaKa@g@oCkCsDuDaDkDg@e@eM_OyIsJkXaXwKcKgKiJwLcKcMyJutNcqLqfBoxAalBa}AyHiG_JqHit@gm@{tAyoA}rA_rAwmB{jBujBigBshBufBo_@e_@cf@{e@oe@_g@si@mm@c|@oeA{nBucCy`@qg@_`@}h@{x@scAgWyZaUuYgXw_@o[af@yb@}q@s[mi@yk@weAwg@wcAkZ{o@i_@o{@wa@wdA__@qbAmT}o@mNib@qXc}@qU}y@gS_x@wXsiA_Qkw@oI{a@aHo^aTalAeJ}j@cJql@qY{qBoTm~Aww@ytF{Mg`AgHcj@yHik@s\\gbCm\\}}By[w|BqNocAqO_bAeHib@qPybAeD{`@mAmM}@gKq@kJo@}N_@aQIkJCyHDeKRqKZaLj@iL|@wMjAgMbAiJ`BoLpBsLvAwHjBmIbBgHlCwJbDkKdCiHtCqHfDeI~CuGvC}F~DeHzDqGjF{HtEcG`D}DzFkGxGyGdE_EvDiDtJwJ|k@q\\dXqO|MyHlNeHpOwJxFoD`GuD~ReL|ToMhBgAbCwAdJqF`BmA`@e@b@k@n@_@tHmEb@WjBeAjh@yZnQoKpFwCjeAsn@zd@wXbKcGfH_EnAs@p@a@xDyBhLyGpHoEdX{OdNcIzf@yYzHsE~FeDjLyGz@e@v]eSn@]dAo@pEkCxw@ie@nY{PzZyQb@Wp@a@pBmAl\\uRzOiJrp@m`@|LkH`KuFpFaDhGmD~^cTDC`f@mYzG}Dt_@qTpt@kc@xXcPnGwD`N_ItIgFpGsDzHmErf@yYfBeA~CkBdAm@pToMdJkFnDwB~VgOnKeGrMmHhOwIfAo@|CgBpDuBnh@a[xVgO~LuHvEaB|n@g]rIoEtElFnKrLdE~CvWh\\~e@|l@tKrMvQlTj]~a@hQvTl@z@Zb@RXnInLnJdOzHhObIjPlBbEhF|KzP~\\|Spb@jQn]fz@baBVd@LT`@x@zBpE|@fBzHxOhLdV|HxOdBhDBF|EtJdAtBnEbJP^lE|IZp@d@~@N\\lCnFrF~KxBdEZj@tAjBvApAvBvArCjAfBb@zI\\lGTfABp@Bd@@`Qd@D@dk@|A|ADX?h@BZ?b[r@~Sn@|HTxCHGbF?x@eAphAAh@a@nd@AdAClAGfGm@`y@Af@?ZAvAw@hqACdDkAbhBAb@O|TCjPF|MFvOBzENnMDrCBnBPbEh@xMt@rEzAfDhCbCxBx@`Ef@pDYxMiCLCp@MZIdI}A~EaApEa@~D~@v@h@jBpAxCzClBfDnBpD~rArjCnKjQ`{@hsAnCnEvMbTvWnb@vKlQfTx]hTp]dR~ZbBjCxA`C|PbYVb@lDjGtAdCf@jAtB~Fd@dAfA`Cl@rArClFhIbN|G|K|Ztg@pu@xnA~P|XzC|E`KpPnPnXpd@bw@tg@dz@zCrFR^rGzL~A|C`DhHhAbE\\pBPnBtBEjBFzB`@tD`CdDvDz^`n@nEbD|FdAfVcBzZwFpHoP~KyXfNmTdTqElMxB~TdIl}@b`@x@mOsA{ECsIuAkLIyIN_IUaNtAaMxJgSlEoTvHy]nAqJlF}SfEaPRie@uCeg@_@gJkAaUvBe_@|H{XxGs`@vFej@hS{mAtAiMB}Nv@uIz@{L|DuQjEqO~E}LnC}OvByFbBaI~CyNnIkOzEiH`CaVjGsU|AwSdCkOdCgRb@{QxAeSfGeSfLgOtCqEr@uGfAaJrDkPlD_MvDmClDaEfAaJhB}UvPaq@Jk@"

func TestRank_PreferredStopNearRoute_GetsDiscount(t *testing.T) {
	candidates := []valhalla.RouteCandidate{
		{DistanceKm: 85, DurationMin: 68, ManeuverCount: 10, HighwayRatio: 0.6, Shape: beogradNoviSadShape},
	}
	prefs := Preferences{FuelPriority: 3, CargoPriority: 3, HighwayPriority: 3, TimePriority: 3}

	// First point of that shape is 44.799996,20.399933 (Belgrade) - a "preferred
	// stop" right on top of it must trigger the discount.
	onRoute := []valhalla.LatLon{{Lat: 44.799996, Lon: 20.399933}}
	farAway := []valhalla.LatLon{{Lat: 10.0, Lon: 10.0}}

	withDiscount := Rank(candidates, prefs, 40000, onRoute)
	without := Rank(candidates, prefs, 40000, nil)
	withFarStop := Rank(candidates, prefs, 40000, farAway)

	if withDiscount[0].RiskScore >= without[0].RiskScore {
		t.Errorf("expected a preferred stop on the route to lower the score: with=%.2f without=%.2f",
			withDiscount[0].RiskScore, without[0].RiskScore)
	}
	if withFarStop[0].RiskScore != without[0].RiskScore {
		t.Errorf("expected a far-away preferred stop to have no effect: far=%.2f without=%.2f",
			withFarStop[0].RiskScore, without[0].RiskScore)
	}
}

func TestNearestPreferredStopWithinRadius(t *testing.T) {
	onRoute := valhalla.LatLon{Lat: 44.799996, Lon: 20.399933}
	farAway := valhalla.LatLon{Lat: 10.0, Lon: 10.0}

	got, found := NearestPreferredStopWithinRadius(beogradNoviSadShape, []valhalla.LatLon{farAway, onRoute})
	if !found {
		t.Fatal("expected the on-route stop to be found")
	}
	if got != onRoute {
		t.Errorf("expected the exact matching stop %+v to be returned, got %+v", onRoute, got)
	}

	if _, found := NearestPreferredStopWithinRadius(beogradNoviSadShape, []valhalla.LatLon{farAway}); found {
		t.Error("expected no match when no preferred stop is near the route")
	}

	if _, found := NearestPreferredStopWithinRadius(beogradNoviSadShape, nil); found {
		t.Error("expected no match for an empty preferred stops list")
	}
}
