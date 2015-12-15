package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/bitly/go-simplejson"
	"github.com/cloudwarehub/css/cache"
	"github.com/cloudwarehub/webftp-go"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"
	"strconv"
	
)

var Piecesize = 131072

type Uconn struct {
	ws      *websocket.Conn
	user_id string
	token   string
	authed  int
	runch   chan int
}

type piece_t struct {
	key	string
	offset          int
	index           int
	overwrite_start int
	overwrite_size  int
	data []byte
}

var api = "http://api.cloudwarehub.com"
var csscache cache.Cache

func apivisit(urlstring string, method string, token string, data map[string]string) ([]byte, error) {
	client := &http.Client{}
	var req *http.Request
	var err error
	if method == "GET" {
		u, err := url.Parse(urlstring)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		q := u.Query()
		for key, value := range data {
			q.Set(key, value)
		}
		u.RawQuery = q.Encode()
		req, err = http.NewRequest("GET", u.String(), nil)
	}

	if method == "POST" {
		u, err := url.Parse(urlstring)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		q := u.Query()
		for key, value := range data {
			q.Set(key, value)
		}
		u.RawQuery = q.Encode()
		req, err = http.NewRequest("POST", u.String(), nil)
	}

	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func make_pieces(offset int, size int) []piece_t {
	s := ((offset + size) / Piecesize) - (offset / Piecesize) + 1
	var pieces = make([]piece_t, s)
	start_offset := offset / Piecesize
	for idx, _ := range pieces {
		pieces[idx].offset = start_offset + Piecesize*idx
		pieces[idx].index = pieces[idx].offset / Piecesize
		pieces[idx].overwrite_start = 0
		pieces[idx].overwrite_size = Piecesize
		pieces[idx].data = make([]byte, Piecesize)
	}

	pieces[0].overwrite_start = offset % Piecesize
	if s == 1 {
		pieces[0].overwrite_size = size
	} else {
		pieces[0].overwrite_size = Piecesize - pieces[0].overwrite_start
		pieces[len(pieces)-1].overwrite_size = (offset + size) % Piecesize
	}

	return pieces
}

func (uconn *Uconn) handlemsg(mt int, message []byte) error {
	if mt == websocket.TextMessage { //commmand message
		obj, err := simplejson.NewJson(message)
		if err != nil {
			log.Println(err)
			return err
		}
		cmd := webftp.Cmd{S: obj.Get("seq").MustInt(), C: obj.Get("cmd").MustString(), P: obj.Get("param").MustMap()}
		switch cmd.C {
		case "ls":
			fmt.Println("ls")
			query := map[string]string{
				"dir_id": cmd.P["dir_id"].(string),
			}
			resp, err := apivisit(api+"/file/ls", "GET", uconn.token, query)
			if err != nil {
				log.Println(err)
			}
			resp_json, _ := simplejson.NewJson(resp)
			resp_json.Set("seq", cmd.S)
			str, _ := resp_json.Encode()
			fmt.Printf("%s\n", str)
			uconn.ws.WriteMessage(websocket.TextMessage, str)
		case "mkdir":
			query := map[string]string{
				"name":   cmd.P["name"].(string),
				"dir_id": cmd.P["dir_id"].(string),
			}
			resp, err := apivisit(api+"/file/mkdir", "POST", uconn.token, query)
			if err != nil {
				log.Println(err)
			}
			fmt.Printf("%s\n", resp)
			resp_json, _ := simplejson.NewJson(resp)
			resp_json.Set("seq", cmd.S)
			str, _ := resp_json.Encode()
			uconn.ws.WriteMessage(websocket.TextMessage, str)
		case "mknod":
			query := map[string]string{
				"name":   cmd.P["name"].(string),
				"dir_id": cmd.P["dir_id"].(string),
			}
			resp, err := apivisit(api+"/file/mknod", "POST", uconn.token, query)
			if err != nil {
				log.Println(err)
			}
			fmt.Printf("%s\n", resp)
			resp_json, _ := simplejson.NewJson(resp)
			resp_json.Set("seq", cmd.S)
			str, _ := resp_json.Encode()
			uconn.ws.WriteMessage(websocket.TextMessage, str)
		case "lookup":
			query := map[string]string{
				"name":   cmd.P["name"].(string),
				"dir_id": cmd.P["dir_id"].(string),
			}
			resp, err := apivisit(api+"/file/lookup", "GET", uconn.token, query)
			if err != nil {
				log.Println(err)
			}
			fmt.Printf("%s\n", resp)
			resp_json, _ := simplejson.NewJson(resp)
			resp_json.Set("seq", cmd.S)
			str, _ := resp_json.Encode()
			uconn.ws.WriteMessage(websocket.TextMessage, str)
		case "write":
			data, err := base64.StdEncoding.DecodeString(cmd.P["data"].(string))
			if err != nil {
				log.Println(err)
				return err
			}
			/* key format: user_id:piece_id:offset */
			key_prefix := uconn.user_id + ":" + cmd.P["id"].(string) + ":"
			offset, _ := obj.Get("data").Get("offset").Int()
			size, _ := obj.Get("data").Get("size").Int()

			pieces := make_pieces(offset, size)
			var wg sync.WaitGroup
			var copyptr = 0
			for idx, value := range pieces {
				wg.Add(1)
				pieces[idx].key = key_prefix+strconv.Itoa(value.index)
				if value.overwrite_start != 0 || (value.overwrite_start + value.overwrite_size) < Piecesize {
					dt, err := csscache.Get(pieces[idx].key)
					if err != nil {
						log.Println(err)
						return err
					}
					if (len(dt.([]byte)) >= Piecesize) { //ucloud文件不存在会返回错误字符串
						pieces[idx].data = dt.([]byte)
					}
				}
				copy(pieces[idx].data[value.overwrite_start:value.overwrite_start+value.overwrite_size], data[copyptr:copyptr+value.overwrite_size])
				copyptr += value.overwrite_size
				go func(key string, data []byte) {
                    defer wg.Done()
					csscache.Set(key, data)
                }(pieces[idx].key, pieces[idx].data)
			}
			wg.Wait()
			
			resp_json := simplejson.New()
			resp_json.Set("code", 0)
			resp_json.Set("seq", cmd.S)
			resp_json.Set("data", "")
			str, _ := resp_json.Encode()
			uconn.ws.WriteMessage(websocket.TextMessage, str)
			//uconn.ws.WriteJSON(map[string]interface{}{"code": 0, "seq": cmd.P["seq"].(string), "data": "write success"})
		case "read":
			key_prefix := uconn.user_id + ":" + cmd.P["id"].(string) + ":"
			offset, _ := obj.Get("data").Get("offset").Int()
			size, _ := obj.Get("data").Get("size").Int()

			pieces := make_pieces(offset, size)
			var wg sync.WaitGroup
			var tmpbuf = make([]byte, len(pieces) * Piecesize)
			for idx, value := range pieces {
				wg.Add(1)
				pieces[idx].key = key_prefix+strconv.Itoa(value.index)
				if value.overwrite_start != 0 || (value.overwrite_start + value.overwrite_size) < Piecesize {
					dt, err := csscache.Get(pieces[idx].key)
					if err != nil {
						log.Println(err)
						return err
					}
					if (len(dt.([]byte)) >= Piecesize) { //ucloud文件不存在会返回错误字符串
						pieces[idx].data = dt.([]byte)
					}
					
				}
				//copy(pieces[idx].data[value.overwrite_start:value.overwrite_start+value.overwrite_size], data[copyptr:copyptr+value.overwrite_size])
				go func(key string, data []byte) {
                    defer wg.Done()
					bf, _ := csscache.Get(key)
					copy(pieces[idx].data, bf.([]byte))
                }(pieces[idx].key, pieces[idx].data)
			}
			wg.Wait()
			for idx, _ := range pieces {
				copy(tmpbuf[idx*Piecesize:idx*Piecesize+Piecesize], pieces[idx].data)
			}
			off, _ := obj.Get("data").Get("offset").Int()
			off = off % Piecesize
			sz, _ := obj.Get("data").Get("size").Int()
			var outbuf = tmpbuf[off:off + sz]
			outstr := base64.StdEncoding.EncodeToString(outbuf)
			
			resp_json := simplejson.New()
			resp_json.Set("code", 0)
			resp_json.Set("seq", cmd.S)
			resp_json.Set("data", outstr)
			str, _ := resp_json.Encode()
			uconn.ws.WriteMessage(websocket.TextMessage, str)
			//uconn.ws.WriteJSON(map[string]interface{}{"code": 0, "seq": cmd.P["seq"].(string), "data": outstr})
		case "getattr":
			query := map[string]string{
				"id": cmd.P["id"].(string),
			}
			resp, err := apivisit(api+"/file/getattr", "GET", uconn.token, query)
			if err != nil {
				log.Println(err)
			}
			fmt.Println(resp)
			resp_json, _ := simplejson.NewJson(resp)
			resp_json.Set("seq", cmd.S)
			str, _ := resp_json.Encode()
			uconn.ws.WriteMessage(websocket.TextMessage, str)
			
		}
		
	}
	return nil
}

/*
wait until auth success
if auth failed, it will be wakeup and exit
*/
func (uconn *Uconn) recv() {
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
		fmt.Printf("%s\n", message)
		go uconn.handlemsg(mt, message)

	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	WriteBufferSize: 1024*1024,
	ReadBufferSize: 1024*1024,
}

func (uconn *Uconn) auth() {
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
	go uconn.auth()
	go uconn.recv()
}

func main() {
	var err error
	csscache, err = cache.Init("10.10.168.190:6379", "34a61e78e28f2320accf45694e2a93fdfb0786dc", "ucloudgd@tongjo.com1352889883743073806", "cloudwarehub")
	if err != nil {
		fmt.Println(err)
	}
	
	fmt.Println("starting ws server")
	var addr = flag.String("port", ":12345", "websocket server port")
	http.HandleFunc("/", handler)
	http.ListenAndServe(*addr, nil)
}
