package sqs

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/rs/zerolog/log"
)

type SQSRepository struct {
	client *sqs.Client
	cfg    config.SQS
}

func New(cfg config.Config) (*SQSRepository, error) {
	if cfg.SQS.QueueURL == "" {
		return nil, errors.New("sqs: queue_url is required")
	}
	awscfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.SQS.Region),
	)
	if err != nil {
		return nil, err
	}
	return &SQSRepository{
		client: sqs.NewFromConfig(awscfg),
		cfg:    cfg.SQS,
	}, nil
}

func (r *SQSRepository) Enqueue(m entities.Message) (string, error) {
	body, err := json.Marshal(ToModel(m))
	if err != nil {
		return "", err
	}
	out, err := r.client.SendMessage(context.Background(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(r.cfg.QueueURL),
		MessageBody: aws.String(string(body)),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(out.MessageId), nil
}

func (r *SQSRepository) Dequeue(ctx context.Context) (entities.Message, error) {
	for {
		msgs, err := r.receive(ctx)
		if err != nil {
			return entities.Message{}, err
		}
		if len(msgs) > 0 {
			return r.toEntity(msgs[0])
		}
		if ctx.Err() != nil {
			return entities.Message{}, ctx.Err()
		}
	}
}

func (r *SQSRepository) receive(ctx context.Context) ([]sqstypes.Message, error) {
	out, err := r.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(r.cfg.QueueURL),
		MaxNumberOfMessages: r.cfg.MaxMessages,
		WaitTimeSeconds:     r.cfg.WaitTimeSeconds,
		VisibilityTimeout:   r.cfg.VisibilityTimeout,
	})
	if err != nil {
		return nil, err
	}
	return out.Messages, nil
}

func (r *SQSRepository) toEntity(sqsMsg sqstypes.Message) (entities.Message, error) {
	log.Debug().Msg("sqs: msg dequeued")
	var model sqsMessage
	if err := json.Unmarshal([]byte(aws.ToString(sqsMsg.Body)), &model); err != nil {
		return entities.Message{}, err
	}
	entity := model.ToMessage()
	entity.Ack = r.deleter(sqsMsg.ReceiptHandle)
	return entity, nil
}

func (r *SQSRepository) deleter(receiptHandle *string) func() error {
	queueURL := r.cfg.QueueURL
	return func() error {
		_, err := r.client.DeleteMessage(context.Background(), &sqs.DeleteMessageInput{
			QueueUrl:      aws.String(queueURL),
			ReceiptHandle: receiptHandle,
		})
		return err
	}
}

func (r *SQSRepository) Ack(_ context.Context, m entities.Message) error {
	if m.Ack != nil {
		return m.Ack()
	}
	return nil
}

func (r *SQSRepository) Size() (int64, error) {
	out, err := r.client.GetQueueAttributes(context.Background(), &sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(r.cfg.QueueURL),
		AttributeNames: []sqstypes.QueueAttributeName{sqstypes.QueueAttributeNameApproximateNumberOfMessages},
	})
	if err != nil {
		return 0, err
	}
	val, ok := out.Attributes[string(sqstypes.QueueAttributeNameApproximateNumberOfMessages)]
	if !ok {
		return 0, nil
	}
	return strconv.ParseInt(val, 10, 64)
}
