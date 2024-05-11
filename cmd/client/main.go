package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zoujiaqing/quic-chat/internal/chat"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	nickname := flag.String("n", "", "nickname")
	addr := flag.String("s", "localhost", "server")
	flag.Parse()

	if *nickname == "" {
		return errors.New("nickname is empty")
	}

	client, err := chat.NewClient(*addr, *nickname)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messages, errs := client.Receive(ctx)

	a := createApp(client.Send, messages, errs)
	if err := tea.NewProgram(a).Start(); err != nil {
		return err
	}

	return nil
}
