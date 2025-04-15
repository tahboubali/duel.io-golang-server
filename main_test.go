package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"testing"
	"time"
)

func TestServer_Run(t *testing.T) {
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/connect"}
	log.Printf("Connecting to %s", u.String())

	conn1, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("Dial error:", err)
	}
	defer func() {
		_ = conn1.Close()
	}()
	conn2, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("Dial error:", err)
	}
	defer func() {
		_ = conn2.Close()
	}()

	err = conn1.WriteJSON(map[string]any{
		"request_type": "new-player",
		"data":         map[string]string{"username": "tahboubali"},
	})

	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(time.Second)

	_, data, _ := conn1.ReadMessage()
	fmt.Println("Conn1:", string(data))

	err = conn2.WriteJSON(map[string]any{
		"request_type": "new-player",
		"data":         map[string]string{"username": "tahboubali2"},
	})
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second)
	_, data, _ = conn2.ReadMessage()
	fmt.Println("Conn2:", string(data))

	err = conn1.WriteJSON(map[string]any{
		"request_type": "enter-duel",
	})
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second)
	_, data, _ = conn1.ReadMessage()
	fmt.Println("Conn1:", string(data))

	err = conn2.WriteJSON(map[string]any{
		"request_type": "enter-duel",
	})
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second)
	_, data, _ = conn2.ReadMessage()
	time.Sleep(time.Second)
	fmt.Println("Conn2:", string(data))

	time.Sleep(time.Second)
	_, data, _ = conn1.ReadMessage()
	fmt.Println("Conn1:", string(data))
	_, data, _ = conn2.ReadMessage()
	fmt.Println("Conn2:", string(data))

	time.Sleep(time.Second)
	_ = conn1.WriteJSON(map[string]any{
		"request_type": "game-end",
		"data":         map[string]any{"player_won": "tahboubali"},
	})

	time.Sleep(time.Second)
	_, data, _ = conn1.ReadMessage()
	fmt.Println("Conn1:", string(data))
	_, data, _ = conn2.ReadMessage()
	fmt.Println("Conn2:", string(data))
}
