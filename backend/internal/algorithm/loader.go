package algorithm

import (
	"encoding/xml"
	"os"
	"strconv"
	"strings"
)

type osmXML struct {
	XMLName xml.Name  `xml:"osm"`
	Nodes   []osmNode `xml:"node"`
	Ways    []osmWay  `xml:"way"`
}

type osmNode struct {
	ID   int64    `xml:"id,attr"`
	Lat  float64  `xml:"lat,attr"`
	Lon  float64  `xml:"lon,attr"`
	Tags []osmTag `xml:"tag"`
}

func (n osmNode) tag(key string) (string, bool) {
	for _, t := range n.Tags {
		if t.K == key {
			return t.V, true
		}
	}
	return "", false
}

type osmWay struct {
	Nds  []osmNd  `xml:"nd"`
	Tags []osmTag `xml:"tag"`
}

type osmNd struct {
	Ref int64 `xml:"ref,attr"`
}

type osmTag struct {
	K string `xml:"k,attr"`
	V string `xml:"v,attr"`
}

func (w osmWay) tag(key string) (string, bool) {
	for _, t := range w.Tags {
		if t.K == key {
			return t.V, true
		}
	}
	return "", false
}

// parseMeters parses OSM maxheight/maxwidth-style values ("4.5", "4.5 m"). Returns
// 0 (no restriction) for missing, unparseable, or "default" values - "default" means
// "the country's legal default applies", which isn't a specific number we can enforce.
func parseMeters(raw string) float64 {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, "m")
	raw = strings.TrimSpace(raw)
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return v
}

// parseTons parses OSM maxweight-style values ("12", "12.0", "12 t").
func parseTons(raw string) float64 {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, "t")
	raw = strings.TrimSpace(raw)
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return v
}

// LoadOSMXML builds a Graph from an OSM XML file (produced by `osmium cat -f osm`,
// see documentations/guides/extract-osm-corridor.md). Only ways are turned into
// edges; every node referenced by an included way is kept.
func LoadOSMXML(path string) (*Graph, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var doc osmXML
	if err := xml.NewDecoder(f).Decode(&doc); err != nil {
		return nil, err
	}

	g := NewGraph()
	for _, n := range doc.Nodes {
		node := Node{ID: n.ID, Lat: n.Lat, Lon: n.Lon}
		// Only treat this as a routing restriction when the node is BOTH a
		// barrier AND carries its own maxheight/maxweight tag - the standard
		// OSM convention for barrier=height_restrictor/lift_gate/etc (see
		// documentations/guides/extract-osm-corridor.md). A bare barrier=gate
		// with no dimension tag isn't something we can turn into a vehicle
		// exclusion, so it's deliberately left unrestricted.
		if _, isBarrier := n.tag("barrier"); isBarrier {
			if v, ok := n.tag("maxheight"); ok {
				node.MaxHeightM = parseMeters(v)
			}
			if v, ok := n.tag("maxweight"); ok {
				node.MaxWeightT = parseTons(v)
			}
		}
		g.Nodes[n.ID] = node
	}

	for _, w := range doc.Ways {
		maxHeight := 0.0
		maxWeight := 0.0
		hazmat := false
		surface := ""
		oneway := false

		if v, ok := w.tag("maxheight"); ok {
			maxHeight = parseMeters(v)
		}
		if v, ok := w.tag("maxweight"); ok {
			maxWeight = parseTons(v)
		}
		if v, ok := w.tag("hazmat"); ok && v == "no" {
			hazmat = true
		}
		if v, ok := w.tag("surface"); ok {
			surface = v
		}
		highway, _ := w.tag("highway")
		if v, ok := w.tag("oneway"); ok {
			oneway = v == "yes"
		} else if highway == "motorway" {
			oneway = true // OSM convention: motorways are oneway unless tagged oneway=no
		}

		for i := 0; i+1 < len(w.Nds); i++ {
			from, to := w.Nds[i].Ref, w.Nds[i+1].Ref
			fromNode, fromOK := g.Nodes[from]
			toNode, toOK := g.Nodes[to]
			if !fromOK || !toOK {
				continue // node outside the extract (shouldn't happen with complete_ways, but be defensive)
			}
			length := haversineMeters(fromNode.Lat, fromNode.Lon, toNode.Lat, toNode.Lon)

			g.addEdge(from, Edge{To: to, LengthM: length, MaxHeightM: maxHeight, MaxWeightT: maxWeight, Hazmat: hazmat, Surface: surface, RoadClass: highway})
			if !oneway {
				g.addEdge(to, Edge{To: from, LengthM: length, MaxHeightM: maxHeight, MaxWeightT: maxWeight, Hazmat: hazmat, Surface: surface, RoadClass: highway})
			}
		}
	}

	return g, nil
}
