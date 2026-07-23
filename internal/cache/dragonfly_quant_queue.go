package cache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const quantBacktestStream = "quant:backtests"

type DragonflyQuantQueue struct {
	client *redis.Client
	prefix string
}

func NewDragonflyQuantQueue(redisURL, prefix string) (*DragonflyQuantQueue, error) {
	options, err := redis.ParseURL(strings.TrimSpace(redisURL))
	if err != nil {
		return nil, fmt.Errorf("parse Dragonfly URL: %w", err)
	}
	return &DragonflyQuantQueue{client: redis.NewClient(options), prefix: strings.Trim(strings.TrimSpace(prefix), ":")}, nil
}

func (q *DragonflyQuantQueue) Ping(ctx context.Context) error {
	if q == nil || q.client == nil {
		return fmt.Errorf("quant queue is unavailable")
	}
	return q.client.Ping(ctx).Err()
}

func (q *DragonflyQuantQueue) Close() error {
	if q == nil || q.client == nil {
		return nil
	}
	return q.client.Close()
}

func (q *DragonflyQuantQueue) EnqueueBacktest(ctx context.Context, jobID string) error {
	if q == nil || q.client == nil {
		return fmt.Errorf("quant queue is unavailable")
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return fmt.Errorf("job id is required")
	}
	return q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.key(quantBacktestStream),
		Values: map[string]interface{}{"job_id": jobID, "queued_at": time.Now().UTC().Format(time.RFC3339Nano)},
	}).Err()
}

type QuantQueueMessage struct {
	ID        string
	JobID     string
	Recovered bool
}

func (q *DragonflyQuantQueue) ClaimStaleBacktests(ctx context.Context, group, consumer string, minIdle time.Duration, count int64) ([]QuantQueueMessage, error) {
	if q == nil || q.client == nil {
		return nil, fmt.Errorf("quant queue is unavailable")
	}
	if count <= 0 {
		count = 1
	}
	messages, _, err := q.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   q.key(quantBacktestStream),
		Group:    strings.TrimSpace(group),
		Consumer: strings.TrimSpace(consumer),
		MinIdle:  minIdle,
		Start:    "0-0",
		Count:    count,
	}).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]QuantQueueMessage, 0, len(messages))
	for _, message := range messages {
		jobID, _ := message.Values["job_id"].(string)
		if strings.TrimSpace(jobID) != "" {
			result = append(result, QuantQueueMessage{ID: message.ID, JobID: jobID, Recovered: true})
		}
	}
	return result, nil
}

func (q *DragonflyQuantQueue) EnsureConsumerGroup(ctx context.Context, group string) error {
	if q == nil || q.client == nil {
		return fmt.Errorf("quant queue is unavailable")
	}
	err := q.client.XGroupCreateMkStream(ctx, q.key(quantBacktestStream), strings.TrimSpace(group), "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func (q *DragonflyQuantQueue) ReadBacktests(ctx context.Context, group, consumer string, count int64, block time.Duration) ([]QuantQueueMessage, error) {
	if q == nil || q.client == nil {
		return nil, fmt.Errorf("quant queue is unavailable")
	}
	if count <= 0 {
		count = 1
	}
	streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    strings.TrimSpace(group),
		Consumer: strings.TrimSpace(consumer),
		Streams:  []string{q.key(quantBacktestStream), ">"},
		Count:    count,
		Block:    block,
	}).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]QuantQueueMessage, 0, len(streams))
	for _, stream := range streams {
		for _, message := range stream.Messages {
			jobID, _ := message.Values["job_id"].(string)
			if strings.TrimSpace(jobID) != "" {
				result = append(result, QuantQueueMessage{ID: message.ID, JobID: jobID})
			}
		}
	}
	return result, nil
}

func (q *DragonflyQuantQueue) AckBacktest(ctx context.Context, group, messageID string) error {
	if q == nil || q.client == nil {
		return fmt.Errorf("quant queue is unavailable")
	}
	return q.client.XAck(ctx, q.key(quantBacktestStream), strings.TrimSpace(group), strings.TrimSpace(messageID)).Err()
}

func (q *DragonflyQuantQueue) key(suffix string) string {
	if q.prefix == "" {
		return suffix
	}
	return q.prefix + ":" + suffix
}
