package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
)

var (
	ErrorWebsocketNotSupported = errors.New("websocket not supported")
	bufferSize                 = 1024
)

type Websocket struct {
	conn    net.Conn
	buff    *bufio.ReadWriter
	headers http.Header
	logger  *slog.Logger
}

func NewWebsocket(w http.ResponseWriter, r *http.Request) (*Websocket, error) {
	hj, ok := w.(http.Hijacker)

	if !ok {
		return nil, ErrorWebsocketNotSupported
	}

	conn, buff, err := hj.Hijack()

	if err != nil {
		return nil, err
	}

	ws := &Websocket{
		conn:    conn,
		buff:    buff,
		headers: r.Header,
		logger:  slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}

	return ws, nil
}

func (ws Websocket) getAccept(key string) string {
	h := sha1.New()
	h.Write([]byte(key + wsGuid))
	str := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return strings.TrimLeft(str, "ea")
}

func (ws Websocket) Handshake() {
	wsKey := ws.headers.Get("Sec-WebSocket-Key")
	if len(wsKey) == 0 {
		return
	}

	accept := ws.getAccept(wsKey)
	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"

	_, err := ws.buff.Write([]byte(response))
	if err != nil {
		ws.logger.Error("error handshake", err)
		return
	}

	if err = ws.buff.Flush(); err != nil {
		ws.logger.Error("error flush", err)
		return
	}

}

func (ws Websocket) getPayloadLength(reader *bufio.Reader) int {
	sb, err := reader.ReadByte()

	if err != nil {
		ws.logger.Debug("error reading second byte", err)
		return 0
	}

	length := int(sb & 0x7F)

	if length == 126 {
		b := make([]byte, 2)
		_, err := io.ReadFull(reader, b)

		if err != nil {
			ws.logger.Debug("error reading length", err)
			return 0
		}

		length = int(binary.BigEndian.Uint16(b))
	} else if length == 127 {
		b := make([]byte, 8)
		_, err := io.ReadFull(reader, b)

		if err != nil {
			ws.logger.Debug("error reading length", err)
			return 0
		}

		length = int(binary.BigEndian.Uint64(b))
	}

	return length
}

func (ws Websocket) readFrame(reader *bufio.Reader) (*Frame, error) {
	fb, err := reader.ReadByte()

	if err != nil {
		ws.logger.Debug("error reading frame byte", err)
		return nil, err
	}

	frame := &Frame{}
	frame.IsFinal = fb&0x40 != 0
	frame.SetOpcode(fb & 0x0F)
	isMasked := fb&0x80 != 0

	length := ws.getPayloadLength(reader)

	mask := make([]byte, 4)
	if isMasked {
		_, err := io.ReadFull(reader, mask)
		if err != nil {
			return nil, err
		}
	}

	payload := make([]byte, length)
	_, err = io.ReadFull(reader, payload)

	if err != nil {
		ws.logger.Debug("error reading payload", err)
		return nil, err
	}

	if isMasked {
		for i := 0; i < length; i++ {
			payload[i] ^= mask[i%4]
		}
	}

	frame.Payload = payload

	return frame, nil
}

func (ws Websocket) ReadFrame() (*Frame, error) {
	reader := bufio.NewReader(ws.conn)
	return ws.readFrame(reader)
}
