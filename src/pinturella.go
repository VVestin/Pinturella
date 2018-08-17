package main

import (
	"github.com/gorilla/websocket"
	"fmt"
	"encoding/json"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

var upgrader = websocket.Upgrader {
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var lettersToReveal = map[int]int{
	3:  1,   4:  1,   5:  1,
	6:  2,   7:  2,   8:  3,
	9:  3,   10: 4,   11: 4,
	12: 4,   13: 5,   14: 5,
	15: 6,   16: 6,   17: 7,
	18: 7,   19: 7,   20: 7,
}

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
	Thickness int `json:"thickness"`
	Color string `json:"color"`
}

type Hub struct {
	clients map[*Client]bool
	messages chan string
	drawing chan *Point
	register chan *Client
	unregister chan *Client
	closed bool

	// Game variables
	drawingTime <-chan time.Time
	updateTime <-chan time.Time
	revealLetters <-chan time.Time
	roundOver chan bool
	word string
	revealedWord string
	artist *Client
	round int
	numCorrect int
	roundStart time.Time
	roundEnd time.Time
}

func newHub() *Hub {
	return &Hub{
		clients: make(map[*Client]bool),
		messages: make(chan string, 256),
		drawing: make(chan *Point, 2048),
		register: make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (hub *Hub) run() {
	for {
		select {
		case client := <-hub.register:
			if !hub.closed {
				hub.clients[client] = true
			}
		case client := <-hub.unregister:
			delete(hub.clients, client)
			hub.messages <- fmt.Sprintf("%v has disconnected", client);
		case msg := <-hub.messages:
			for client, _ := range hub.clients {
				select {
				case client.sendChat <- msg:
				default:
				}
			}
		case point := <-hub.drawing:
			for client, _ := range hub.clients {
				if client != hub.artist {
					select {
					case client.sendDrawing <- point:
					default:
					}
				}
			}
		}
	}
}

func (hub *Hub) clientList() []*Client {
	list := make([]*Client, len(hub.clients))
	for client, _ := range hub.clients {
		list = append(list, client)
	}
	return list
}

func (hub *Hub) runGame() {
	hub.roundOver = make(chan bool)
	hub.startRound()
	hub.updateTime = time.Tick(1 * time.Second)
	for hub.round <= 6 {
		select {
		case <-hub.drawingTime:
			hub.roundOver <- true
		case curTime := <-hub.revealLetters:
			if hub.roundEnd.Sub(curTime).Seconds() < 2 {
				break
			}
			i := rand.Intn(len(hub.word))
			for hub.revealedWord[i] != '-' {
				i = rand.Intn(len(hub.word))
			}
			if i == len(hub.word) - 1 {
				hub.revealedWord = hub.revealedWord[:len(hub.word)-1] + string(hub.word[i])
			} else if i == 0 {
				hub.revealedWord = string(hub.word[0]) + hub.revealedWord[1:]
			} else {
				hub.revealedWord = hub.revealedWord[:i] + string(hub.word[i]) + hub.revealedWord[i+1:]
			}
			for client := range hub.clients {
				if client != hub.artist {
					client.sendChat <- fmt.Sprintf("/word %v", hub.revealedWord)
				}
			}
		case <-hub.roundOver:
			hub.messages <- fmt.Sprintf("[SERVER] Round Over. The word was %v", hub.word)
			hub.messages <- fmt.Sprintf("/word %v", hub.word)
			hub.word = ""
			time.Sleep(5 * time.Second)
			hub.startRound()
		case curTime := <-hub.updateTime:
			if curTime.Before(hub.roundEnd) {
				hub.messages <- fmt.Sprintf("/time %v", int(math.Floor(hub.roundEnd.Sub(curTime).Seconds())))
			}
		}
	}
	var winner, second *Client
	for client := range hub.clients {
		if winner == nil || client.Points >= winner.Points {
			winner, second = client, winner
		}
	}
	hub.messages <- fmt.Sprintf("[SERVER] Game Over. %v is the winner. Honorable mention goes to %v", winner, second)
}

func (hub *Hub) startRound() {
	hub.messages <- "/clear"
	hub.roundStart = time.Now()
	hub.roundEnd = hub.roundStart.Add(100 * time.Second)
	hub.numCorrect = 0
	for client := range hub.clients {
		client.Correct = false
	}

	resp, err := http.Get("http://vvest.in/Pinturella/words.php")
	if err != nil {
		fmt.Println("Error requesting vvest.in/Pinturella/words.php", err)
		return
	}
	respData := make([]byte, 120)
	wordLen, err := resp.Body.Read(respData)
	hub.word = string(respData[:wordLen])

	hub.revealedWord = ""
	for range hub.word {
		hub.revealedWord += "-"
	}
	if len(hub.revealedWord) != len(hub.word) {
		fmt.Println("ERROR")
	}
	fmt.Println(hub.revealedWord)

	hub.artist = nil
	for hub.artist == nil {
		for client := range hub.clients {
			if client.drawings <= hub.round {
				hub.artist = client
				hub.artist.drawings += 1
				break
			}
		}
		if hub.artist == nil {
			hub.round += 1
			if hub.round >= 7 {
				return
			}
		}
	}

	for client := range hub.clients {
		if client == hub.artist {
			client.sendChat <- "/word " + hub.word
		} else {
			client.sendChat <- "/word " + hub.revealedWord
		}
	}

	hub.messages <- fmt.Sprintf("[SERVER] New Round: %v is drawing", hub.artist)
	hub.sendScoreboard()
	hub.drawingTime = time.After(100 * time.Second)

	numReveal, ok := lettersToReveal[len(hub.word)]
	if !ok {
		numReveal = 8
	}
	hub.revealLetters = time.Tick(time.Duration(100 / (numReveal + 1)) * time.Second)
}

func (hub *Hub) sendScoreboard() {
	sb := "[\"" + hub.artist.Username + "\""
	for client := range hub.clients {
		clientJSON, _ := json.Marshal(client)
		sb += ", " + string(clientJSON)
	}
	sb += "]"
	fmt.Println(sb)
	hub.messages <- "/sb " + sb
}

type Client struct {
	hub *Hub
	chatConn *websocket.Conn
	drawConn *websocket.Conn
	sendChat chan string
	sendDrawing chan *Point
	running bool
	Username string `json:"username"`
	Points int `json:"points"`
	Correct bool `json:"correct"`
	drawings int
}

func (c *Client) readChat() {
	for c.running {
		mType, msg, err := c.chatConn.ReadMessage()
		if err != nil {
			fmt.Println("conn.ReadMessage() error:", err)
			c.hub.unregister <- c
			c.disconnect()
			return
		}
		if mType == websocket.CloseMessage {
			c.hub.unregister <- c
			c.disconnect()
			return
		} else if mType == websocket.TextMessage {
			var text = string(msg)
			fmt.Println("New message:", text)
			if c.Username == "" {
				c.Username = text
				c.sendChat <- fmt.Sprintf("[SERVER]: %d players connected %v", len(c.hub.clients), c.hub.clientList())
				c.hub.messages <- text + " has connected"
			} else if strings.HasPrefix(text, "/") {
				fmt.Println("command", text)
				text = text[1:]
				switch text {
				case "start":
					go c.hub.runGame()
				case "clear":
					c.hub.messages <- "/clear"
				case "ping":
					c.sendChat <- "Pong!"
				default:
					c.sendChat <- fmt.Sprintf("<span style=\"color: red\">'%v' is not a command</span>", text)
				}
			} else {
				if c != c.hub.artist && c.hub.word != "" && strings.HasPrefix(strings.ToLower(text), strings.ToLower(c.hub.word)) {
					timeLeft := c.hub.roundEnd.Sub(time.Now()).Seconds()
					c.Points += int(math.Floor(timeLeft / 2))
					c.hub.artist.Points += int(math.Floor(timeLeft / 4))
					c.Correct = true
					c.hub.sendScoreboard()

					c.hub.numCorrect += 1

					if c.hub.numCorrect == len(c.hub.clients) - 1 {
						fmt.Println("Everyone has guessed correctly")
						c.hub.roundOver <- true
					}
				} else {
					c.hub.messages <- c.Username + ": " + text
				}
			}
		}
	}
}

func (c *Client) writeChat() {
	for c.running {
		msg, ok := <-c.sendChat
		if !ok {
			c.chatConn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		c.chatConn.WriteMessage(websocket.TextMessage, []byte(msg))
	}
}

func (c *Client) readDrawing() {
	for c.running {
		point := &Point{}
		err := c.drawConn.ReadJSON(point)
		if err == nil {
			fmt.Println("New point:", point.X, point.Y)
			c.hub.drawing <- point
		}
	}
}

func (c *Client) writeDrawing() {
	for c.running {
		point, ok := <-c.sendDrawing
		if !ok {
			c.drawConn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		c.drawConn.WriteJSON(point)
	}
}

func (c *Client) disconnect() {
	if c.running {
		c.running = false
		c.drawConn.Close()
		c.chatConn.Close()
		close(c.sendChat)
		close(c.sendDrawing)
	}
}

func (c *Client) String() string {
	return c.Username
}

func main() {
	rand.Seed(time.Now().Unix())

	hub := newHub()
	go hub.run()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("served index.html")
		http.ServeFile(w, r, "index.html")
	})
	http.HandleFunc("/pinturella/chat", func(w http.ResponseWriter, r *http.Request) {
		var conn, err = upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println("Connection could not be upgraded to a websocket")
			fmt.Println(err)
			return
		}
		fmt.Println("New chat websocket connected")

		if hub.closed {
			hub = newHub()
		}
		client := &Client{
			hub: hub,
			chatConn: conn,
			sendChat: make(chan string, 256),
			sendDrawing: make(chan *Point, 2048),
			running: true,
		}
		hub.register <- client

		go client.readChat()
		go client.writeChat()
	})
	http.HandleFunc("/pinturella/draw", func(w http.ResponseWriter, r *http.Request) {
		var conn, err = upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println("Connection could not be upgraded to a websocket")
			fmt.Println(err)
			return
		}
		if hub.closed {
			hub = newHub()
		}
		Username := r.FormValue("un")
		var client *Client
		for c, _ := range hub.clients {
			if c.Username == Username {
				fmt.Println("New drawing websocket connected for", Username)
				client = c
				client.drawConn = conn
				break
			}
		}
		go client.readDrawing()
		go client.writeDrawing()
	})

	fmt.Println("Server Started...")

	http.ListenAndServe(":6213", nil)
}
