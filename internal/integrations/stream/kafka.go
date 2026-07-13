package stream

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/connectors"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

func startKafkaConsumer(ctx context.Context, sub Subscription, onEvent eventHandler, onConnected func(bool), onError func(error)) error {
	prof, err := connectors.Get(sub.ConnectorID)
	if err != nil {
		return err
	}
	if prof.Type != connectors.TypeKafka {
		return fmt.Errorf("connector %q is type %s, expected kafka", sub.ConnectorID, prof.Type)
	}
	brokersRaw := strings.TrimSpace(prof.Config["brokers"])
	if brokersRaw == "" {
		return fmt.Errorf("kafka connector missing brokers")
	}
	brokers := splitCSV(brokersRaw)
	groupID := strings.TrimSpace(prof.Config["group_id"])
	if groupID == "" {
		groupID = "neural-junkie-" + sub.ID
	}

	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}
	username := strings.TrimSpace(prof.Config["username"])
	if username != "" && prof.Secret != "" {
		dialer.SASLMechanism = plain.Mechanism{
			Username: username,
			Password: prof.Secret,
		}
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		GroupID:        groupID,
		Topic:          sub.Topic,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		Dialer:         dialer,
	})
	defer reader.Close()

	if onConnected != nil {
		onConnected(true)
	}
	log.Printf("[stream] kafka consuming %s topic=%s group=%s", sub.ID, sub.Topic, groupID)

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				if onConnected != nil {
					onConnected(false)
				}
				return ctx.Err()
			}
			if onError != nil {
				onError(err)
			}
			time.Sleep(2 * time.Second)
			continue
		}
		onEvent(Event{
			Protocol:       ProtocolKafka,
			Topic:          msg.Topic,
			Key:            string(msg.Key),
			Payload:        string(msg.Value),
			ReceivedAt:     time.Now().UTC(),
			SubscriptionID: sub.ID,
		})
		if err := reader.CommitMessages(ctx, msg); err != nil {
			if onError != nil {
				onError(fmt.Errorf("kafka commit: %w", err))
			}
		}
	}
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
