package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
)

//go:embed ui/*
var uiFiles embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	port := flag.String("port", "", "MIDI port name (substring match)")
	flag.Parse()

	if *port == "" {
		log.Println("Available MIDI ports:")
		for _, p := range midi.GetOutPorts() {
			log.Printf("  OUT: %s", p.String())
		}
		for _, p := range midi.GetInPorts() {
			log.Printf("  IN:  %s", p.String())
		}
		log.Fatal("Please specify -port flag")
	}

	outPort, err := midi.FindOutPort(*port)
	if err != nil {
		log.Fatalf("Cannot find MIDI out port %q: %v", *port, err)
	}
	inPort, err := midi.FindInPort(*port)
	if err != nil {
		log.Fatalf("Cannot find MIDI in port %q: %v", *port, err)
	}

	send, err := midi.SendTo(outPort)
	if err != nil {
		log.Fatalf("Cannot open MIDI out: %v", err)
	}

	// Serve embedded UI
	uiFS, _ := fs.Sub(uiFiles, "ui")
	http.Handle("/", http.FileServer(http.FS(uiFS)))

	// WebSocket MIDI bridge
	http.HandleFunc("/midi", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}
		defer conn.Close()

		var mu sync.Mutex

		// MIDI In → WebSocket
		stop, err := midi.ListenTo(inPort, func(msg midi.Message, timestampms int32) {
			mu.Lock()
			defer mu.Unlock()
			conn.WriteMessage(websocket.BinaryMessage, msg.Bytes())
		})
		if err != nil {
			log.Printf("Cannot listen to MIDI in: %v", err)
			return
		}
		defer stop()

		// WebSocket → MIDI Out
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				break
			}
			send(data)
		}
	})

	log.Printf("opendeck-bridge listening on %s (MIDI: %s)", *addr, outPort.String())
	log.Fatal(http.ListenAndServe(*addr, nil))
}
