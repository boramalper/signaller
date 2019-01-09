package main

import (
	"flag"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type pcEntry struct {
	conn     *websocket.Conn
	deadline time.Time
}

// Command-Line Flags
var addr = flag.String("addr", ":8080", "address")
var originRES = flag.String("originRE", "", "regular expression for the Origin request header")
var listenDL = flag.Uint("listenDL", 60, "deadline in seconds before a listening WS is closed")
var pipeDL = flag.Uint("pipeDL", 10, "deadline in seconds before a whole pipe is closed")
var maxSize = flag.Uint("maxSize", 1024, "max size of a message in bytes")
var maxMsg = flag.Uint("maxMsg", 8, "max number of messages that can be sent from a peer")

var mutex sync.Mutex
var pendingConnections map[string]pcEntry

var upgrader websocket.Upgrader

// handleRES specifies that a handle
//   - must begin with a lowercase character, followed by lower characters or underscores
//   - must be at least 3 at most 32 characters long
//   - cannot contain consecutive underscores
const handleRES = `[a-z](?:_?[a-z0-9]){2,31}`

// tickPD is the period between two consecutive iterations of a cleaner that
// cleans up the expired pending connections.
// In seconds.
const tickPD = 5

// Application Close Codes
const listenTimeout = 4000
const pipeTimeout = 4001
const tooManyMessages = 4002

func main() {
	flag.Parse()

	if *originRES == "" {
		log.Printf("-originRE is empty, WS handshake will fail if the Origin request header is present and the" +
			" Origin host is not equal to the Host request header.")
		upgrader = websocket.Upgrader{
			// If the CheckOrigin field is nil, then the Upgrader uses a safe
			// default: fail the handshake if the Origin request header is
			// present and the Origin host is not equal to the Host request header.
			// https://godoc.org/github.com/gorilla/websocket#hdr-Origin_Considerations
			CheckOrigin: nil,
		}
	} else {
		originRE, err := regexp.Compile(*originRES)
		if err != nil {
			log.Fatal("could not parse origin regex", err)
		}

		upgrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header["Origin"][0]
				return originRE.MatchString(origin)
			},
		}
	}

	if *listenDL < tickPD+1 {
		log.Fatalf("-listenDL cannot be less than %d", tickPD+1)
	}
	if *pipeDL < tickPD+1 {
		log.Fatalf("-pipeDL cannot be less than %d", tickPD+1)
	}

	if *maxSize == 0 {
		log.Fatal("-maxSize cannot be zero")
	}

	pendingConnections = make(map[string]pcEntry)

	// Close the expired connections periodically
	ticker := time.NewTicker(tickPD * time.Second)
	go func() {
		for range ticker.C {
			mutex.Lock()

			now := time.Now()
			for handle, entry := range pendingConnections {
				if entry.deadline.Before(now) {
					_ = entry.conn.CloseHandler()(listenTimeout, "Listen Timeout")
					delete(pendingConnections, handle)
				}
			}

			mutex.Unlock()
		}
	}()

	router := mux.NewRouter()
	router.HandleFunc("/listen/{handle:"+handleRES+"}", listen)
	router.HandleFunc("/connect/{handle:"+handleRES+"}", connect)

	if err := http.ListenAndServe(*addr, router); err != nil {
		log.Fatal("ListenAndServe", err)
	}
	ticker.Stop()
}

func listen(w http.ResponseWriter, r *http.Request) {
	handle := mux.Vars(r)["handle"]

	mutex.Lock()
	defer mutex.Unlock()

	if _, exists := pendingConnections[handle]; exists {
		w.WriteHeader(http.StatusConflict)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	conn.SetCloseHandler(func(code int, text string) error {
		message := websocket.FormatCloseMessage(code, text)
		_ = conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))
		return conn.Close()
	})

	// A message cannot exceed 1 KiB.
	conn.SetReadLimit(int64(*maxSize))

	pendingConnections[handle] = pcEntry{
		conn:     conn,
		deadline: time.Now().Add(time.Duration(*listenDL) * time.Second),
	}
}

func connect(w http.ResponseWriter, r *http.Request) {
	handle := mux.Vars(r)["handle"]

	mutex.Lock()
	entry, exists := pendingConnections[handle]
	if !exists {
		mutex.Unlock()
		w.WriteHeader(http.StatusNotFound)
		return
	}
	// delete the handle from pendingConnections so that another peer cannot
	// attempt to connect to it whilst we are initialising.
	delete(pendingConnections, handle)
	mutex.Unlock()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// We could not upgrade the connection to WebSockets, so add handle
		// back to pendingConnections
		mutex.Lock()
		pendingConnections[handle] = entry
		mutex.Unlock()
		return
	}

	conn.SetCloseHandler(func(code int, text string) error {
		message := websocket.FormatCloseMessage(code, text)
		_ = conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))
		return conn.Close()
	})

	// A message cannot exceed 1 KiB.
	conn.SetReadLimit(int64(*maxSize))

	// `deadline` applies for the whole pipe so read/write operations on both
	// connections have the same deadline.
	// Ignore errors while setting the deadline.
	deadline := time.Now().Add(time.Duration(*pipeDL) * time.Second)
	_ = entry.conn.SetReadDeadline(deadline)
	_ = entry.conn.SetWriteDeadline(deadline)
	_ = conn.SetReadDeadline(deadline)
	_ = conn.SetWriteDeadline(deadline)

	pipe := func(from *websocket.Conn, to *websocket.Conn) {
		var nMsg uint

		for nMsg = 0; nMsg < *maxMsg; nMsg++ {
			typ, dat, err := from.ReadMessage()
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				_ = from.Close()
				_ = to.CloseHandler()(websocket.CloseNormalClosure, "Normal Closure")
				return
			} else if err != nil {
				break
			}

			err = to.WriteMessage(typ, dat)
			if err != nil {
				break
			}
		}

		if deadline.Before(time.Now()) { // deadline has passed!
			_ = from.CloseHandler()(pipeTimeout, "Pipe Timeout")
			_ = to.CloseHandler()(pipeTimeout, "Pipe Timeout")
		} else if nMsg >= *maxMsg {
			_ = from.CloseHandler()(tooManyMessages, "Too Many Messages")
			_ = to.CloseHandler()(tooManyMessages, "Too Many Messages")
		} else {
			_ = from.CloseHandler()(websocket.CloseInternalServerErr, "Internal Error")
			_ = to.CloseHandler()(websocket.CloseInternalServerErr, "Internal Error")
		}
	}

	go pipe(entry.conn, conn)
	go pipe(conn, entry.conn)
}
