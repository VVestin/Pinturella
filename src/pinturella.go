package main

import (
	"github.com/gorilla/websocket"
	"fmt"
	"net/http"
)

var upgrader = websocket.Upgrader {}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	http.HandleFunc("/v1/ws", func(w http.ResponseWriter, r *http.Request) {
		var conn, _ = upgrader.Upgrade(w, r, nil)
		go func(conn *websocket.Conn) {
			for {
				mType, msg, _ := conn.ReadMessage();
				conn.WriteMessage(mType, msg);
			}
		}(conn)
	})

	fmt.Println("Server Started...");

	http.ListenAndServe(":8080", nil)
}
