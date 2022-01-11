package main

import (
	"encoding/json"
	"github.com/go-redis/redis"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"os"
)

var (
	rdb *redis.Client
)

var clients = make(map[*websocket.Conn]bool)
var broadcaster = make(chan ChatMessage)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ChatMessage struct {
	Username string `json:”username”`
	Text     string `json:”text”`
}

func main() {
	//redisURL := os.Getenv("REDIS_URL")
	//opt,err := redis.ParseURL(redisURL)
	//if err != nil {
	//	panic(err)
	//}
	//rdb = redis.NewClient(opt)
	//err = godotenv.Load()
	//if err != nil {
	//	log.Fatal("Error loading .env file")
	//}
	port := os.Getenv("PORT")
	http.Handle("/", http.FileServer(http.Dir("./public")))
	log.Print("Server starting at localhost:4444")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/websocket", handleConnections)
	go handleMessages()

}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	defer ws.Close()
	clients[ws] = true

	for {
		var msg ChatMessage
		err = ws.ReadJSON(&msg)
		if err != nil {
			delete(clients, ws)
		}

		broadcaster <- msg
	}
}

func handleMessages() {
	var msg ChatMessage
	for {
		msg = <-broadcaster
	}

	json, err := json.Marshal(msg)

	if err != nil {
		panic(err)
	}

	err = rdb.RPush("chatmessages", json).Err()
	if err != nil {
		panic(err)
	}

	for client := range clients {
		err := client.WriteJSON(msg)
		if err != nil && unsafeError(err) {
			log.Printf("error : %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}

func unsafeError(err error) bool {
	return !websocket.IsCloseError(err, websocket.CloseGoingAway) && err != io.EOF
}
