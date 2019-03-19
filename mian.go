package main

import (
	"fmt"
	"github.com/kataras/iris"
	"github.com/kataras/iris/websocket"
	"jumpserver-automation/util"
	"log"
	"time"
)




func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.ServeFile("static/websockets.html", false) // second parameter: enable gzip?
	})

	setupWebsocket(app)

	// x2
	// http://localhost:8080
	// http://localhost:8080
	// write something, press submit, see the result.
	app.Run(iris.Addr(":8080"))
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

		log.Println(msg)
		if "jump" == msg{
			go func() {
				jump(c)
			}()
			go func() {

				for  {
					select {
					case msg := <- util.OUT:{
						c.Emit("chat", msg)
						if msg == "close channel session"{
							goto CLOSE
						}
						break
					}

					}
				}
				CLOSE:
				log.Println("close channel session")
			}()
		}/*else if strings.Contains(msg,"token:") {
			util.IN <- strings.Replace(msg,"token:","",-1)
		}*/
		// Write message back to the client message owner with:


		// Write message to all except this client with:
		//c.To(websocket.Broadcast).Emit("chat","aaaaaaaaa")
	})
}

func jump(c websocket.Connection)  {

	client,err := util.NewJumpserverClient(&util.JumpserverConfig{
		User:"",
		Password:"",
		Ip:"",
		Port:62012,
	},c)
	if err != nil{
		log.Fatal("gt client err:",err)
	}

	session := util.NewSession(client)

	session.SendCommand("g")

	time.Sleep( 3 * time.Second)

	session.SendCommand("g24")

	time.Sleep( 3 * time.Second)

	session.SendCommand("1")

	time.Sleep( 3 * time.Second)

	session.SendCommand("sudo su -")

	time.Sleep( 3 * time.Second)

	session.SendCommand("free")

	time.Sleep( 3 * time.Second)

	session.Close()
}
