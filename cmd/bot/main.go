package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Content          string           `json:"content"`
	MessageReference ReferenceMessage `json:"message_reference"`
}
type ReferenceMessage struct {
	ID        string `json:"message_id"`
	ChannelID string `json:"channel_id"`
	GuildID   string `json:"guild_id"`
}
type Identity struct {
	Token      string             `json:"token"`
	Intents    int                `json:"intents"`
	Properties IdentityProperties `json:"properties"`
}

type IdentityProperties struct {
	Os      string `json:"$os"`
	Browser string `json:"$browser"`
	Device  string `json:"$device"`
}

type GatewayIdentityRequest struct {
	Opcode      int      `json:"op"`
	Data        Identity `json:"d"`
	EventName   string   `json:"t"`
	SequenceNum int      `json:"s"`
}

type GatewayRequestResponse struct {
	Opcode      int         `json:"op"`
	Data        interface{} `json:"d"`
	EventName   string      `json:"t"`
	SequenceNum int         `json:"s"`
}

func main() {

	// Special for interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	var key string
	fmt.Scan(&key)
	fmt.Println("Testing authorization in Discord...")

	client, _, err := websocket.DefaultDialer.Dial("wss://gateway.discord.gg/?v=8&encoding=json", nil)
	if err != nil {
		log.Println(err)
	}
	defer client.Close()

	var botID string
	done := make(chan struct{})
	welcome := make(chan bool)
	messages := make(chan ReferenceMessage)
	heartbeatInterval := make(chan string)

	go func() {
		defer close(done)
		for {
			var decodedData GatewayRequestResponse
			_, message, err := client.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				break
			}
			if err := json.Unmarshal(message, &decodedData); err != nil {
				log.Println(err)
			}
			fmt.Println(decodedData.SequenceNum)
			if decodedData.EventName == "READY" {
				fmt.Println("Ready event appeared!")
				fmt.Println("Setting our unique bot id...")
				nestedData := decodedData.Data.(map[string]interface{})
				innerData := nestedData["user"].(map[string]interface{})
				botID = innerData["id"].(string)
				fmt.Println("OK")
			}
			if decodedData.EventName == "MESSAGE_CREATE" {
				fmt.Println("Message event appeared!")
				nestedData := decodedData.Data.(map[string]interface{})
				innerData := nestedData["author"].(map[string]interface{})
				userID := innerData["id"].(string)
				fmt.Println("Checking user id...")
				if userID != botID {
					messageInfo := ReferenceMessage{
						ID:        nestedData["id"].(string),
						ChannelID: nestedData["channel_id"].(string),
						GuildID:   nestedData["guild_id"].(string),
					}
					messages <- messageInfo
				}
			}
			if decodedData.Opcode == 10 {
				fmt.Println("Discord Gateway said hi!")
				fmt.Println("Getting heartbeat interval...")
				nestedData := decodedData.Data.(map[string]interface{})
				temp := nestedData["heartbeat_interval"].(float64)
				heartbeatInterval <- strconv.FormatFloat(temp, 'f', -1, 64)
				fmt.Println("OK")
				fmt.Println("Getting ready to send our identity...")
				welcome <- true
			}
		}
	}()

	temp := <-heartbeatInterval
	duration, _ := time.ParseDuration(temp + "ms")
	ticker := time.NewTicker(duration)

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			fmt.Println("Sending hearbeat...")
			var hearbeat = GatewayRequestResponse{
				Opcode: 1,
				Data:   nil,
			}
			var encoded, _ = json.Marshal(hearbeat)
			err := client.WriteMessage(websocket.TextMessage, encoded)
			if err != nil {
				log.Println(err)
			}
			fmt.Println("OK")
		case <-welcome:
			// Sending ours identity
			fmt.Println("Sending our identity...")
			var identity = GatewayIdentityRequest{
				Opcode: 2,
				Data: Identity{
					Token:   key,
					Intents: 512,
					Properties: IdentityProperties{
						Os:      "linux",
						Browser: "Lacia_internal",
						Device:  "Lacia_internal",
					},
				},
			}
			jsonData, _ := json.Marshal(identity)
			err := client.WriteMessage(websocket.TextMessage, jsonData)
			if err != nil {
				log.Println(err)
			}
			fmt.Println("OK")
		case userMessage := <-messages:
			var messageToSend = Message{
				Content:          "Hi",
				MessageReference: userMessage,
			}
			jsonData, _ := json.Marshal(messageToSend)
			hclient := http.Client{}
			req, _ := http.NewRequest(
				"POST",
				"https://discord.com/api/v8/channels/"+userMessage.ChannelID+"/messages",
				bytes.NewReader(jsonData),
			)
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("Authorization", "Bot "+key)
			_, err := hclient.Do(req)
			if err != nil {
				log.Println(err)
			}
			fmt.Println("OK")
		case <-interrupt:
			log.Printf("Interrupt")
			err := client.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			)
			if err != nil {
				log.Println("write close:", err)
				return
			}
			return
		}
	}
}
