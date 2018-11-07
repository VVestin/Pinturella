package main

import (
	"github.com/gorilla/websocket"
	"fmt"
	"net/http"
	"strings"
	"io"
	"io/ioutil"
	"encoding/json"
)

type quoteResponse struct {
	Quotes []quote `json:"quotes"`
}

type quote struct {
	Author string `json:"author"`
	Quote string `json:"quote"`
	Tags []string `json:"tags"`
}

var upgrader = websocket.Upgrader {
	ReadBufferSize:  128,
	WriteBufferSize: 128,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func pigLatin(word string) string {
	vowelIdx := strings.IndexAny(word, "aeiouyAEIOUY")
	if vowelIdx == -1 {
		fmt.Printf("could not convert %s into piglatin\n", word)
		return word
	}
	if vowelIdx == 0 {
		return word + "-ay"
	}
	return word[vowelIdx:] + "-" + word[:vowelIdx] + "ay"
}

func echoPigLatin(conn *websocket.Conn) {
	defer conn.Close()
	for {
		msgType, msg, err := conn.ReadMessage()
		if msgType == websocket.CloseMessage {
			fmt.Println("Connection closed")
			return
		}
		if err != nil {
			fmt.Printf("Error reading message from socket: %v\n", err)
			return
		}
		if msgType == websocket.TextMessage {
			words := strings.Split(string(msg), " ")
			for i := range words {
				words[i] = pigLatin(words[i])
			}
			err := conn.WriteMessage(websocket.TextMessage, []byte(strings.Join(words, " ")))
			if err != nil {
				fmt.Printf("Error writing message to socket: %v\n", err)
				return
			}
		} else {
			fmt.Println("non close or text message recieved")
		}
	}
}

func main() {
	http.HandleFunc("/piglatin", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Connection!")
		// TODO should I close w and r?
		var conn, err = upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println("Connection could not be upgraded to a websocket")
			return
		}
		go echoPigLatin(conn)
	})
	http.HandleFunc("/quote", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		resp, err := http.Get("http://opinionated-quotes-api.gigalixirapp.com/v1/quotes")
		if err != nil {
			fmt.Printf("Error fetching quote: %v\n", err)
			return
		}
		rawQuote, err := ioutil.ReadAll(resp.Body)
		var quoteResp quoteResponse
		err = json.Unmarshal(rawQuote, &quoteResp)
		if err != nil {
			fmt.Printf("Error parsing JSON: %v\n", err)
			return
		}
		quoteJSON, err := json.Marshal(quoteResp.Quotes[0])
		if err != nil {
			fmt.Printf("Error converting to JSON: %v\n", err)
			return
		}
		if err != nil {
			fmt.Printf("Error reading quote from response: %v\n", err)
			return
		}
		fmt.Println(string(quoteJSON))
		io.WriteString(w, string(quoteJSON))
	})
	fmt.Println("Server Started...")

	http.ListenAndServe(":6213", nil)
}
