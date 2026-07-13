package stream

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/connectors"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Consumer runs until context cancellation.
type Consumer interface {
	Run(ctx context.Context) error
}

type eventHandler func(Event)

func startMQTTConsumer(ctx context.Context, sub Subscription, onEvent eventHandler, onConnected func(bool), onError func(error)) error {
	prof, err := connectors.Get(sub.ConnectorID)
	if err != nil {
		return err
	}
	if prof.Type != connectors.TypeMQTT {
		return fmt.Errorf("connector %q is type %s, expected mqtt", sub.ConnectorID, prof.Type)
	}
	broker := strings.TrimSpace(prof.Config["broker_url"])
	if broker == "" {
		return fmt.Errorf("mqtt connector missing broker_url")
	}
	clientID := strings.TrimSpace(prof.Config["client_id"])
	if clientID == "" {
		clientID = "neural-junkie-" + sub.ID
	}
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetOrderMatters(false)
	if user := strings.TrimSpace(prof.Config["username"]); user != "" {
		opts.SetUsername(user)
	}
	if prof.Secret != "" {
		opts.SetPassword(prof.Secret)
	}
	if strings.EqualFold(prof.Config["tls"], "true") || strings.EqualFold(prof.Config["tls"], "1") {
		opts.SetTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12})
	}
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		if onConnected != nil {
			onConnected(false)
		}
		if onError != nil && err != nil {
			onError(err)
		}
	})
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		if onConnected != nil {
			onConnected(true)
		}
		token := c.Subscribe(sub.Topic, 1, func(_ mqtt.Client, msg mqtt.Message) {
			onEvent(Event{
				Protocol:       ProtocolMQTT,
				Topic:          msg.Topic(),
				Payload:        string(msg.Payload()),
				ReceivedAt:     time.Now().UTC(),
				SubscriptionID: sub.ID,
			})
		})
		token.Wait()
		if err := token.Error(); err != nil {
			if onError != nil {
				onError(fmt.Errorf("mqtt subscribe: %w", err))
			}
		}
	})

	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()
	if err := token.Error(); err != nil {
		return fmt.Errorf("mqtt connect: %w", err)
	}
	log.Printf("[stream] mqtt subscribed %s topic=%s", sub.ID, sub.Topic)

	<-ctx.Done()
	client.Disconnect(250)
	if onConnected != nil {
		onConnected(false)
	}
	return ctx.Err()
}
