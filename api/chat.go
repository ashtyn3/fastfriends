package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"locy/auth"
	"locy/crypto"
	"locy/user"
	"log"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/lib/pq"
	"github.com/valyala/fasthttp"
)

type RoomRequest struct {
	Id      int      `json:"id,omitempty"`
	Members []string `json:"members,omitempty"`
}

var db, str = auth.Auth().DB.Init()

func (r *RoomRequest) Enter(ctx *fasthttp.RequestCtx) int64 {
	arr := "'{"
	for _, sig := range r.Members {
		u := user.GetUser(sig)
		if u.Username == "" {
			fmt.Fprintf(ctx, "Failed to find chat member.")
		}
		arr += sig + ","
	}
	arr += "}'"
	query := fmt.Sprintf("insert into rooms (members) values (%s)", arr)
	res, _ := db.Exec(query)
	id, _ := res.LastInsertId()
	return id
}

func (r *RoomRequest) Exit() {
	query := fmt.Sprintf("delete from rooms where id=%d", r.Id)
	db.Exec(query)
}

var upgrader = websocket.FastHTTPUpgrader{}

var clients = make(map[*websocket.Conn]int)
var broadcast = make(chan string)
var mutex sync.Mutex

func handleMessages(id int) {
	defer mutex.Lock()
	mutex.Unlock()
	for {
		msg := <-broadcast
		for client, RID := range clients {
			if RID == id {
				err := client.WriteJSON(msg)
				if err != nil {
					log.Printf("error: %v", err)
					//client.Close()
					//delete(clients, client)
				}
			}
		}
	}
}

type notif struct {
	Table  string `json:"table"`
	Action string `json:"action"`
	Data   struct {
		ID        int         `json:"id"`
		Msg       string      `json:"msg"`
		Timestamp string      `json:"timestamp"`
		From      interface{} `json:"From"`
		User      interface{} `json:"User"`
		Room      interface{} `json:"room"`
	} `json:"data"`
}

func waitForNotification(l *pq.Listener) {
	for {
		select {
		case n := <-l.Notify:
			tdata := notif{}
			json.Unmarshal([]byte(n.Extra), &tdata)
			broadcast <- n.Extra
			return
		case <-time.After(90 * time.Second):
			go func() {
				l.Ping()
			}()
			return
		}
	}
}

func (r *RoomRequest) Listen() {

	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			fmt.Println(err.Error())
		}
	}
	listener := pq.NewListener(str, 10*time.Second, time.Minute, reportProblem)
	err := listener.Listen("events")
	if err != nil {
		panic(err)
	}
	for {
		waitForNotification(listener)
	}
}

type SendBody struct {
	Msg    string `json:"msg"`
	RoomId int    `json:"roomId"`
}

func serveWs(ctx *fasthttp.RequestCtx, roomInt int, req RoomRequest) {
	upgrader.CheckOrigin = func(r *fasthttp.RequestCtx) bool { return true }
	err := upgrader.Upgrade(ctx, func(ws *websocket.Conn) {
		defer ws.Close()
		clients[ws] = int(roomInt)
		go req.Listen()
		go handleMessages(int(roomInt))
		for {
			ws.PingHandler()
		}
	})

	if err != nil {
		if _, ok := err.(websocket.HandshakeError); ok {
			log.Println(err)
		}
		return
	}
}

func ChatEndpoint(ctx *fasthttp.RequestCtx) {
	switch ctx.UserValue("op") {
	case "rooms":
		{
			tok := ctx.Request.Header.Peek("Authorization")
			keys := strings.Split(string(tok), " ")
			grant_tok := ledger[keys[0]]
			if grant_tok == "" {
				log.Println("Could not authorize request.")
				fmt.Fprintf(ctx, "Could not authorize request.")
				return
			}

			enc := user.GetLink(keys[0])
			if enc.Data == "" {
				log.Println("Could not authorize request.")
				fmt.Fprintf(ctx, "Could not authorize request.")
				return
			}
			sig := crypto.Decrypt(keys[1], enc.Data)
			u := user.GetUser(sig)

			if u.Username == "" {
				log.Println("User does not exist.")
				fmt.Fprintf(ctx, "User does not exist.")
				return
			}

			data, _ := json.Marshal(user.UserRooms(sig))
			fmt.Fprint(ctx, data)
		}
	case "create":
		{
			tok := ctx.Request.Header.Peek("Authorization")
			keys := strings.Split(string(tok), " ")
			grant_tok := ledger[keys[0]]
			if grant_tok == "" {
				log.Println("Could not authorize request.")
				fmt.Fprintf(ctx, "Could not authorize request.")
				return
			}

			enc := user.GetLink(keys[0])
			if enc.Data == "" {
				log.Println("Could not authorize request.")
				fmt.Fprintf(ctx, "Could not authorize request.")
				return
			}
			sig := crypto.Decrypt(keys[1], enc.Data)

			u := user.GetUser(sig)
			if u.Username == "" {
				log.Println("User does not exist.")
				fmt.Fprintf(ctx, "User does not exist.")
				return
			}
			bod := ctx.Request.Body()
			roomReq := RoomRequest{}
			json.Unmarshal(bod, &roomReq)
			roomId := roomReq.Enter(ctx)
			std := base64.StdEncoding.EncodeToString(big.NewInt(roomId).Bytes())
			fmt.Fprint(ctx, std)
		}
	case "listen":
		{
			body := ctx.QueryArgs().Peek("token")
			args := strings.Split(string(body), "-")
			keys := args[0:2]
			codedRoomId := args[2]
			grant_tok := ledger[keys[0]]
			fmt.Println(keys[0])
			if grant_tok == "" {
				log.Println("Could not authorize request.")
				fmt.Fprintf(ctx, "Could not authorize request.")
				return
			}

			enc := user.GetLink(keys[0])
			if enc.Data == "" {
				log.Println("Could not authorize request.")
				fmt.Fprintf(ctx, "Could not authorize request.")
				return
			}
			sig := crypto.Decrypt(keys[1], enc.Data)

			u := user.GetUser(sig)
			if u.Username == "" {
				log.Println("User does not exist.")
				fmt.Fprintf(ctx, "User does not exist.")
				return
			}

			roomId, _ := base64.StdEncoding.DecodeString(codedRoomId)
			roomInt, _ := strconv.ParseInt(string(roomId), 16, 64)
			req := RoomRequest{Id: int(roomInt)}
			serveWs(ctx, int(roomInt), req)
		}
	case "send":
		{
			tok := ctx.Request.Header.Peek("Authorization")
			keys := strings.Split(string(tok), " ")
			grant_tok := ledger[keys[0]]
			if grant_tok == "" {
				log.Println("Could not authorize request.")
				fmt.Fprintf(ctx, "Could not authorize request.")
				return
			}

			enc := user.GetLink(keys[0])
			if enc.Data == "" {
				log.Println("Could not authorize request.")
				fmt.Fprintf(ctx, "Could not authorize request.")
				return
			}
			sig := crypto.Decrypt(keys[1], enc.Data)

			u := user.GetUser(sig)
			if u.Username == "" {
				log.Println("User does not exist.")
				fmt.Fprintf(ctx, "User does not exist.")
				return
			}
			bod := ctx.Request.Body()
			sendReqData := SendBody{}
			json.Unmarshal(bod, &sendReqData)
			_, err := db.Exec(fmt.Sprintf(`insert into messages (msg, "From", room) values ('%s','%s','%d')`, sendReqData.Msg, u.Username, sendReqData.RoomId))
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
}
