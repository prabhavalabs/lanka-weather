// Command riverseed writes or removes a pretend flood in a local database, to
// check how floods are drawn when none is happening. See package riverseed.
//
// Usage:
//
//	LANKAWEATHER_ALLOW_SEED=true riverseed apply
//	LANKAWEATHER_ALLOW_SEED=true riverseed clear
//	LANKAWEATHER_ALLOW_SEED=true riverseed polygons manifest.json
//
// The manifest lists real serialized polygons (see riverseed.Polygon). Keep it
// out of the repository: Google's terms limit how long they may be kept.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/platform/database"
	"github.com/prabhavalabs/lanka-weather/internal/rivernet"
	"github.com/prabhavalabs/lanka-weather/internal/riverseed"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	polygons := len(args) == 2 && args[0] == "polygons"
	if !polygons && (len(args) != 1 || (args[0] != "apply" && args[0] != "clear")) {
		return fmt.Errorf("usage: riverseed apply|clear|polygons manifest.json")
	}
	if os.Getenv("LANKAWEATHER_ALLOW_SEED") != "true" {
		return fmt.Errorf("riverseed writes pretend floods; set LANKAWEATHER_ALLOW_SEED=true on a local database")
	}
	ctx := context.Background()
	pool, err := database.Open(ctx, os.Getenv("LANKAWEATHER_DATABASE_URL"))
	if err != nil {
		return err
	}
	defer pool.Close()
	now := time.Now().UTC()
	if polygons {
		payload, err := os.ReadFile(args[1])
		if err != nil {
			return err
		}
		var list []riverseed.Polygon
		if err := json.Unmarshal(payload, &list); err != nil {
			return fmt.Errorf("read the manifest: %w", err)
		}
		if err := riverseed.Polygons(ctx, pool, list, now); err != nil {
			return err
		}
		fmt.Printf("%d real polygons written; the API reports seeded while they are in place\n", len(list))
		return nil
	}
	if args[0] == "apply" {
		err = riverseed.Apply(ctx, pool, now)
	} else {
		err = riverseed.Clear(ctx, pool)
	}
	if err != nil {
		return err
	}
	network := &rivernet.Collector{Pool: pool, Logger: slog.Default()}
	if _, err := network.Collect(ctx, now); err != nil {
		return err
	}
	if args[0] == "apply" {
		fmt.Println("pretend flood written; the API reports seeded while it is in place")
	} else {
		fmt.Println("pretend flood removed")
	}
	return nil
}
