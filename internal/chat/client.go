package chat

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/quic-go/quic-go"
)

type client struct {
	nickname string
	conn     quic.Connection
}

func NewClient(addr, nickname string) (*client, error) {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-chat-example"},
	}
	conn, err := quic.DialAddr(
		context.Background(),
		fmt.Sprintf("%s:%d", addr, port),
		tlsConf,
		nil)
	if err != nil {
		return nil, err
	}

	return &client{nickname: nickname, conn: conn}, nil
}

func (c *client) Send(text string) error {
	stream, err := c.conn.OpenStream()
	if err != nil {
		return err
	}
	defer func() { _ = stream.Close() }()

	message := Message{Nickname: c.nickname, Text: text}

	return message.Write(stream)
}

func (c *client) Receive(ctx context.Context) (<-chan Message, <-chan error) {
	messages, errs := make(chan Message), make(chan error)
	go func() {
		defer close(messages)
		defer close(errs)
		for {
			stream, err := c.conn.AcceptStream(ctx)
			if err != nil {
				errs <- err
				return
			}

			go c.readStream(stream, messages, errs)
		}
	}()

	return messages, errs
}

func (c *client) readStream(stream quic.Stream, messages chan<- Message, errs chan<- error) {
	defer func() { _ = stream.Close() }()

	var message Message
	if err := message.Read(stream); err != nil {
		errs <- err
		return
	}

	messages <- message
}
