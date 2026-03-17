package confluent

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	kafka "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"

	"github.com/papercomputeco/sweeper/pkg/telemetry"
)

const defaultPublishTimeout = 5 * time.Second

var (
	errMissingBrokers = errors.New("confluent: brokers are required")
	errMissingTopic   = errors.New("confluent: topic is required")
)

type message = kafka.Message

type writer interface {
	WriteMessages(ctx context.Context, msgs ...message) error
	Close() error
}

// Config configures a Confluent Cloud Kafka publisher.
type Config struct {
	Brokers        []string
	Topic          string
	ClientID       string
	APIKeyEnv      string // env var name holding the API key
	APISecretEnv   string // env var name holding the API secret
	PublishTimeout time.Duration
}

// Publisher publishes telemetry events to Confluent Cloud via Kafka.
type Publisher struct {
	writer         writer
	publishTimeout time.Duration
}

var _ telemetry.Publisher = (*Publisher)(nil)

// NewPublisher creates a Confluent publisher with SASL/TLS for Confluent Cloud.
func NewPublisher(c Config) (*Publisher, error) {
	if len(c.Brokers) == 0 {
		return nil, errMissingBrokers
	}
	if c.Topic == "" {
		return nil, errMissingTopic
	}

	apiKey := os.Getenv(c.APIKeyEnv)
	apiSecret := os.Getenv(c.APISecretEnv)

	transport := &kafka.Transport{
		TLS: &tls.Config{MinVersion: tls.VersionTLS12},
	}
	if apiKey != "" && apiSecret != "" {
		transport.SASL = plain.Mechanism{
			Username: apiKey,
			Password: apiSecret,
		}
	}
	if c.ClientID != "" {
		transport.ClientID = c.ClientID
	}

	kw := &kafka.Writer{
		Addr:      kafka.TCP(c.Brokers...),
		Topic:     c.Topic,
		Balancer:  &kafka.Hash{},
		Transport: transport,
	}

	return newPublisherWithWriter(c, kw)
}

func newPublisherWithWriter(c Config, w writer) (*Publisher, error) {
	if w == nil {
		return nil, errors.New("confluent: writer must not be nil")
	}
	timeout := c.PublishTimeout
	if timeout <= 0 {
		timeout = defaultPublishTimeout
	}
	return &Publisher{writer: w, publishTimeout: timeout}, nil
}

func (p *Publisher) Publish(ctx context.Context, event telemetry.Event) error {
	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("confluent: marshal event: %w", err)
	}

	publishCtx, cancel := context.WithTimeout(ctx, p.publishTimeout)
	defer cancel()

	key := []byte(event.Type + ":" + event.Timestamp.Format(time.RFC3339Nano))
	return p.writer.WriteMessages(publishCtx, message{
		Key:   key,
		Value: value,
		Time:  event.Timestamp,
	})
}

func (p *Publisher) Close() error {
	return p.writer.Close()
}
