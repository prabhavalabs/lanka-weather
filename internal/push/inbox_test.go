package push

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/push/messages"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

func TestInboxGroupsDeliveriesIntoNotices(t *testing.T) {
	sent := time.Date(2026, 11, 12, 13, 0, 0, 0, time.UTC)
	row := func(notice int64, key, district, station string) store.PushInboxRow {
		facts, _ := json.Marshal(map[string]string{"district": district, "station": station, "river": "Kalu Ganga", "source": "irrigation",
			"issuedAt": "2026-11-12T12:45:00Z"})
		return store.PushInboxRow{NoticeID: notice, SentAt: sent, NoticeSeverity: "warning", DedupeKey: key, Kind: messages.KindRiver,
			Variant: "minor_flood", Topic: "river", Severity: "warning", Subject: "river:" + station, Districts: []string{district},
			Facts: facts, Url: "/?station=" + station, Tag: "river:station:" + station, DetectedAt: sent, ExpiresAt: sent.Add(6 * time.Hour)}
	}
	names := DistrictNames{"LK-91": {"Ratnapura", "රත්නපුර", "இரத்தினபுரி"}, "LK-13": {"Kalutara", "කළුතර", "களுத்துறை"}}
	notices, err := InboxFrom([]store.PushInboxRow{row(9, "a", "LK-91", "Ratnapura"), row(9, "b", "LK-13", "Ellagawa"), row(8, "c", "LK-91", "Ratnapura")}, "si", names)
	if err != nil {
		t.Fatal(err)
	}
	if len(notices) != 2 || notices[0].ID != "9" || len(notices[0].Events) != 2 || len(notices[1].Events) != 1 {
		t.Fatalf("notices %+v", notices)
	}
	first := notices[0]
	if len(first.Areas) != 2 || first.Areas[0].Name != "රත්නපුර" || first.Areas[1].Name != "කළුතර" || !first.Official ||
		first.Source != "වාරිමාර්ග දෙපාර්තමේන්තුව" || first.National || first.URL != "/" {
		t.Fatalf("first %+v", first)
	}
	event := first.Events[0]
	if event.IssuedAt == nil || !event.IssuedAt.Equal(time.Date(2026, 11, 12, 12, 45, 0, 0, time.UTC)) || event.URL == "" || len(event.Areas) != 1 {
		t.Fatalf("event %+v", event)
	}
}
