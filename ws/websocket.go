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
		params := param(uri)
		id := params["id"]
		sessionId := params["sessionId"]
		wsSesion, ok := cons.Load(sessionId)
		if ok {
			ws := wsSesion.(*session.WsSesion)
			log.Println(ws.ID)
			log.Println("get task", id)
			m := store.Select(id)
			context.Write([]byte(m))
		} else {
			context.Write([]byte("no login"))
		}

	})

	app.Get("/task/execute", func(context context.Context) {
		uri := context.Request().RequestURI
		params := param(uri)
		id := params["id"]
		sessionId := params["sessionId"]
		wsSesion, ok := cons.Load(sessionId)
		if ok {
			ws := wsSesion.(*session.WsSesion)
			log.Println(ws.ID)
			log.Println("execute task :", id)
			m := store.Select(id)
			util.Execute(ws, m)
			context.Write([]byte("ok"))
		} else {
			context.Write([]byte("no login"))
		}
	})

	app.Post("/task/update", func(context context.Context) {
		uri := context.Request().RequestURI
		params := param(uri)
		id := params["id"]
		sessionId := params["sessionId"]
		wsSesion, ok := cons.Load(sessionId)
		if ok {
			ws := wsSesion.(*session.WsSesion)
			log.Println(ws.ID)
			log.Println("update task :", id)
			body, err := ioutil.ReadAll(context.Request().Body)
			if err != nil {
				log.Println(err)
				context.Write([]byte(err.Error()))
			} else {
				store.Update(id, string(body))
				context.Write(body)
			}
		} else {
			context.Write([]byte("no login"))
		}

	})

	app.Get("/task/delete", func(context context.Context) {
		uri := context.Request().RequestURI
		params := param(uri)
		id := params["id"]
		sessionId := params["sessionId"]
		wsSesion, ok := cons.Load(sessionId)
		if ok {
			ws := wsSesion.(*session.WsSesion)
			log.Println(ws.ID)
			log.Println("delete task :", id)
			store.Delete(id)
			context.Write([]byte(id + " deleted ok"))
		} else {
			context.Write([]byte("no login"))
		}
	})

	app.Get("/task/stopExecute", func(context context.Context) {
		uri := context.Request().RequestURI
		params := param(uri)
		id := params["id"]
		sessionId := params["sessionId"]
		wsSesion, ok := cons.Load(sessionId)
		if ok {
			ws := wsSesion.(*session.WsSesion)
			ws.OUT <- "close channel session"
			//close(ws.Session.Out.Out)
			close(ws.Session.In.In)
			ws.Session.Close()
			ws.Session = nil
			context.Write([]byte("stopExecute sessionId:" + id))
			ws.C.Emit("chat", "sessionId:"+id+"is closed")
		} else {
			context.Write([]byte("no login"))
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
		fmt.Printf("%s resive sent: %s\n", c.Context().RemoteAddr(), msg)
		var ws *session.WsSesion
		wsSesion, ok := cons.Load(c.ID())
		if !ok {
			var loginServer uint32 = 1
			wsSesion = &session.WsSesion{ID: c.ID(), OUT: make(chan string, 500), IN: make(chan string), LoginServer: &loginServer}
			cons.Store(c.ID(), wsSesion)
			ws = wsSesion.(*session.WsSesion)
			log.Println("create new session:", c.ID())
		} else {
			ws = wsSesion.(*session.WsSesion)
			log.Println(ws.ID, msg)
		}

		if strings.Contains(msg, "jump") {
			ms := strings.Split(msg, "|")
			go func() {
				ws.C = c
				client, jumpserverSession := util.Jump(ms[1], ms[2], "", 0, c, ws)
				if client == nil {
					log.Println("logon fail")
					return
				}
				ws := wsSesion.(*session.WsSesion)
				ws.Client = client
				ws.Session = jumpserverSession
				jumpserverSession.WebSesion = ws
				c.Emit("chat", "WebSocketId:"+c.ID())

				/*go func() {

					for {
						select {
						case msg := <-ws.OUT:
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
				}()*/
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
			close(ws.Session.Out.Out)
			close(ws.Session.In.In)
			ws.Session.Close()
			ws.Client.Close()
		}
		cons.Delete(c.ID())
		log.Println("delete session:", c.ID())
	})
}

func param(urlStr string) map[string]string {

	//查找字符串的位置
	questionIndex := strings.Index(urlStr, "?")
	//判断是否存在/符号
	cutIndex := strings.Index(urlStr, "/")
	//打散成数组
	rs := []rune(urlStr)
	//用于存储请求的地址切割
	requestSlice := make([]string, 0, 0)
	//用于存储请求的参数字典
	parameterDict := make(map[string]string)
	//请求地址
	requsetStr := ""
	//参数地址
	parameterStr := ""
	//判断是否存在 ?
	if questionIndex != -1 {
		//判断url的长度
		parameterStr = string(rs[questionIndex+1 : len(urlStr)])
		requsetStr = string(rs[0:questionIndex])
		//参数数组
		parameterArray := strings.Split(parameterStr, "&")
		//生成参数字典
		for i := 0; i < len(parameterArray); i++ {
			str := parameterArray[i]
			if len(str) > 0 {
				tem := strings.Split(str, "=")
				if len(tem) > 0 && len(tem) == 1 {
					parameterDict[tem[0]] = ""
				} else if len(tem) > 1 {
					parameterDict[tem[0]] = tem[1]
				}
			}
		}
	} else {
		requsetStr = urlStr
	}

	//判断是否存在 /
	if cutIndex == -1 {
		requestSlice = append(requestSlice, requsetStr)
	} else {
		//按 / 切割
		requestArray := strings.Split(requsetStr, "/")
		for i := 0; i < len(requestArray); i++ {
			//判断第一个字符
			if i == 0 {
				//判断第一个字符串是否为空
				if len(requestArray[i]) != 0 {
					requestSlice = append(requestSlice, requestArray[i])
				}
			} else {
				requestSlice = append(requestSlice, requestArray[i])
			}
		}

	}

	//log.Println("参数url:")
	//log.Println(parameterStr)
	/*log.Println("请求url:")
	log.Println(requsetStr)
	log.Println("参数字典:")
	log.Println(parameterDict)
	log.Println("请求的字典：")
	log.Println(requestSlice)*/

	return parameterDict
}
