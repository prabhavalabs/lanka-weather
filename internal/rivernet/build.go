// Package rivernet turns HydroRIVERS segments, river names and gauge positions
// into the network the map and the risk estimate use: named rivers, and those
// rivers split into reaches each governed by the same gauges.
//
// Build is a pure function of its inputs, so the walks over the network are
// table-tested. The Collector reads the inputs, runs it when they change and
// stores the result.
package rivernet

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/irrigationgis"
)

// Segment is the topology of one HydroRIVERS line.
type Segment struct {
	ID, NextDown, MainRiv int64
	LengthKM              float64
	UplandKM2             float64
}

const (
	// minStemUplandKM2 is where a river's named line stops going upstream:
	// above it HydroRIVERS' lines are small streams, drawn as such.
	minStemUplandKM2 = 25
	// minRiverUplandKM2 makes a river of an unnamed system. Below it a system
	// without a gauge stays a stream on the map, with no panel of its own.
	minRiverUplandKM2 = 100
)

// Tributary names a stretch of river from its source segment. EndID is where
// the name is known to stop; 0 means it runs until it meets another stem.
type Tributary struct {
	SourceID, EndID int64
	Name            string
}

// Gauge is a gauge placed on the network.
type Gauge struct {
	// Kind is "station" (a row of river_stations, from any collector) or
	// "google" (a Google flood forecast point).
	Kind, ID string
	// Label names it in reach names: "Hanwella".
	Label     string
	SegmentID int64
}

// Inputs are what a network is built from.
type Inputs struct {
	// Segments may be in any order.
	Segments []Segment
	// SystemNames are names from the official basins, by outlet segment.
	SystemNames map[int64]SystemName
	// SourceNames is lk_rivers' table: an outlet's name becomes an alias, or
	// the name if the official basins give none. A name on any other segment
	// is the source of a tributary.
	SourceNames map[int64]string
	// Tributaries are further named tributaries, from the department's maps.
	Tributaries []Tributary
	Gauges      []Gauge
}

// River is a named or unnamed river. A tributary has a parent.
type River struct {
	ID, Name, NameSI string
	Aliases          []string
	Named            bool
	ParentID         string
	MainRiv, MouthID int64
	LengthKM         float64
	UplandKM2        float64
	BasinFID         int64
}

// Reach is a stretch of a river governed by the same gauges.
type Reach struct {
	// FID is the HydroRIVERS id of its most downstream segment.
	FID      int64
	RiverID  string
	Name     string
	Segments []int64
	// Upstream are the gauges whose water reaches this stretch; Downstream is
	// the gauge at its lower end, if the river has one before the next
	// confluence with a larger river. A reach with neither Upstream gauge is
	// ungauged, and Next names the first gauge below it.
	Upstream   []GaugeRef
	Downstream *GaugeRef
	Next       *GaugeRef
	LengthKM   float64
	UplandKM2  float64
	// Position orders a river's reaches from the source to the sea.
	Position int
}

// Gauged reports whether anything measures this stretch.
func (reach Reach) Gauged() bool { return len(reach.Upstream) > 0 }

// GaugeRef identifies a gauge.
type GaugeRef struct {
	Kind, ID, Label string
}

// Network is the built result.
type Network struct {
	Rivers       []River
	Reaches      []Reach
	SegmentRiver map[int64]string
	SegmentReach map[int64]int64
}

type builder struct {
	segments map[int64]*Segment
	upstream map[int64][]int64
	// river assigns a stem segment to a river; stems are chains of segments.
	river   map[int64]string
	stems   []*stem
	stemOf  map[int64]*stem
	gauges  map[int64][]GaugeRef
	rivers  map[string]*River
	nameIDs map[string]bool
}

// stem is one chain of segments of a river, listed from the top down. A river
// has one main stem, and more where a branch carries its name or where a gauge
// stands on an unnamed tributary.
type stem struct {
	river    string
	segments []int64
}

// Build constructs the network. It never fails: names and gauges that do not
// fit the segments are ignored.
func Build(inputs Inputs) Network {
	b := &builder{
		segments: make(map[int64]*Segment, len(inputs.Segments)),
		upstream: make(map[int64][]int64, len(inputs.Segments)),
		river:    map[int64]string{},
		stemOf:   map[int64]*stem{},
		gauges:   map[int64][]GaugeRef{},
		rivers:   map[string]*River{},
		nameIDs:  map[string]bool{},
	}
	for index := range inputs.Segments {
		segment := &inputs.Segments[index]
		b.segments[segment.ID] = segment
		if segment.NextDown != 0 {
			b.upstream[segment.NextDown] = append(b.upstream[segment.NextDown], segment.ID)
		}
	}
	for _, list := range b.upstream {
		sort.Slice(list, func(i, j int) bool {
			a, c := b.segments[list[i]], b.segments[list[j]]
			if a.UplandKM2 != c.UplandKM2 {
				return a.UplandKM2 > c.UplandKM2
			}
			return a.ID < c.ID
		})
	}
	for _, gauge := range inputs.Gauges {
		if _, ok := b.segments[gauge.SegmentID]; ok {
			b.gauges[gauge.SegmentID] = append(b.gauges[gauge.SegmentID], GaugeRef{Kind: gauge.Kind, ID: gauge.ID, Label: gauge.Label})
		}
	}
	for _, list := range b.gauges {
		sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	}

	b.mainRivers(inputs)
	b.namedTributaries(inputs)
	b.gaugedTributaries()

	network := Network{SegmentRiver: map[int64]string{}, SegmentReach: map[int64]int64{}}
	for id, river := range b.river {
		network.SegmentRiver[id] = river
	}
	for _, river := range b.rivers {
		network.Rivers = append(network.Rivers, *river)
	}
	sort.Slice(network.Rivers, func(i, j int) bool {
		if network.Rivers[i].UplandKM2 != network.Rivers[j].UplandKM2 {
			return network.Rivers[i].UplandKM2 > network.Rivers[j].UplandKM2
		}
		return network.Rivers[i].ID < network.Rivers[j].ID
	})
	b.splitReaches(&network)
	return network
}

// mainRivers names each system's outlet and walks up its largest branch.
func (b *builder) mainRivers(inputs Inputs) {
	var outlets []*Segment
	for _, segment := range b.segments {
		if segment.NextDown == 0 {
			outlets = append(outlets, segment)
		}
	}
	sort.Slice(outlets, func(i, j int) bool {
		if outlets[i].UplandKM2 != outlets[j].UplandKM2 {
			return outlets[i].UplandKM2 > outlets[j].UplandKM2
		}
		return outlets[i].ID < outlets[j].ID
	})
	gauged := map[int64]bool{}
	for id := range b.gauges {
		gauged[b.segments[id].MainRiv] = true
	}
	for _, outlet := range outlets {
		system, hasOfficial := inputs.SystemNames[outlet.ID]
		source := inputs.SourceNames[outlet.ID]
		name := system.Name
		if !hasOfficial {
			name = source
		}
		if name == "" && outlet.UplandKM2 < minRiverUplandKM2 && !gauged[outlet.MainRiv] {
			continue
		}
		official := name
		if display, ok := displayNames[name]; ok {
			name = display
		}
		var aliases []string
		for _, alias := range []string{official, source} {
			if alias != "" && Fold(alias) != Fold(name) && !containsFold(aliases, alias) {
				aliases = append(aliases, alias)
			}
		}
		river := &River{
			ID: b.riverID(name, outlet.ID), Name: name, NameSI: siName[name], Aliases: aliases, Named: name != "",
			MainRiv: outlet.MainRiv, MouthID: outlet.ID, UplandKM2: outlet.UplandKM2, BasinFID: system.BasinFID,
		}
		b.rivers[river.ID] = river
		b.addStem(river, b.walkUp(outlet.ID))
	}
}

// riverID slugs a name; an unnamed river, or a name used twice, adds its id.
func (b *builder) riverID(name string, id int64) string {
	slug := irrigationgis.Slug(name)
	if slug == "" {
		slug = "river-" + strconv.FormatInt(id, 10)
	} else if b.nameIDs[slug] {
		slug += "-" + strconv.FormatInt(id, 10)
	}
	b.nameIDs[slug] = true
	return slug
}

// walkUp lists the chain from a segment up its largest branch, top first,
// stopping at small streams and at segments already on a stem.
func (b *builder) walkUp(from int64) []int64 {
	chain := []int64{from}
	for current := from; ; {
		next := int64(0)
		for _, candidate := range b.upstream[current] {
			if _, taken := b.river[candidate]; !taken && b.segments[candidate].UplandKM2 >= minStemUplandKM2 {
				next = candidate
			}
			break // only ever the largest branch
		}
		if next == 0 {
			break
		}
		chain = append(chain, next)
		current = next
	}
	reverse(chain)
	return chain
}

func reverse(list []int64) {
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
}

func (b *builder) addStem(river *River, chain []int64) {
	if len(chain) == 0 {
		return
	}
	part := &stem{river: river.ID, segments: chain}
	b.stems = append(b.stems, part)
	for _, id := range chain {
		b.river[id] = river.ID
		b.stemOf[id] = part
		river.LengthKM += b.segments[id].LengthKM
	}
}

// walkDown lists the chain from a source down to the first segment on a stem,
// which it leaves out, and returns that junction (0 at the sea).
func (b *builder) walkDown(from int64) (chain []int64, junction int64) {
	for current := from; current != 0; {
		segment, ok := b.segments[current]
		if !ok {
			return chain, 0
		}
		if _, taken := b.river[current]; taken {
			return chain, current
		}
		chain = append(chain, current)
		current = segment.NextDown
	}
	return chain, 0
}

// namedTributaries adds named tributaries, in two tiers. First those whose
// names the department or lk_rivers states directly, then those read from the
// department's stream polygons, which fill in what is left. Within a tier the
// largest goes first, so a small stream never takes a larger one's course. A
// polygon name runs only as far as the polygons say: the stretch below it, if
// nothing else claims it, then joins it.
func (b *builder) namedTributaries(inputs Inputs) {
	var stated []Tributary
	for id, name := range inputs.SourceNames {
		if segment, ok := b.segments[id]; ok && segment.NextDown != 0 {
			stated = append(stated, Tributary{SourceID: id, Name: name})
		}
	}
	var read []Tributary
	for _, tributary := range inputs.Tributaries {
		if tributary.EndID == 0 {
			stated = append(stated, tributary)
		} else {
			read = append(read, tributary)
		}
	}
	var open []*tributaryStem
	open = append(open, b.claimTier(stated)...)
	open = append(open, b.claimTier(read)...)
	sort.SliceStable(open, func(i, j int) bool { return b.segments[open[i].end].UplandKM2 > b.segments[open[j].end].UplandKM2 })
	for _, part := range open {
		chain, junction := b.walkDown(b.segments[part.end].NextDown)
		b.addStem(part.river, chain)
		if part.river.ParentID == "" && b.rivers[b.river[junction]] != part.river {
			part.river.ParentID = b.river[junction]
		}
		if len(chain) > 0 {
			last := b.segments[chain[len(chain)-1]]
			part.river.MouthID, part.river.UplandKM2 = last.ID, max(part.river.UplandKM2, last.UplandKM2)
		}
	}
}

// tributaryStem is a claimed chain still open at its lower end.
type tributaryStem struct {
	river *River
	end   int64
}

func (b *builder) claimTier(pending []Tributary) []*tributaryStem {
	sort.Slice(pending, func(i, j int) bool { return pending[i].SourceID < pending[j].SourceID })
	var open []*tributaryStem
	for len(pending) > 0 {
		bestIndex, bestUpland := -1, -1.0
		for index, candidate := range pending {
			chain := b.claimable(candidate)
			if len(chain) == 0 {
				continue
			}
			if upland := b.segments[chain[len(chain)-1]].UplandKM2; upland > bestUpland {
				bestIndex, bestUpland = index, upland
			}
		}
		if bestIndex < 0 {
			return open
		}
		chosen := pending[bestIndex]
		pending = append(pending[:bestIndex], pending[bestIndex+1:]...)
		chain := b.claimable(chosen)
		last := b.segments[chain[len(chain)-1]]
		if last.UplandKM2 < minStemUplandKM2 {
			continue // a named headwater too small to draw as a river
		}
		river := b.tributaryRiver(chosen.Name, chain, last)
		if river == nil {
			continue
		}
		b.addStem(river, chain)
		open = append(open, &tributaryStem{river: river, end: last.ID})
	}
	return open
}

// claimable lists a tributary's chain from its source down to its end, stopping
// at any segment already on a stem.
func (b *builder) claimable(tributary Tributary) []int64 {
	chain, _ := b.walkDown(tributary.SourceID)
	if tributary.EndID == 0 {
		return chain
	}
	for index, id := range chain {
		if id == tributary.EndID {
			return chain[:index+1]
		}
	}
	return chain
}

// tributaryRiver finds or makes the river a chain belongs to. A name the system
// already has continues that river: two branches called Gurugoda Oya are one
// river. A chain carrying the name of its system's main river counts only if
// it joins that river directly; lk_rivers calls a headwater of the Gurugoda Oya
// "Kelani Ganga", which is not where the Kelani runs.
func (b *builder) tributaryRiver(name string, chain []int64, last *Segment) *River {
	for _, existing := range b.rivers {
		if existing.MainRiv != last.MainRiv || Fold(existing.Name) != Fold(name) {
			continue
		}
		if b.segments[existing.MouthID].NextDown != 0 {
			return existing // a tributary, not the system's main river
		}
		if _, junction := b.walkDown(last.ID); b.river[junction] == existing.ID {
			return existing
		}
		return nil
	}
	river := &River{
		ID: b.riverID(name, chain[0]), Name: name, NameSI: siName[name], Named: true,
		MainRiv: last.MainRiv, MouthID: last.ID, UplandKM2: last.UplandKM2,
	}
	b.rivers[river.ID] = river
	return river
}

// gaugedTributaries gives a gauge standing on an unnamed tributary a stem from
// the gauge to where the tributary meets a river, so the gauge can govern it.
func (b *builder) gaugedTributaries() {
	var ids []int64
	for id := range b.gauges {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, c := b.segments[ids[i]], b.segments[ids[j]]
		if a.UplandKM2 != c.UplandKM2 {
			return a.UplandKM2 > c.UplandKM2
		}
		return a.ID < c.ID
	})
	for _, id := range ids {
		chain, junction := b.walkDown(id)
		if len(chain) == 0 {
			continue
		}
		parent := b.river[junction]
		river := b.rivers[parent]
		if river == nil {
			// The whole system is unnamed and small, yet it has a gauge.
			last := b.segments[chain[len(chain)-1]]
			river = &River{ID: b.riverID("", last.ID), MainRiv: last.MainRiv, MouthID: last.ID, UplandKM2: last.UplandKM2}
			b.rivers[river.ID] = river
		}
		b.addStem(river, chain)
	}
}

// splitReaches cuts every stem wherever the gauges governing it change.
func (b *builder) splitReaches(network *Network) {
	governing := b.upstreamGauges()
	for _, part := range b.stems {
		var current *Reach
		var currentKey string
		flush := func() {
			if current == nil {
				return
			}
			last := b.segments[current.Segments[len(current.Segments)-1]]
			current.FID = last.ID
			current.UplandKM2 = last.UplandKM2
			current.Name = reachName(*current)
			for _, id := range current.Segments {
				network.SegmentReach[id] = current.FID
			}
			network.Reaches = append(network.Reaches, *current)
			current = nil
		}
		for _, id := range part.segments {
			up := governing[id]
			down := b.nextGaugeBelow(id, part)
			key := refsKey(up) + "|"
			if down != nil {
				key += down.ID
			}
			if current == nil || key != currentKey {
				flush()
				current = &Reach{RiverID: part.river, Upstream: up}
				currentKey = key
				if len(up) > 0 {
					current.Downstream = down
				} else {
					current.Next = down
				}
			}
			current.Segments = append(current.Segments, id)
			current.LengthKM += b.segments[id].LengthKM
		}
		flush()
	}
	sort.SliceStable(network.Reaches, func(i, j int) bool {
		a, c := network.Reaches[i], network.Reaches[j]
		if a.RiverID != c.RiverID {
			return a.RiverID < c.RiverID
		}
		return a.UplandKM2 < c.UplandKM2
	})
	position, river := 0, ""
	for index := range network.Reaches {
		if network.Reaches[index].RiverID != river {
			river, position = network.Reaches[index].RiverID, 0
		}
		network.Reaches[index].Position = position
		position++
	}
}

// upstreamGauges finds, for every stem segment, the nearest gauges upstream or
// on it. At a confluence the largest branch that has any gauge decides: a small
// tributary in flood should not colour a main stem whose own gauge reads calm.
func (b *builder) upstreamGauges() map[int64][]GaugeRef {
	memo := map[int64][]GaugeRef{}
	done := map[int64]bool{}
	// Iterating from small to large upland means upstream is always done first.
	ordered := make([]*Segment, 0, len(b.river))
	for id := range b.river {
		ordered = append(ordered, b.segments[id])
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].UplandKM2 != ordered[j].UplandKM2 {
			return ordered[i].UplandKM2 < ordered[j].UplandKM2
		}
		return ordered[i].ID < ordered[j].ID
	})
	var resolve func(id int64) []GaugeRef
	resolve = func(id int64) []GaugeRef {
		if done[id] {
			return memo[id]
		}
		done[id] = true
		if here := b.gauges[id]; len(here) > 0 {
			memo[id] = here
			return here
		}
		for _, candidate := range b.upstream[id] {
			if _, onStem := b.river[candidate]; !onStem {
				continue
			}
			if found := resolve(candidate); len(found) > 0 {
				memo[id] = found
				return found
			}
		}
		return nil
	}
	for _, segment := range ordered {
		resolve(segment.ID)
	}
	return memo
}

// nextGaugeBelow finds the first gauge downstream on the same stem. It stops at
// the stem's end: a gauge far along the larger river says nothing of a
// tributary's lower course.
func (b *builder) nextGaugeBelow(id int64, part *stem) *GaugeRef {
	passed := false
	for _, candidate := range part.segments {
		if passed && len(b.gauges[candidate]) > 0 {
			ref := b.gauges[candidate][0]
			return &ref
		}
		if candidate == id {
			passed = true
		}
	}
	return nil
}

func containsFold(list []string, name string) bool {
	for _, item := range list {
		if Fold(item) == Fold(name) {
			return true
		}
	}
	return false
}

func refsKey(refs []GaugeRef) string {
	parts := make([]string, len(refs))
	for index, ref := range refs {
		parts[index] = ref.Kind + ":" + ref.ID
	}
	return strings.Join(parts, ",")
}

func labels(refs []GaugeRef) string {
	parts := make([]string, len(refs))
	for index, ref := range refs {
		parts[index] = ref.Label
	}
	return strings.Join(parts, " and ")
}

// reachName says where a stretch runs, in terms of its gauges.
func reachName(reach Reach) string {
	switch {
	case len(reach.Upstream) > 0 && reach.Downstream != nil:
		return fmt.Sprintf("%s to %s", labels(reach.Upstream), reach.Downstream.Label)
	case len(reach.Upstream) > 0:
		return "Below " + labels(reach.Upstream)
	case reach.Next != nil:
		return "Above " + reach.Next.Label
	}
	return "Whole river"
}
