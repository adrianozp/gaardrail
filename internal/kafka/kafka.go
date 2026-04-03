package kafka

import "github.com/IBM/sarama"

type Client struct {
	client   sarama.Client
	consumer sarama.PartitionConsumer
	producer sarama.SyncProducer
	manager  sarama.OffsetManager
	pom      sarama.PartitionOffsetManager
	topic    string
}

type Config struct {
	Brokers   []string
	Topic     string
	Partition int32
	GroupID   string
}

func New(cfg Config) (*Client, error) {
	saramaCfg := sarama.NewConfig()
	saramaCfg.Consumer.Return.Errors = true
	saramaCfg.Consumer.Offsets.AutoCommit.Enable = false
	saramaCfg.Producer.Return.Successes = true

	client, err := sarama.NewClient(cfg.Brokers, saramaCfg)
	if err != nil {
		return nil, err
	}

	if err := ensureTopic(client, cfg.Topic, cfg.Partition+1); err != nil {
		client.Close()
		return nil, err
	}

	if err := client.RefreshMetadata(cfg.Topic); err != nil {
		client.Close()
		return nil, err
	}

	consumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		client.Close()
		return nil, err
	}

	manager, err := sarama.NewOffsetManagerFromClient(cfg.GroupID, client)
	if err != nil {
		consumer.Close()
		client.Close()
		return nil, err
	}

	pom, err := manager.ManagePartition(cfg.Topic, cfg.Partition)
	if err != nil {
		manager.Close()
		consumer.Close()
		client.Close()
		return nil, err
	}

	nextOffset, _ := pom.NextOffset()
	if nextOffset == sarama.OffsetNewest {
		nextOffset = sarama.OffsetOldest
	}

	pc, err := consumer.ConsumePartition(cfg.Topic, cfg.Partition, nextOffset)
	if err != nil {
		pom.Close()
		manager.Close()
		consumer.Close()
		client.Close()
		return nil, err
	}

	producer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		pc.Close()
		pom.Close()
		manager.Close()
		consumer.Close()
		client.Close()
		return nil, err
	}

	return &Client{
		client:   client,
		consumer: pc,
		producer: producer,
		manager:  manager,
		pom:      pom,
		topic:    cfg.Topic,
	}, nil
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
	defer admin.Close()

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
	newest := c.consumer.HighWaterMarkOffset()
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
	c.producer.Close()
	c.pom.Close()
	c.manager.Close()
	c.consumer.Close()
	return c.client.Close()
}
