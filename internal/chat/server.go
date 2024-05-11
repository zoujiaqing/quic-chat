package chat

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
)

type server struct {
	listener *quic.Listener
	clients  map[string]quic.Connection
	messages chan Message
	mutex    sync.Mutex
}

func NewServer() (*server, error) {
	tlsConf := generateTLSConfig()

	listener, err := quic.ListenAddr(fmt.Sprintf(":%d", port), tlsConf, &quic.Config{
		KeepAlivePeriod: 10 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &server{
		listener: listener,
		clients:  map[string]quic.Connection{},
		messages: make(chan Message),
	}, nil
}

func (s *server) Close() error {
	close(s.messages)
	return s.listener.Close()
}

func (s *server) Broadcast(ctx context.Context) {
	for {
		select {
		case message := <-s.messages:
			s.mutex.Lock()
			for addr, client := range s.clients {
				go s.sendMessage(client, addr, message)
			}
			s.mutex.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

func (s *server) Accept(ctx context.Context) {
	for {
		conn, err := s.listener.Accept(ctx)
		if err != nil {
			log.Printf("[ERROR] failed to accept new connection: %v\n", err)
			return
		}

		go s.handleConn(ctx, conn)
	}
}

func (s *server) handleConn(ctx context.Context, conn quic.Connection) {
	defer func() { _ = conn.CloseWithError(serverError, "failed to handle connection") }()

	s.mutex.Lock()
	s.clients[conn.RemoteAddr().String()] = conn
	s.mutex.Unlock()

	log.Printf("[INFO] added client: %s\n", conn.RemoteAddr().String())

	for {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			s.removeClient(conn.RemoteAddr().String())
			return
		}

		go s.readMessage(stream)
	}
}

func (s *server) removeClient(addr string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, ok := s.clients[addr]; ok {
		delete(s.clients, addr)
		log.Printf("[INFO] removed client %s\n", addr)
	}
}

func (s *server) readMessage(stream quic.Stream) {
	defer func() { _ = stream.Close() }()

	var message Message
	if err := message.Read(stream); err != nil {
		log.Printf("[ERROR] failed to decode message: %v\n", err)
		return
	}

	s.messages <- message
}

func (s *server) sendMessage(client quic.Connection, addr string, message Message) {
	stream, err := client.OpenStream()
	if err != nil {
		log.Printf("[ERROR] failed to connect to client %s: %v\n", addr, err)
		return
	}
	defer func() { _ = stream.Close() }()

	if err := message.Write(stream); err != nil {
		log.Printf("[ERROR] failed to send message to %s: %v\n", addr, err)
		return
	}
}

func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-chat-example"},
	}
}
