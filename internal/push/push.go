// Package push holds Web Push alerts (ADR 0018): subscriptions and their
// preferences, the events the alert engine raises, how events are matched to
// subscribers and gathered into notices, and their delivery.
//
// There are no accounts. A subscription's endpoint is its identity and its
// auth secret proves ownership. Nothing else about the person is stored.
package push

import (
	"math"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Severity orders notices. Official sources set it; forecasts and estimates
// say what they are in their text whatever their severity.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWatch   Severity = "watch"
	SeverityWarning Severity = "warning"
	SeverityDanger  Severity = "danger"
)

// Rank orders severities from info (1) to danger (4); 0 is unknown.
func (severity Severity) Rank() int {
	switch severity {
	case SeverityInfo:
		return 1
	case SeverityWatch:
		return 2
	case SeverityWarning:
		return 3
	case SeverityDanger:
		return 4
	}
	return 0
}

// Topic is a kind of notice a subscriber chooses.
type Topic string

const (
	TopicOfficial        Topic = "official"
	TopicLandslide       Topic = "landslide"
	TopicLandslideLevel1 Topic = "landslide_level1"
	TopicRiver           Topic = "river"
	TopicFloodForecast   Topic = "flood_forecast"
	TopicCyclone         Topic = "cyclone"
	TopicEarthquake      Topic = "earthquake"
	TopicReservoir       Topic = "reservoir"
	// TopicHeavyRain is rain measured at the live gauges.
	TopicHeavyRain Topic = "heavy_rain"
	// TopicFloodRisk is the combined flood-risk estimate (package floodrisk).
	TopicFloodRisk Topic = "flood_risk"
)

// Topics are every topic, in the order the settings list them.
var Topics = []Topic{TopicOfficial, TopicLandslide, TopicLandslideLevel1, TopicRiver, TopicHeavyRain, TopicFloodRisk,
	TopicFloodForecast, TopicCyclone, TopicEarthquake, TopicReservoir}

// DefaultTopics are on for a new subscription: official warnings, landslide
// levels 2 and 3, river flood levels, heavy rain measured, the flood-risk
// estimate, flood forecasts, cyclones and large earthquakes. Level 1 landslides
// and reservoir spills are opt-in.
var DefaultTopics = []Topic{TopicOfficial, TopicLandslide, TopicRiver, TopicHeavyRain, TopicFloodRisk, TopicFloodForecast,
	TopicCyclone, TopicEarthquake}

// DefaultMinSeverity is a new subscription's threshold.
const DefaultMinSeverity = SeverityWarning

// Locales are the languages notices are written in.
var Locales = []string{"en", "si", "ta"}

// Payload is what the service worker receives: the contract with
// apps/web/src/sw.ts. Do not change it without changing the worker.
type Payload struct {
	Title              string    `json:"title"`
	Body               string    `json:"body"`
	URL                string    `json:"url"`
	Tag                string    `json:"tag"`
	Severity           Severity  `json:"severity"`
	SentAt             time.Time `json:"sentAt"`
	RequireInteraction bool      `json:"requireInteraction,omitempty"`
	Lang               string    `json:"lang,omitempty"`
}

// Near is a "near me" area: a point rounded to about a kilometre and a radius.
type Near struct {
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	RadiusKm int     `json:"radiusKm"`
}

// Areas are where a subscriber wants notices for. Country takes in all of Sri
// Lanka; otherwise districts and a point with a radius, either or both.
type Areas struct {
	Country   bool     `json:"country"`
	Districts []string `json:"districts"`
	Near      *Near    `json:"near"`
}

// Preferences are a subscription's choices.
type Preferences struct {
	Topics      []Topic  `json:"topics"`
	MinSeverity Severity `json:"minSeverity"`
	Areas       Areas    `json:"areas"`
}

// DefaultPreferences are the whole country, the default topics and warning
// or worse.
func DefaultPreferences() Preferences {
	return Preferences{Topics: slices.Clone(DefaultTopics), MinSeverity: DefaultMinSeverity, Areas: Areas{Country: true, Districts: []string{}}}
}

// Radii a "near me" area may have, in km.
const (
	MinRadiusKm = 1
	MaxRadiusKm = 100
)

// Bounds of a "near me" point: the area the API answers for.
const (
	southBound, northBound = 5.5, 10.5
	westBound, eastBound   = 79.0, 82.5
)

// RoundPoint rounds a coordinate to two decimals, about 1.1 km: the most
// precise location kept.
func RoundPoint(value float64) float64 { return math.Round(value*100) / 100 }

// ValidationError is a request that cannot be accepted, with a code for the API.
type ValidationError struct {
	Code    string
	Message string
}

func (err *ValidationError) Error() string { return err.Message }

func invalid(code, message string) error { return &ValidationError{Code: code, Message: message} }

var districtCode = regexp.MustCompile(`^LK-\d{2}$`)

// Normalise checks preferences and returns them in canonical form: topics
// known, sorted and unique; districts known (when known is not nil), sorted
// and unique; the point rounded.
func (preferences Preferences) Normalise(known map[string]bool) (Preferences, error) {
	out := Preferences{MinSeverity: preferences.MinSeverity, Areas: Areas{Country: preferences.Areas.Country, Districts: []string{}}}
	if out.MinSeverity == "" {
		out.MinSeverity = DefaultMinSeverity
	}
	if out.MinSeverity.Rank() == 0 {
		return Preferences{}, invalid("invalid_severity", "minSeverity must be info, watch, warning or danger")
	}
	if len(preferences.Topics) > len(Topics) {
		return Preferences{}, invalid("invalid_topics", "too many topics")
	}
	for _, topic := range preferences.Topics {
		if !slices.Contains(Topics, topic) {
			return Preferences{}, invalid("invalid_topics", "unknown topic "+string(topic))
		}
		if !slices.Contains(out.Topics, topic) {
			out.Topics = append(out.Topics, topic)
		}
	}
	if len(out.Topics) == 0 {
		return Preferences{}, invalid("invalid_topics", "choose at least one topic")
	}
	slices.SortFunc(out.Topics, func(a, b Topic) int { return slices.Index(Topics, a) - slices.Index(Topics, b) })
	if len(preferences.Areas.Districts) > 25 {
		return Preferences{}, invalid("invalid_districts", "at most 25 districts")
	}
	for _, code := range preferences.Areas.Districts {
		code = strings.ToUpper(strings.TrimSpace(code))
		if !districtCode.MatchString(code) || (known != nil && !known[code]) {
			return Preferences{}, invalid("invalid_districts", "unknown district "+code)
		}
		if !slices.Contains(out.Areas.Districts, code) {
			out.Areas.Districts = append(out.Areas.Districts, code)
		}
	}
	slices.Sort(out.Areas.Districts)
	if near := preferences.Areas.Near; near != nil {
		if math.IsNaN(near.Lat) || math.IsNaN(near.Lon) || near.Lat < southBound || near.Lat > northBound || near.Lon < westBound || near.Lon > eastBound {
			return Preferences{}, invalid("invalid_location", "the location must be in Sri Lanka")
		}
		if near.RadiusKm < MinRadiusKm || near.RadiusKm > MaxRadiusKm {
			return Preferences{}, invalid("invalid_radius", "radiusKm must be from 1 to 100")
		}
		out.Areas.Near = &Near{Lat: RoundPoint(near.Lat), Lon: RoundPoint(near.Lon), RadiusKm: near.RadiusKm}
	}
	if !out.Areas.Country && len(out.Areas.Districts) == 0 && out.Areas.Near == nil {
		return Preferences{}, invalid("invalid_areas", "choose the whole country, districts or a location")
	}
	return out, nil
}

// ValidLocale reports whether notices can be written in a language.
func ValidLocale(locale string) bool { return slices.Contains(Locales, locale) }

// allowedHosts are the push services subscriptions may point at: Google's FCM
// (Chrome, Edge on Android, Samsung Internet, Opera), Mozilla's autopush,
// Apple's Web Push and Microsoft's WNS.
var allowedHosts = []string{"fcm.googleapis.com", "updates.push.services.mozilla.com", "web.push.apple.com"}

var allowedSuffixes = []string{".push.services.mozilla.com", ".notify.windows.com", ".push.apple.com"}

// MaxEndpointLength bounds an endpoint URL.
const MaxEndpointLength = 1024

// AllowedEndpoint checks that an endpoint is an https URL on a known push
// service, so the server never posts to an address someone chose.
func AllowedEndpoint(endpoint string) error {
	if len(endpoint) > MaxEndpointLength {
		return invalid("invalid_endpoint", "the endpoint is too long")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Host == "" || parsed.Fragment != "" {
		return invalid("invalid_endpoint", "the endpoint must be an https URL")
	}
	if port := parsed.Port(); port != "" && port != "443" {
		return invalid("invalid_endpoint", "the endpoint must use the https port")
	}
	host := strings.ToLower(parsed.Hostname())
	if slices.Contains(allowedHosts, host) {
		return nil
	}
	for _, suffix := range allowedSuffixes {
		if strings.HasSuffix(host, suffix) && len(host) > len(suffix) {
			return nil
		}
	}
	return invalid("unsupported_push_service", "the endpoint is not on a supported push service")
}

// UserAgentFamily reduces a User-Agent to its browser family and platform,
// such as "chrome-android" or "safari-ios". Nothing finer is kept.
func UserAgentFamily(userAgent string) string {
	ua := strings.ToLower(userAgent)
	platform := "other"
	switch {
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ipod"):
		platform = "ios"
	case strings.Contains(ua, "android"):
		platform = "android"
	case strings.Contains(ua, "windows"):
		platform = "windows"
	case strings.Contains(ua, "mac os x") || strings.Contains(ua, "macintosh"):
		platform = "macos"
	case strings.Contains(ua, "linux") || strings.Contains(ua, "cros"):
		platform = "linux"
	}
	browser := "other"
	switch {
	case strings.Contains(ua, "edg/") || strings.Contains(ua, "edga/") || strings.Contains(ua, "edgios/"):
		browser = "edge"
	case strings.Contains(ua, "samsungbrowser"):
		browser = "samsung"
	case strings.Contains(ua, "opr/") || strings.Contains(ua, "opera"):
		browser = "opera"
	case strings.Contains(ua, "firefox/") || strings.Contains(ua, "fxios/"):
		browser = "firefox"
	case strings.Contains(ua, "chrome/") || strings.Contains(ua, "crios/") || strings.Contains(ua, "chromium/"):
		browser = "chrome"
	case strings.Contains(ua, "safari/"):
		browser = "safari"
	}
	return browser + "-" + platform
}

// Event is something the alert engine raised: a warning newly in force, a
// gauge rising across a level. Kind and Variant pick its text (package
// messages); Facts fill it.
type Event struct {
	Key      string
	Kind     string
	Variant  string
	Topic    Topic
	Severity Severity
	Subject  string
	// National events go to everyone with the topic.
	National  bool
	Districts []string
	// Lon and Lat place a point event (a gauge), for "near me".
	Lon, Lat   *float64
	Facts      map[string]string
	URL        string
	Tag        string
	DetectedAt time.Time
	ExpiresAt  time.Time
}

// Subscriber is a subscription as matching and delivery need it.
type Subscriber struct {
	ID          int64
	Endpoint    string
	P256dh      string
	Auth        string
	Locale      string
	Preferences Preferences
	// NearDistricts are the districts within the "near me" radius.
	NearDistricts []string
	CreatedAt     time.Time
	RetryAfter    *time.Time
}

// Wants reports whether a subscriber asked for an event: its topic, at least
// its minimum severity (except opt-in topics), and in its area. An event raised before the
// subscription existed is not sent: the app shows what is already in force.
func (subscriber Subscriber) Wants(event Event) bool {
	preferences := subscriber.Preferences
	if event.DetectedAt.Before(subscriber.CreatedAt) {
		return false
	}
	if !slices.Contains(preferences.Topics, event.Topic) {
		return false
	}
	// An opt-in topic is its own choice of severity: ticking "Level 1" or
	// "reservoir spills" means wanting them, whatever the threshold.
	if event.Severity.Rank() < preferences.MinSeverity.Rank() && !slices.Contains(OptInTopics, event.Topic) {
		return false
	}
	return subscriber.inArea(event)
}

// OptInTopics are off by default and, once chosen, are sent whatever the
// minimum severity.
var OptInTopics = []Topic{TopicLandslideLevel1, TopicReservoir}

func (subscriber Subscriber) inArea(event Event) bool {
	areas := subscriber.Preferences.Areas
	if event.National || areas.Country {
		return true
	}
	for _, district := range event.Districts {
		if slices.Contains(areas.Districts, district) || slices.Contains(subscriber.NearDistricts, district) {
			return true
		}
	}
	if areas.Near != nil && event.Lon != nil && event.Lat != nil {
		return DistanceKm(areas.Near.Lat, areas.Near.Lon, *event.Lat, *event.Lon) <= float64(areas.Near.RadiusKm)
	}
	return false
}

// DistanceKm is the great-circle distance between two points.
func DistanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0
	toRadians := math.Pi / 180
	dLat, dLon := (lat2-lat1)*toRadians, (lon2-lon1)*toRadians
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*toRadians)*math.Cos(lat2*toRadians)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(a)))
}
