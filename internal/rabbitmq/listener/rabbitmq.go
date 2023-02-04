package listener

import (
	"context"
	"github.com/emortalmc/proto-specs/gen/go/message/common"
	"github.com/google/uuid"
	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"player-tracker/internal/repository"
)

const (
	queueName = "player-tracker:all"

	connectType    = "emortal.message.PlayerConnectMessage"
	disconnectType = "emortal.message.PlayerDisconnectMessage"
	switchType     = "emortal.message.PlayerSwitchServerMessage"
)

type rabbitMqListener struct {
	logger *zap.SugaredLogger
	repo   repository.Repository
	chann  *amqp091.Channel
}

func NewRabbitMQListener(logger *zap.SugaredLogger, repo repository.Repository, conn *amqp091.Connection) error {
	channel, err := conn.Channel()
	if err != nil {
		return err
	}

	msgChan, err := channel.Consume(queueName, "", false, false, false, false, amqp091.Table{})
	if err != nil {
		return err
	}

	listener := rabbitMqListener{
		logger: logger,
		repo:   repo,
		chann:  channel,
	}

	logger.Infow("listening for messages", "queue", queueName)
	// Run as goroutine as it is blocking
	go listener.listen(msgChan)

	return nil
}

func (l *rabbitMqListener) listen(msgChan <-chan amqp091.Delivery) {
	for d := range msgChan {
		success := true

		switch d.Type {
		case connectType:
			msg := &common.PlayerConnectMessage{}
			err := proto.Unmarshal(d.Body, msg)
			if err != nil {
				l.logger.Errorw("error unmarshaling PlayerConnectMessage", err)
			}

			err = l.handlePlayerConnect(msg)
			if err != nil {
				success = false
			}
		case disconnectType:
			msg := &common.PlayerDisconnectMessage{}

			err := proto.Unmarshal(d.Body, msg)
			if err != nil {
				l.logger.Errorw("error unmarshaling PlayerDisconnectMessage", err)
			}

			err = l.handlePlayerDisconnect(msg)
			if err != nil {
				success = false
			}
		case switchType:
			msg := &common.PlayerSwitchServerMessage{}

			err := proto.Unmarshal(d.Body, msg)
			if err != nil {
				l.logger.Errorw("error unmarshaling PlayerSwitchServerMessage", err)
			}

			err = l.handlePlayerSwitch(msg)
			if err != nil {
				success = false
			}
		default:
			l.logger.Errorw("unknown message type", d.Type)
		}

		if success {
			err := d.Ack(false)
			if err != nil {
				l.logger.Errorw("error acknowledging message", err)
			}
		}
	}
}

func (l *rabbitMqListener) handlePlayerConnect(msg *common.PlayerConnectMessage) error {
	pId, err := uuid.Parse(msg.PlayerId)
	if err != nil {
		return err
	}

	err = l.repo.SetPlayerProxy(context.TODO(), pId, msg.ServerId)
	if err != nil {
		return err
	}
	return nil
}

func (l *rabbitMqListener) handlePlayerDisconnect(msg *common.PlayerDisconnectMessage) error {
	pId, err := uuid.Parse(msg.PlayerId)
	if err != nil {
		return err
	}

	err = l.repo.DeletePlayer(context.TODO(), pId)
	if err != nil {
		return err
	}
	return nil
}

func (l *rabbitMqListener) handlePlayerSwitch(msg *common.PlayerSwitchServerMessage) error {
	pId, err := uuid.Parse(msg.PlayerId)
	if err != nil {
		return err
	}

	err = l.repo.SetPlayerGameServer(context.TODO(), pId, msg.ServerId)
	if err != nil {
		return err
	}
	return nil
}
