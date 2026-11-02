//go:build integration

package gfs_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/gfs"
	"github.com/prabhavalabs/lanka-weather/internal/grib2"
	"github.com/prabhavalabs/lanka-weather/internal/grib2/grib2test"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database/dbtest"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

// fakeBucket serves GFS files and inventories like S3: byte ranges, and
// 404 for files not published.
type fakeBucket struct {
	mutex sync.Mutex
	files map[string][]byte
	asked []string
}

func (bucket *fakeBucket) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bucket.mutex.Lock()
	body, ok := bucket.files[r.URL.Path]
	bucket.asked = append(bucket.asked, r.URL.Path+" "+r.Header.Get("Range"))
	bucket.mutex.Unlock()
	if !ok {
		http.Error(w, "<Error><Code>NoSuchKey</Code></Error>", http.StatusNotFound)
		return
	}
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(body))
}

// publish adds one hour of a run: a 1° field over the region with u = 5 m/s
// and v = -2.5 m/s, between other messages, and its inventory.
func (bucket *fakeBucket) publish(run time.Time, hour int) {
	field := grib2.Grid{Nx: 30, Ny: 26, Lon0: 66, Lat0: 21, DLon: 1, DLat: -1}
	values := func(value float32) []float32 {
		out := make([]float32, field.Nx*field.Ny)
		for index := range out {
			out[index] = value
		}
		return out
	}
	u := grib2test.Encode(grib2test.Message{Grid: field, Reference: run, Hour: hour, Category: 2, Number: 2, Values: values(5)})
	v := grib2test.Encode(grib2test.Message{Grid: field, Reference: run, Hour: hour, Category: 2, Number: 3, Values: values(-2.5)})
	before := bytes.Repeat([]byte{0xAA}, 100)
	file := append(append(append(append([]byte{}, before...), u...), v...), bytes.Repeat([]byte{0xBB}, 50)...)
	date := run.Format("2006010215")
	index := fmt.Sprintf("1:0:d=%s:PRMSL:mean sea level:anl:\n2:%d:d=%s:UGRD:10 m above ground:anl:\n3:%d:d=%s:VGRD:10 m above ground:anl:\n4:%d:d=%s:ICEG:10 m above mean sea level:anl:\n",
		date, len(before), date, len(before)+len(u), date, len(before)+len(u)+len(v), date)
	path := gfs.FileURL("", run, hour)
	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()
	bucket.files[path] = file
	bucket.files[path+".idx"] = []byte(index)
}

func TestCollectStoresTheNewestCompleteRun(t *testing.T) {
	pool := dbtest.Open(t, "wind_fields", "source_runs")
	ctx := context.Background()
	bucket := &fakeBucket{files: map[string][]byte{}}
	server := httptest.NewServer(bucket)
	t.Cleanup(server.Close)

	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	complete := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	for _, hour := range gfs.Hours() {
		bucket.publish(complete, hour)
	}
	// The 06 UTC run has only its first hours: it is not used yet.
	partial := time.Date(2026, 11, 2, 6, 0, 0, 0, time.UTC)
	bucket.publish(partial, 0)

	collector := &gfs.Collector{Pool: pool, Client: source.NewClient("test"), BaseURL: server.URL}
	stored, err := collector.Collect(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if stored != len(gfs.Hours()) {
		t.Fatalf("stored %d hours, want %d", stored, len(gfs.Hours()))
	}
	// A second run finds the complete run stored and fetches nothing.
	asked := len(bucket.asked)
	if stored, err := collector.Collect(ctx, now.Add(30*time.Minute)); err != nil || stored != 0 {
		t.Fatalf("second collect stored %d, %v", stored, err)
	}
	if extra := len(bucket.asked) - asked; extra != 1 {
		t.Fatalf("second collect made %d requests, want only the partial run's last inventory", extra)
	}

	service := &weather.Service{Pool: pool}
	plain, gzipped, err := service.WindJSON(ctx, time.Date(2026, 11, 2, 13, 10, 0, 0, time.UTC))
	if err != nil || len(gzipped) == 0 {
		t.Fatalf("wind = %v", err)
	}
	var field weather.WindField
	if err := json.Unmarshal(plain, &field); err != nil {
		t.Fatal(err)
	}
	if field.ForecastHour != 12 || !field.RunAt.Equal(complete) || field.StepHours != 3 ||
		!field.LastValidAt.Equal(complete.Add(72*time.Hour)) {
		t.Fatalf("field = %+v", field)
	}
	if field.Grid != (weather.WindGrid{Nx: 23, Ny: 21, Lon0: 70, Lat0: 18, DLon: 1, DLat: -1}) || field.Scale != 0.1 {
		t.Fatalf("grid = %+v", field.Grid)
	}
	u, _ := base64.StdEncoding.DecodeString(field.U)
	v, _ := base64.StdEncoding.DecodeString(field.V)
	if values := gfs.Dequantise(u); len(values) != 23*21 || values[0] != 5 {
		t.Fatalf("u = %v…", values[:3])
	}
	if values := gfs.Dequantise(v); values[100] != -2.5 {
		t.Fatalf("v = %v", values[100])
	}
	if _, _, err := service.WindJSON(ctx, complete.Add(80*time.Hour)); err != weather.ErrNoWind {
		t.Fatalf("beyond the run = %v", err)
	}

	// Pruning keeps the newest complete run however old it is.
	if err := collector.Prune(ctx, now.Add(72*time.Hour)); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM wind_fields`).Scan(&count); err != nil || count != len(gfs.Hours()) {
		t.Fatalf("after pruning %d fields, %v", count, err)
	}
}
