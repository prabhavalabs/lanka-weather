package httpapi

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/prabhavalabs/lanka-weather/internal/floodrisk"
)

// FloodRisk serves the combined flood-risk estimate by district.
type FloodRisk interface {
	Estimate(ctx context.Context) (*floodrisk.Document, error)
}

var districtID = regexp.MustCompile(`^LK-\d{2}$`)

// floodRisk returns every district's estimate, or one with ?district=LK-11:
// GET /v1/flood-risk. A district nothing is known about is at level 0 with
// no inputs, which is not a statement that it is safe.
func (api *api) floodRisk(w http.ResponseWriter, r *http.Request) {
	if api.deps.FloodRisk == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "the flood-risk estimate is not configured")
		return
	}
	district := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("district")))
	if district != "" && !districtID.MatchString(district) {
		writeError(w, http.StatusBadRequest, "invalid_district", "district must be a code such as LK-11")
		return
	}
	document, err := api.deps.FloodRisk.Estimate(r.Context())
	if err != nil {
		api.internalError(w, "flood risk", err)
		return
	}
	if district != "" {
		limited := *document
		limited.Districts = []floodrisk.District{}
		for _, entry := range document.Districts {
			if entry.District == district {
				limited.Districts = append(limited.Districts, entry)
			}
		}
		if len(limited.Districts) == 0 {
			limited.Districts = append(limited.Districts, floodrisk.District{
				Assessment: floodrisk.Assess(floodrisk.Inputs{District: district})})
		}
		document = &limited
	}
	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, http.StatusOK, Envelope{Data: document})
}
