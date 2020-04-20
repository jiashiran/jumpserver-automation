package ws

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	iris_recover "github.com/kataras/iris/middleware/recover"
	"github.com/kataras/iris/websocket"
	"io/ioutil"
	"jumpserver-automation/logs"
	"jumpserver-automation/session"
	"jumpserver-automation/store"
	"jumpserver-automation/util"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"sync"
)

var cons sync.Map

func Service() {
	go func() {
		logs.Logger.Info(http.ListenAndServe(":6060", nil))
	}()
	app := iris.New()
	app.Use(iris_recover.New())
	app.Get("/", func(ctx iris.Context) {
		ctx.ServeFile("/Users/jiashiran/go/src/jumpserver-automation/static/websockets.html", true) // second parameter: enable gzip?
	})
	app.Get("/help", func(ctx iris.Context) {
		ctx.ServeFile("/Users/jiashiran/go/src/jumpserver-automation/static/help.html", true) // second parameter: enable gzip?
	})

	app.Get("/taskGroups/list", func(context context.Context) {
		m := store.SelectAll()
		newM := make(map[string]string, 0)

		for k, _ := range m {
			values := getJobGroupAndJobName(k)
			newM[values[0]] = values[1]
		}

		b, _ := json.Marshal(newM)
		context.Write(b)
	})

	app.Get("/tasks/list", func(context context.Context) {
		m := store.SelectAll()
		newM := make(map[string]string, 0)
		params := param(context.Request().RequestURI)
		group := params["group"]
		logs.Logger.Info("group:", group)
		for k, v := range m {
			if strings.Contains(k, "JOB_GROUP《"+group) {
				values := getJobGroupAndJobName(k)
				newM[values[1]] = v
			}
		}
		b, _ := json.Marshal(newM)
		context.Write(b)
	})

	app.Get("/task", func(context context.Context) {
		uri := context.Request().RequestURI
		params := param(uri)
		id := params["id"]
		group := params["group"]
		sessionId := params["sessionId"]
		wsSesion, ok := cons.Load(sessionId)
		if ok {
			ws := wsSesion.(*session.WsSesion)
			logs.Logger.Info("get task", id, group, ws.ID)
			m := store.Select(buildJobFullName(group, id))
			args := store.SelectArgs(buildJobArgsFullName(group, id))
			result := make(map[string]string)
			result["Id"] = id
			result["Task"] = m
			result["Args"] = args
			bs, err := json.Marshal(result)
			util.CheckErr(err, "json.Marshal error:")
			context.Write(bs)
		} else {
			context.Write([]byte("no login"))
		}

	})

	app.Get("/task/execute", func(context context.Context) {
		uri := context.Request().RequestURI
		params := param(uri)
		id := params["id"]
		group := params["group"]
		sessionId := params["sessionId"]
		wsSesion, ok := cons.Load(sessionId)
		if ok {
			ws := wsSesion.(*session.WsSesion)
			logs.Logger.Info(ws.ID)
			logs.Logger.Info("execute task :", id)
			m := store.Select(buildJobFullName(group, id))
			args := store.SelectArgs(buildJobArgsFullName(group, id))
			if args != "" {
				argsArray := strings.Split(args, ";")
				for _, arg := range argsArray {
					kv := strings.Split(arg, ":")
					if len(kv) == 2 && kv[0] != "" && kv[1] != "" {
						m = strings.ReplaceAll(m, "${"+kv[0]+"}", kv[1])
					}
				}
			}
			util.Execute(ws, m)
			context.Write([]byte("执行成功"))
		} else {
			context.Write([]byte("no login"))
		}
	})

	app.Post("/task/update", func(context context.Context) {
		uri := context.Request().RequestURI
		params := param(uri)
		id := params["id"]
		group := params["group"]
		args := params["args"]
		sessionId := params["sessionId"]
		wsSesion, ok := cons.Load(sessionId)
		if ok {
			ws := wsSesion.(*session.WsSesion)
			logs.Logger.Info(ws.ID)
			logs.Logger.Info("update task :", id)
			body, err := ioutil.ReadAll(context.Request().Body)
			if err != nil {
				logs.Logger.Error(err)
				context.Write([]byte(err.Error()))
			} else {
				store.Update(buildJobFullName(group, id), string(body))
				store.UpdateArgs(buildJobArgsFullName(group, id), args)
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
		group := params["group"]
		sessionId := params["sessionId"]
		wsSesion, ok := cons.Load(sessionId)
		if ok {
			ws := wsSesion.(*session.WsSesion)
			logs.Logger.Info(ws.ID)
			logs.Logger.Info("delete task :", id)
			store.Delete(buildJobFullName(group, id))
			store.DeleteArgs(buildJobArgsFullName(group, id))
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
			ws.F.WriteString("close channel session\n")
			close(ws.Session.In.In)
			ws.Session.Close()
			ws.Session = nil
			context.Write([]byte("stopExecute sessionId:" + id))
			ws.C.Emit("chat", "sessionId:"+id+"is closed")
		} else {
			context.Write([]byte("no login"))
		}
	})
	AddTestFunction(app)
	setupWebsocket(app)
	// x2
	// http://localhost:8080
	// http://localhost:8080
	// write something, press submit, see the result.
	app.Run(iris.Addr(":8089"), iris.WithoutServerError(iris.ErrServerClosed))
	defer func() {
		store.Close()
	}()
}

func buildJobFullName(jobGroup, jobName string) string {
	return "JOB_GROUP《" + jobGroup + "》JOB_NAME《" + jobName + "》_END"
}

func getJobGroupAndJobName(key string) []string {
	key = strings.Replace(key, "JOB_GROUP《", "", -1)
	key = strings.Replace(key, "》_END", "", -1)
	values := strings.Split(key, "》JOB_NAME《")
	return values
}

func buildJobArgsFullName(jobGroup, jobName string) string {
	return "JOB_GROUP《" + jobGroup + "》JOB_NAME《" + jobName + "》_ARGS_END"
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
		logs.Logger.Infof("%s resive sent: %s\n", c.Context().RemoteAddr(), msg)
		var ws *session.WsSesion
		wsSesion, ok := cons.Load(c.ID())
		if !ok {
			var loginServer uint32 = 1
			logFile, err := os.Create("/usr/local/db/logs/" + c.ID() + ".log")
			if err != nil {
				fmt.Println("os.Create err :", err)
			}
			bufioWriter := bufio.NewWriterSize(logFile, 4096*100)
			logger := log.New(bufioWriter, "logger: ", log.Lshortfile)
			logFileRead, _ := os.Open("/usr/local/db/logs/" + c.ID() + ".log")
			wsSesion = &session.WsSesion{
				ID:          c.ID(),
				LogFile:     logFile,
				LogFileRead: logFileRead,
				F:           bufioWriter,
				ReadLog:     bufio.NewReader(logFileRead),
				IN:          make(chan string),
				LoginServer: &loginServer,
				Logger:      logger,
			}
			cons.Store(c.ID(), wsSesion)
			ws = wsSesion.(*session.WsSesion)
			ws.C = c
			logs.Logger.Info("create new session:", c.ID())
		} else {
			ws = wsSesion.(*session.WsSesion)
			logs.Logger.Info(ws.ID, msg)
		}
		if strings.Contains(msg, "test-test-test") {
			logs.Logger.Info("ttt-eee-sss-ttt")
			c.Emit("chat", "test-test-test:"+c.ID())
		}

		if strings.Contains(msg, "jump") {
			msg = strings.ReplaceAll(msg, "\n", "")
			ms := strings.Split(msg, "|")
			go func() {
				ws.C = c
				jumpserverIp := ""
				jumpserverPort := 0
				if len(ms) > 3 && ms[3] == "Clink" {
					jumpserverIp = ""
					jumpserverPort = 0
				}
				client, jumpserverSession := util.Jump(ms[1], ms[2], jumpserverIp, jumpserverPort, c, ws)
				if client == nil {
					logs.Logger.Error("logon fail")
					return
				}
				ws := wsSesion.(*session.WsSesion)
				ws.Client = client
				ws.Session = jumpserverSession
				jumpserverSession.WebSesion = ws
				c.Emit("chat", "WebSocketId:"+c.ID())

			}()

		} else if strings.Contains(msg, "[MFA auth]: ") {
			ws.IN <- strings.Replace(msg, "[MFA auth]: ", "", -1)
		}

		// Write message back to the client message owner with:

		// Write message to all except this client with:
	})

	c.OnDisconnect(func() {
		wsSesion, ok := cons.Load(c.ID())
		if ok {
			defer func() {
				if err := recover(); err != nil {
					logs.Logger.Error("close wsSesion error", err)
				}
			}()
			ws := wsSesion.(*session.WsSesion)
			ws.F.WriteString("close channel session\n")
			close(ws.Session.In.In)
			ws.Session.Close()
			ws.Client.Close()
			ws.Session = nil
			ws.Client = nil
			ws.F = nil
			ws.LogFile.Close()
			ws.LogFileRead.Close()
			ws.Logger = nil
			os.Remove("/usr/local/db/logs/" + c.ID() + ".log")
		}
		cons.Delete(c.ID())
		logs.Logger.Info("delete session:", c.ID())
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

	//log.Logger.Println("参数url:")
	//log.Logger.Println(parameterStr)
	/*log.Logger.Println("请求url:")
	log.Logger.Println(requsetStr)
	log.Logger.Println("参数字典:")
	log.Logger.Println(parameterDict)
	log.Logger.Println("请求的字典：")
	log.Logger.Println(requestSlice)*/

	return parameterDict
}
