package main

import (
	"log"
	"log/slog"
	"net/http"
)

const (
	wsGuid      = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	defaultPort = ":8080"
)

func main() {
	http.HandleFunc("/", handlerWs)

	log.Print("Server started on port", defaultPort)
	if err := http.ListenAndServe(defaultPort, nil); err != nil {
		log.Fatal(err)
	}
}

func handlerWs(w http.ResponseWriter, r *http.Request) {
	ws, err := NewWebsocket(w, r)

	if err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ws.Handshake()

	for {
		frame, err := ws.ReadFrame()
		if err != nil {
			slog.Error("error reading frame:", "error", err)
			return
		}

		// switch frame.Opcode {
		// case Text:
		// 	slog.Info("Text frame received: ", string(frame.Payload))
		// }

		slog.Info("Frame received: ", "frame", frame)
	}
}
