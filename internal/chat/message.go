package chat

import (
	"encoding/gob"
	"io"
)

type Message struct {
	Nickname string
	Text     string
}

func (m *Message) Read(r io.Reader) error {
	return gob.NewDecoder(r).Decode(m)
}

func (m *Message) Write(w io.Writer) error {
	return gob.NewEncoder(w).Encode(m)
}
