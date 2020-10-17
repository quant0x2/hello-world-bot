package main

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/http"
)

func main() {
	var arr = make([]byte, 16)
	rand.Read(arr)
	var client http.Client
	b64 := base64.StdEncoding.EncodeToString(arr)
	req, er := http.NewRequest("GET", "http://localhost:9001/ws", nil)
	if er != nil {
		return
	}
	req.Header.Add("Upgrade", "websocket")
	req.Header.Add("Connection", "Upgrade")
	req.Header.Add("Sec-WebSocket-Key", b64)
	req.Header.Add("Sec-WebSocket-Version", "13")
	_, err := client.Do(req)
	if err != nil {
		fmt.Println("Error")
	}
	//fmt.Println(resp.Header["Sec-WebSocket-Accept"])
}
