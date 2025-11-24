package main

import (
	"context"
	"embed"
	"flag"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/spf13/afero"
	"github.com/telnet2/go-practice/go-memsh"
)

//go:embed static/*
var staticFiles embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// WebSocketIO implements io.Reader and io.Writer for WebSocket
type WebSocketIO struct {
	conn       *websocket.Conn
	inputBuf   []byte
	inputMu    sync.Mutex
	outputMu   sync.Mutex
	readChan   chan []byte
	closeChan  chan struct{}
	closeOnce  sync.Once
}

func newWebSocketIO(conn *websocket.Conn) *WebSocketIO {
	wsio := &WebSocketIO{
		conn:      conn,
		readChan:  make(chan []byte, 100),
		closeChan: make(chan struct{}),
	}

	// Start reading from WebSocket
	go wsio.readLoop()

	return wsio
}

func (w *WebSocketIO) readLoop() {
	for {
		select {
		case <-w.closeChan:
			return
		default:
		}

		_, message, err := w.conn.ReadMessage()
		if err != nil {
			close(w.closeChan)
			return
		}

		select {
		case w.readChan <- message:
		case <-w.closeChan:
			return
		}
	}
}

func (w *WebSocketIO) Read(p []byte) (n int, err error) {
	// If we have buffered input, return it
	w.inputMu.Lock()
	if len(w.inputBuf) > 0 {
		n = copy(p, w.inputBuf)
		w.inputBuf = w.inputBuf[n:]
		w.inputMu.Unlock()
		return n, nil
	}
	w.inputMu.Unlock()

	// Wait for input
	select {
	case data := <-w.readChan:
		w.inputMu.Lock()
		w.inputBuf = data
		n = copy(p, w.inputBuf)
		w.inputBuf = w.inputBuf[n:]
		w.inputMu.Unlock()
		return n, nil
	case <-w.closeChan:
		return 0, io.EOF
	}
}

func (w *WebSocketIO) Write(p []byte) (n int, err error) {
	w.outputMu.Lock()
	defer w.outputMu.Unlock()

	select {
	case <-w.closeChan:
		return 0, io.EOF
	default:
	}

	err = w.conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

func (w *WebSocketIO) Close() error {
	w.closeOnce.Do(func() {
		close(w.closeChan)
		w.conn.Close()
	})
	return nil
}

func main() {
	addr := flag.String("addr", ":8080", "HTTP server address")
	flag.Parse()

	// Serve static files
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", handleWebSocket)

	log.Printf("Starting web shell server on %s", *addr)
	log.Printf("Open http://localhost%s in your browser", *addr)

	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Serve the embedded HTML file
	data, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		log.Printf("Error reading index.html: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	log.Printf("New connection from %s", conn.RemoteAddr())

	// Create WebSocket IO
	wsio := newWebSocketIO(conn)
	defer wsio.Close()

	// Create in-memory filesystem
	fs := afero.NewMemMapFs()

	// Create shell
	sh, err := memsh.NewShell(fs)
	if err != nil {
		log.Printf("Failed to create shell: %v", err)
		return
	}

	// Set WebSocket as I/O
	sh.SetIO(wsio, wsio, wsio)

	// Run the shell
	ctx := context.Background()
	if err := sh.RunInteractive(ctx); err != nil {
		log.Printf("Shell error: %v", err)
	}

	log.Printf("Connection closed from %s", conn.RemoteAddr())
}
