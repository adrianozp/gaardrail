package kafka

import (
	"github.com/IBM/sarama"
	"github.com/adrianozp/gaardrail/pkg/config"
)

type Client struct {
	client    sarama.Client
	base      sarama.Consumer
	consumer  sarama.PartitionConsumer
	producer  sarama.SyncProducer
	manager   sarama.OffsetManager
	pom       sarama.PartitionOffsetManager
	topic     string
	partition int32
}

type closer interface {
	Close() error
}

func New(cfg config.Config) (*Client, error) {
	client, err := connect(cfg.Kafka)
	if err != nil {
		return nil, err
	}
	c := &Client{client: client, topic: cfg.Kafka.Topic, partition: cfg.Kafka.Partition}
	if err := c.open(cfg.Kafka); err != nil {
		_ = c.Close()
		return nil, err
	}
	return c, nil
}

func newSaramaConfig() *sarama.Config {
	saramaCfg := sarama.NewConfig()
	saramaCfg.Consumer.Return.Errors = true
	saramaCfg.Consumer.Offsets.AutoCommit.Enable = false
	saramaCfg.Producer.Return.Successes = true
	return saramaCfg
}

func connect(k config.Kafka) (sarama.Client, error) {
	client, err := sarama.NewClient(k.Brokers, newSaramaConfig())
	if err != nil {
		return nil, err
	}
	if err := prepareTopic(client, k); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

func prepareTopic(client sarama.Client, k config.Kafka) error {
	if err := ensureTopic(client, k.Topic, k.Partition+1); err != nil {
		return err
	}
	return client.RefreshMetadata(k.Topic)
}

func (c *Client) open(k config.Kafka) error {
	steps := []func(config.Kafka) error{c.openConsumer, c.openOffsets, c.openPartition, c.openProducer}
	for _, step := range steps {
		if err := step(k); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) openConsumer(config.Kafka) error {
	base, err := sarama.NewConsumerFromClient(c.client)
	c.base = base
	return err
}

func (c *Client) openOffsets(k config.Kafka) error {
	manager, err := sarama.NewOffsetManagerFromClient(k.GroupID, c.client)
	if err != nil {
		return err
	}
	c.manager = manager
	pom, err := manager.ManagePartition(k.Topic, k.Partition)
	c.pom = pom
	return err
}

func (c *Client) openPartition(k config.Kafka) error {
	nextOffset, _ := c.pom.NextOffset()
	if nextOffset == sarama.OffsetNewest {
		nextOffset = sarama.OffsetOldest
	}
	pc, err := c.base.ConsumePartition(k.Topic, k.Partition, nextOffset)
	c.consumer = pc
	return err
}

func (c *Client) openProducer(config.Kafka) error {
	producer, err := sarama.NewSyncProducerFromClient(c.client)
	c.producer = producer
	return err
}

func ensureTopic(client sarama.Client, topic string, numPartitions int32) error {
	brokers := make([]string, 0, len(client.Brokers()))
	for _, b := range client.Brokers() {
		brokers = append(brokers, b.Addr())
	}
	admin, err := sarama.NewClusterAdmin(brokers, client.Config())
	if err != nil {
		return err
	}
	defer func() { _ = admin.Close() }()

	topics, err := admin.ListTopics()
	if err != nil {
		return err
	}
	if _, exists := topics[topic]; exists {
		return nil
	}

	return admin.CreateTopic(topic, &sarama.TopicDetail{
		NumPartitions:     numPartitions,
		ReplicationFactor: 1,
	}, false)
}

func (c *Client) Messages() <-chan *sarama.ConsumerMessage {
	return c.consumer.Messages()
}

func (c *Client) Errors() <-chan *sarama.ConsumerError {
	return c.consumer.Errors()
}

func (c *Client) MarkOffset(msg *sarama.ConsumerMessage) {
	c.pom.MarkOffset(msg.Offset+1, "")
	c.manager.Commit()
}

func (c *Client) Size() (int64, error) {
	newest, err := c.client.GetOffset(c.topic, c.partition, sarama.OffsetNewest)
	if err != nil {
		return 0, err
	}
	offset, _ := c.pom.NextOffset()
	lag := newest - offset
	if lag < 0 {
		return 0, nil
	}
	return lag, nil
}

func (c *Client) Publish(key string, body []byte) error {
	_, _, err := c.producer.SendMessage(&sarama.ProducerMessage{
		Topic: c.topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(body),
	})
	return err
}

func (c *Client) Close() error {
	for _, part := range []closer{c.producer, c.consumer, c.pom, c.manager, c.base} {
		closeIfOpen(part)
	}
	return c.client.Close()
}

func closeIfOpen(part closer) {
	if part == nil {
		return
	}
	_ = part.Close()
}
