package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
)

type Uconn struct {
	wconn   *websocket.Conn
	user_id string
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func serve(uconn *Uconn) {
	//go recv_routine(uconn)
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("new client")
	var user_id string
	queryForm, err := url.ParseQuery(r.URL.RawQuery)
	if err == nil && len(queryForm["user_id"]) > 0 {
		user_id = queryForm["user_id"][0]
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	uconn := Uconn{wconn: conn, user_id: user_id}
	go serve(&uconn)
}

func main() {
	var addr = flag.String("port", ":12345", "http service address")
	http.HandleFunc("/", handler)
	http.ListenAndServe(*addr, nil)
}
