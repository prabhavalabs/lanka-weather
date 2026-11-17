package push

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// Delivery limits.
const (
	defaultWorkers    = 4
	defaultRetries    = 2
	defaultRetryDelay = 2 * time.Second
	// maxBackoff bounds the wait after repeated failures.
	maxBackoff = time.Hour
	// deadAfterFailures and deadAfter drop a subscription that has failed this
	// many times in a row with no success for this long.
	deadAfterFailures = 20
	deadAfter         = 7 * 24 * time.Hour
)

// Dispatcher sends pending events to the subscribers who want them. Only the
// worker holding the scheduler's lock runs it.
type Dispatcher struct {
	Pool   *pgxpool.Pool
	Sender Sender
	Logger *slog.Logger
	// Workers is how many notices are sent at once.
	Workers int
	// HourlyCap is HourlyCap unless set.
	HourlyCap int
	// Retries and RetryDelay bound how often one notice is retried in a run
	// after a server error or a network failure.
	Retries    int
	RetryDelay time.Duration
	Now        func() time.Time

	districts districtCache
}

// Stats count a run. Endpoints are never logged.
type Stats struct {
	Events      int `json:"events"`
	Subscribers int `json:"subscribers"`
	Notices     int `json:"notices"`
	Sent        int `json:"sent"`
	Failed      int `json:"failed"`
	Gone        int `json:"gone"`
	Held        int `json:"held"`
	BackingOff  int `json:"backingOff"`
	Dropped     int `json:"dropped"`
}

type job struct {
	subscriber Subscriber
	failures   int32
	notice     Notice
}

func (dispatcher *Dispatcher) now() time.Time {
	if dispatcher.Now != nil {
		return dispatcher.Now().UTC()
	}
	return time.Now().UTC()
}

// Run sends what is pending now.
func (dispatcher *Dispatcher) Run(ctx context.Context) (Stats, error) {
	now := dispatcher.now()
	queries := store.New(dispatcher.Pool)
	stats := Stats{}
	rows, err := queries.ListActivePushEvents(ctx, now)
	if err != nil {
		return stats, fmt.Errorf("list events: %w", err)
	}
	stats.Events = len(rows)
	if len(rows) == 0 {
		return stats, nil
	}
	events := make([]Event, 0, len(rows))
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		event, err := EventFrom(row)
		if err != nil {
			return stats, err
		}
		events = append(events, event)
		keys = append(keys, event.Key)
	}
	subscriptions, err := queries.ListPushSubscriptions(ctx)
	if err != nil {
		return stats, fmt.Errorf("list subscriptions: %w", err)
	}
	stats.Subscribers = len(subscriptions)
	deliveries, err := queries.ListPushDeliveriesFor(ctx, keys)
	if err != nil {
		return stats, fmt.Errorf("list deliveries: %w", err)
	}
	sent := make(map[int64]map[string]bool, len(deliveries))
	for _, delivery := range deliveries {
		if sent[delivery.SubscriptionID] == nil {
			sent[delivery.SubscriptionID] = map[string]bool{}
		}
		sent[delivery.SubscriptionID][delivery.EventKey] = true
	}
	counts, err := queries.RecentPushNoticeCounts(ctx, now.Add(-time.Hour))
	if err != nil {
		return stats, fmt.Errorf("count recent notices: %w", err)
	}
	recent := make(map[int64]int, len(counts))
	for _, count := range counts {
		recent[count.SubscriptionID] = int(count.Notices)
	}
	hourlyCap := dispatcher.HourlyCap
	if hourlyCap <= 0 {
		hourlyCap = HourlyCap
	}

	jobs := []job{}
	for _, row := range subscriptions {
		subscriber := SubscriberFrom(row)
		if subscriber.RetryAfter != nil && subscriber.RetryAfter.After(now) {
			stats.BackingOff++
			continue
		}
		pending := []Event{}
		for _, event := range events {
			if !sent[subscriber.ID][event.Key] && subscriber.Wants(event) {
				pending = append(pending, event)
			}
		}
		if len(pending) == 0 {
			continue
		}
		send, held := Plan(pending, recent[subscriber.ID], hourlyCap)
		stats.Held += len(held)
		for _, notice := range send {
			jobs = append(jobs, job{subscriber: subscriber, failures: row.ConsecutiveFailures, notice: notice})
		}
	}
	stats.Notices = len(jobs)
	if len(jobs) > 0 {
		names, err := dispatcher.districts.load(ctx, queries, now)
		if err != nil {
			return stats, err
		}
		dispatcher.deliverAll(ctx, jobs, names, now, &stats)
	}
	before := now.Add(-deadAfter)
	dropped, err := queries.DeleteFailingPushSubscriptions(ctx, store.DeleteFailingPushSubscriptionsParams{MaxFailures: deadAfterFailures, Before: before})
	if err != nil {
		return stats, fmt.Errorf("drop failing subscriptions: %w", err)
	}
	stats.Dropped = int(dropped)
	return stats, nil
}

// deliverAll sends jobs with a small pool of workers.
func (dispatcher *Dispatcher) deliverAll(ctx context.Context, jobs []job, names DistrictNames, now time.Time, stats *Stats) {
	workers := dispatcher.Workers
	if workers <= 0 {
		workers = defaultWorkers
	}
	queue := make(chan job)
	var mu sync.Mutex
	var group sync.WaitGroup
	for range min(workers, len(jobs)) {
		group.Go(func() {
			for item := range queue {
				outcome := dispatcher.deliver(ctx, item, names, now)
				mu.Lock()
				switch outcome {
				case outcomeSent:
					stats.Sent++
				case outcomeGone:
					stats.Gone++
				case outcomeFailed:
					stats.Failed++
				}
				mu.Unlock()
			}
		})
	}
	for _, item := range jobs {
		if ctx.Err() != nil {
			break
		}
		queue <- item
	}
	close(queue)
	group.Wait()
}

type outcome int

const (
	outcomeSkipped outcome = iota
	outcomeSent
	outcomeGone
	outcomeFailed
)

// backoff is the wait after a number of failures in a row: a minute, doubling,
// up to an hour.
func backoff(failures int32) time.Duration {
	wait := time.Minute
	for range max(0, failures-1) {
		wait *= 2
		if wait >= maxBackoff {
			return maxBackoff
		}
	}
	return wait
}

// deliver claims a notice's events for a subscriber, sends it and records the
// outcome. A claim is released when the notice could not be delivered, so it
// is tried again in a later run while it is still relevant.
func (dispatcher *Dispatcher) deliver(ctx context.Context, item job, names DistrictNames, now time.Time) outcome {
	queries := store.New(dispatcher.Pool)
	logger := dispatcher.Logger
	if logger == nil {
		logger = slog.Default()
	}
	noticeID, err := queries.NextPushNoticeID(ctx)
	if err != nil {
		logger.Warn("push notice id failed", "error", err)
		return outcomeSkipped
	}
	release := func() {
		if err := queries.ReleasePushNotice(context.WithoutCancel(ctx), noticeID); err != nil {
			logger.Warn("release push notice failed", "error", err)
		}
	}
	for _, event := range item.notice.Events {
		claimed, err := queries.ClaimPushDelivery(ctx, store.ClaimPushDeliveryParams{SubscriptionID: item.subscriber.ID,
			EventKey: event.Key, NoticeID: noticeID, Severity: string(item.notice.Severity), SentAt: now})
		if err != nil || claimed == 0 {
			// Sent already, or the store failed: leave it to the next run.
			release()
			return outcomeSkipped
		}
	}
	payload, message := item.notice.Message(item.subscriber, names, now, noticeID)
	if message.Payload, err = json.Marshal(payload); err != nil {
		release()
		return outcomeFailed
	}
	failed := func(retryAfter time.Duration) outcome {
		at := now.Add(retryAfter)
		if err := queries.RecordPushFailure(context.WithoutCancel(ctx), store.RecordPushFailureParams{ID: item.subscriber.ID, At: now, RetryAfter: &at}); err != nil {
			logger.Warn("record push failure failed", "error", err)
		}
		return outcomeFailed
	}
	retries := dispatcher.Retries
	if retries <= 0 {
		retries = defaultRetries
	}
	delay := dispatcher.RetryDelay
	if delay <= 0 {
		delay = defaultRetryDelay
	}
	for attempt := 0; ; attempt++ {
		result, err := dispatcher.Sender.Send(ctx, message)
		switch {
		case err == nil && result.Accepted():
			if err := queries.RecordPushSuccess(context.WithoutCancel(ctx), store.RecordPushSuccessParams{ID: item.subscriber.ID, At: now}); err != nil {
				logger.Warn("record push success failed", "error", err)
			}
			return outcomeSent
		case err == nil && result.Gone:
			if _, err := queries.DeletePushSubscription(context.WithoutCancel(ctx), item.subscriber.ID); err != nil {
				logger.Warn("delete gone subscription failed", "error", err)
			}
			return outcomeGone
		case err == nil && result.Status == 429:
			release()
			return failed(max(result.RetryAfter, backoff(item.failures+1)))
		case err == nil && result.Status >= 400 && result.Status < 500 && result.Status != 408:
			// The push service refused this message as it is (a 400 or 413):
			// sending it again would be refused again. The claim stays, so it
			// is not retried; a 401 or 403 is our VAPID identity, which an
			// operator may fix, so those are released.
			if result.Status == 401 || result.Status == 403 {
				release()
			}
			logger.Warn("push refused", "status", result.Status, "kind", item.notice.Kind)
			return failed(backoff(item.failures + 1))
		}
		if attempt >= retries || ctx.Err() != nil {
			release()
			return failed(backoff(item.failures + 1))
		}
		select {
		case <-ctx.Done():
		case <-time.After(delay * time.Duration(attempt+1)):
		}
	}
}
