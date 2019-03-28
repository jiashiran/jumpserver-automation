package ws

import (
	"encoding/json"
	"fmt"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/middleware/recover"
	"github.com/kataras/iris/websocket"
	"io/ioutil"
	"jumpserver-automation/session"
	"jumpserver-automation/store"
	"jumpserver-automation/util"
	"log"
	"strings"
	"sync"
)

var cons sync.Map


func Service() {
	app := iris.New()
	app.Use(recover.New())
	app.Get("/", func(ctx iris.Context) {
		ctx.ServeFile("static/websockets.html", false) // second parameter: enable gzip?
	})

	app.Get("/tasks/list", func(context context.Context) {
		m := store.SelectAll()
		b, _ := json.Marshal(m)
		context.Write(b)
	})

	app.Get("/task", func(context context.Context) {
		uri := context.Request().RequestURI
		id := strings.Split(uri, "?")[1]
		id = strings.Split(id, "=")[1]
		log.Println("get task", id)
		m := store.Select(id)
		context.Write([]byte(m))
	})

	app.Get("/task/execute", func(context context.Context) {
		uri := context.Request().RequestURI
		id := strings.Split(uri, "?")[1]
		id = strings.Split(id, "=")[1]
		log.Println("execute task :", id)
		m := store.Select(id)
		_ = m
		//util.Execute(m)
		context.Write([]byte("ok"))
	})

	app.Post("/task/update", func(context context.Context) {
		uri := context.Request().RequestURI
		log.Println(uri)
		id := strings.Split(uri, "?")[1]
		id = strings.Split(id, "=")[1]
		log.Println("update task:", id)
		body, err := ioutil.ReadAll(context.Request().Body)
		if err != nil {
			log.Println(err)
			context.Write([]byte(err.Error()))
		} else {
			store.Update(id, string(body))
			context.Write(body)
		}

	})

	app.Get("/task/delete", func(context context.Context) {
		uri := context.Request().RequestURI
		id := strings.Split(uri, "?")[1]
		id = strings.Split(id, "=")[1]
		log.Println("delete task", id)
		store.Delete(id)
		context.Write([]byte(id + " deleted ok"))
	})

	app.Get("/task/stopExecute", func(context context.Context) {
		uri := context.Request().RequestURI
		id := strings.Split(uri, "?")[1]
		id = strings.Split(id, "=")[1]
		log.Println("stopExecute sessionId", id)
		wsSesion, ok := cons.Load(id)
		if ok {
			ws := wsSesion.(session.WsSesion)
			ws.Session.Close()
			context.Write([]byte("stopExecute sessionId:" + id))
		}else {
			context.Write([]byte("no stopExecute sessionId:" + id))
		}
	})

	setupWebsocket(app)

	// x2
	// http://localhost:8080
	// http://localhost:8080
	// write something, press submit, see the result.
	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
	defer func() {
		store.Close()
	}()
}

func setupWebsocket(app *iris.Application) {
	// create our echo websocket server
	ws := websocket.New(websocket.Config{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	})
	ws.OnConnection(handleConnection)

	// register the server on an endpoint.
	// see the inline javascript code in the websockets.html,
	// this endpoint is used to connect to the server.
	app.Get("/echo", ws.Handler())
	// serve the javascript built'n client-side library,
	// see websockets.html script tags, this path is used.
	app.Any("/iris-ws.js", websocket.ClientHandler())
}

func handleConnection(c websocket.Connection) {
	// Read events from browser
	c.On("chat", func(msg string) {
		// Print the message to the console, c.Context() is the iris's http context.
		fmt.Printf("%s sent: %s\n", c.Context().RemoteAddr(), msg)
		var ws session.WsSesion
		wsSesion, ok := cons.Load(c.ID())
		if !ok {
			wsSesion = session.WsSesion{ID: c.ID(),OUT:make(chan string, 100),IN:make(chan string),LoginServer:false}
			cons.Store(c.ID(), wsSesion)
			ws = session.WsSesion{}
			log.Println("create new session:", c.ID())
		} else {
			ws = wsSesion.(session.WsSesion)
			log.Println(ws.ID, msg)
		}

		if strings.Contains(msg, "jump") {
			ms := strings.Split(msg, "|")
			go func() {
				client,jumpserverSession:=util.Jump(ms[1], ms[2], "", 0, c,ws)
				ws := wsSesion.(session.WsSesion)
				ws.Client = client
				ws.Session = jumpserverSession
				jumpserverSession.WebSesion = ws
				c.Emit("chat", "WebSocketId:"+c.ID())
			}()
			go func() {

				for {
					select {
					case msg := <- ws.OUT:
						{
							c.Emit("chat", msg)
							if msg == "close channel session" {
								goto CLOSE
							}
							break
						}

					}
				}
			CLOSE:
				log.Println("close channel session")
			}()
		} else if strings.Contains(msg, "[MFA auth]: ") {
			ws.IN <- strings.Replace(msg, "[MFA auth]: ", "", -1)
		}
		// Write message back to the client message owner with:

		// Write message to all except this client with:
		//c.To(websocket.Broadcast).Emit("chat","aaaaaaaaa")
	})

	c.OnDisconnect(func() {
		wsSesion, ok := cons.Load(c.ID())
		if ok {
			ws := wsSesion.(session.WsSesion)
			ws.Session.Close()
			ws.Client.Close()
		}
		cons.Delete(c.ID())
		log.Println("delete session:", c.ID())
	})
}
