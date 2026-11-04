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
