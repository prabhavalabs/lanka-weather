package grid

import "math"

// DistanceKm is the great-circle distance between two points.
func DistanceKm(lon1, lat1, lon2, lat2 float64) float64 {
	const earthRadiusKm = 6371.0088
	toRadians := math.Pi / 180
	dLat, dLon := (lat2-lat1)*toRadians, (lon2-lon1)*toRadians
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*toRadians)*math.Cos(lat2*toRadians)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(a)))
}

// CoastDistanceKm is the distance from a point to Sri Lanka's nearest coast,
// 0 on land. It measures to the coastline's vertices, about a kilometre apart,
// which is ample for storms and earthquakes hundreds of kilometres away.
func CoastDistanceKm(lon, lat float64) float64 {
	if OnLand(lon, lat) {
		return 0
	}
	best := math.Inf(1)
	for _, ring := range coastline {
		for _, vertex := range ring {
			best = math.Min(best, DistanceKm(lon, lat, vertex[0], vertex[1]))
		}
	}
	return best
}

// NearestCoast is the point of Sri Lanka's coastline nearest a point at sea and
// its distance; for a point on land, the point itself and 0.
func NearestCoast(lon, lat float64) (coastLon, coastLat, km float64) {
	if OnLand(lon, lat) {
		return lon, lat, 0
	}
	km = math.Inf(1)
	for _, ring := range coastline {
		for _, vertex := range ring {
			if distance := DistanceKm(lon, lat, vertex[0], vertex[1]); distance < km {
				coastLon, coastLat, km = vertex[0], vertex[1], distance
			}
		}
	}
	return coastLon, coastLat, km
}

// Compass is the eight-point direction from one point to another: N, NE, E,
// SE, S, SW, W or NW.
func Compass(fromLon, fromLat, toLon, toLat float64) string {
	toRadians := math.Pi / 180
	phi1, phi2 := fromLat*toRadians, toLat*toRadians
	dLon := (toLon - fromLon) * toRadians
	y := math.Sin(dLon) * math.Cos(phi2)
	x := math.Cos(phi1)*math.Sin(phi2) - math.Sin(phi1)*math.Cos(phi2)*math.Cos(dLon)
	bearing := math.Mod(math.Atan2(y, x)/toRadians+360, 360)
	return [...]string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}[int(math.Round(bearing/45))%8]
}
