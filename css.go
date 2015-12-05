package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/bitly/go-simplejson"
	"net/http"
	"net/url"
	"io/ioutil"
	"log"
)

type Uconn struct {
	ws   *websocket.Conn
	user_id string
	token string
	authed int
	runch chan int
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}


func auth(uconn *Uconn) {
    resp, err := http.Get("http://api.cloudwarehub.com/user?token=" + uconn.token)
	if err != nil {
	    fmt.Println("api access error")
		return
	}
	
	js, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		fmt.Println(err)
	}
	
	obj, err := simplejson.NewJson(js)
	if obj.Get("code").MustInt() != 0 {
	    uconn.ws.Close()
	    uconn.runch <- 1 // wakeup and exit recv routine
	    return
	}
	info := obj.Get("data").MustMap()
	fmt.Println(info)
	
	uconn.user_id = info["id"].(string)
	uconn.authed = 1
	uconn.runch <- 1
}

func recv(uconn *Uconn) {
    /*
    wait until auch success
    */
    <-uconn.runch
    if uconn.authed == 0 { //auth failed, exit recv goroutine
        return
    }
    ws := uconn.ws
    for {
		mt, message, err := ws.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		err = ws.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

/*
To achieve none-blocking connect:
    1. upgrade to websocket directly and set status to unauthed
    2. goroutine auth and get storage information, set status to authed
    3. goroutine recv
    
    if status is unauthed, recv routine just ignore all messages from client
*/
func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Println("new client")
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	

	queryForm, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
	    log.Println("url parse error")
	    ws.Close()
	    return
	}
	
	if len(queryForm["token"]) < 0 {
	    ws.Close()
	    return
	}
	
	token := queryForm["token"][0]
	uconn := &Uconn{ws: ws, token: token, authed: 0, runch: make(chan int)}
	go auth(uconn)
	go recv(uconn)
}

func main() {
	var addr = flag.String("port", ":12345", "websocket server port")
	http.HandleFunc("/", handler)
	http.ListenAndServe(*addr, nil)
}
