// Command riverseed writes or removes a pretend flood in a local database, to
// check how floods are drawn when none is happening. See package riverseed.
//
// Usage:
//
//	LANKAWEATHER_ALLOW_SEED=true riverseed apply
//	LANKAWEATHER_ALLOW_SEED=true riverseed clear
package main

import (
	"context"
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
	if len(args) != 1 || (args[0] != "apply" && args[0] != "clear") {
		return fmt.Errorf("usage: riverseed apply|clear")
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
	fmt.Println("seed", args[0], "done; the API reports seeded while it is in place")
	return nil
}
