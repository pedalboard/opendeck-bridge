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

	// Serve embedded UI with auto-connect injection
	uiFS, _ := fs.Sub(uiFiles, "ui")
	fileServer := http.FileServer(http.FS(uiFS))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			indexData, err := uiFiles.ReadFile("ui/index.html")
			if err != nil {
				http.Error(w, "not found", 404)
				return
			}
			// Inject script to auto-set WebConfig address to current host
			autoConnect := `<script>localStorage.setItem("opendeck-webconfig-address",location.host)</script>`
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(autoConnect))
			w.Write(indexData)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	// WebSocket MIDI bridge (compatible with OpenDeckUI WebConfig transport)
	http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
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
