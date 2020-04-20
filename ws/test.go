package ws

import (
	"encoding/json"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"io/ioutil"
	"jumpserver-automation/logs"
	"jumpserver-automation/session"
	"jumpserver-automation/store"
	"jumpserver-automation/util"
	"strings"
)

func AddTestFunction(app *iris.Application) {
	app.Get("/sshKeys", func(context context.Context) {
		fileInfos := util.GetDirList(util.KeyPath)
		m := make(map[string]string, 0)
		for _, info := range fileInfos {
			m[info.Name()] = info.Name()
		}
		b, _ := json.Marshal(m)
		context.Write(b)
	})

	app.Get("/addShhServer", func(context context.Context) {
		uri := context.Request().RequestURI
		params := param(uri)
		name := params["name"]
		ip := params["ip"]
		port := params["port"]
		user := params["user"]
		password := params["password"]
		key := params["key"]
		logs.Logger.Info(name, ip, port, password, key)
		if name == "" || ip == "" || port == "" || (password == "" && key == "") {
			context.Write([]byte("param error"))
			return
		}
		config := util.SSHConfig{Ip: ip, KeyPath: key, Port: port, Password: password, User: user}
		if config.KeyPath != "" {
			config.KeyPath = util.KeyPath + config.KeyPath
		}
		server := util.SSHServer{name, config}
		k := "SSH_NAME_" + name + "_IP_" + ip + "_"
		bs, err := json.Marshal(server)
		if err != nil {
			logs.Logger.Error(err)
		}
		store.UpdateWithBucket(k, string(bs), store.TestBucket)
		m := store.SelectAllWithBucket(store.TestBucket)
		newM := make(map[string]string, 0)
		for ks, _ := range m {
			if strings.Contains(ks, "SSH_NAME_") {
				newM[name+"-"+ip] = ip
			}
		}
		b, _ := json.Marshal(newM)
		context.Write(b)
	})

	app.Get("/deleteShhServer", func(context context.Context) {
		uri := context.Request().RequestURI
		params := param(uri)
		name := params["name"]
		ip := params["ip"]
		logs.Logger.Info(name, ip)
		if name == "" || ip == "" {
			context.Write([]byte("param error"))
			return
		}
		k := "SSH_NAME_" + name + "_IP_" + ip + "_"
		store.DeleteWithBucket(k, store.TestBucket)
		m := store.SelectAllWithBucket(store.TestBucket)
		newM := make(map[string]string, 0)
		for ks, _ := range m {
			if strings.Contains(ks, "SSH_NAME<") {
				newM[name+"-"+ip] = ip
			}
		}
		b, _ := json.Marshal(newM)
		context.Write(b)
	})

	app.Get("/listShhServer", func(context context.Context) {
		m := store.SelectAllWithBucket(store.TestBucket)
		newM := make(map[string]string, 0)
		for k, v := range m {
			if strings.Contains(k, "SSH_NAME_") {
				var server util.SSHServer
				json.Unmarshal([]byte(v), &server)
				newM[server.Name+"-"+server.Config.Ip] = k
			}
		}
		b, _ := json.Marshal(newM)
		context.Write(b)
	})

	app.Get("/listTestTaskGroup", func(context context.Context) {
		m := store.SelectAllWithBucket(store.TestBucket)
		newM := make(map[string]string, 0)
		for k, v := range m {
			if strings.Contains(k, "JOB_GROUP《") && !strings.Contains(k, "ARG") {
				logs.Logger.Info(k, v)
				values := getJobGroupAndJobName(k)
				newM[values[0]] = values[1]
			}
		}
		b, _ := json.Marshal(newM)
		context.Write(b)
	})

	app.Get("/listTestTask", func(context context.Context) {
		m := store.SelectAllWithBucket(store.TestBucket)
		newM := make(map[string]string, 0)
		params := param(context.Request().RequestURI)
		group := params["group"]
		logs.Logger.Info("group:", group)
		for k, v := range m {
			if strings.Contains(k, "JOB_GROUP《"+group) && !strings.Contains(k, "ARG") {
				values := getJobGroupAndJobName(k)
				newM[values[1]] = v
			}
		}
		b, _ := json.Marshal(newM)
		context.Write(b)
	})

	app.Post("/addTestTask", func(context context.Context) {
		uri := context.Request().RequestURI
		params := param(uri)
		name := params["name"]
		group := params["group"]
		args := params["args"]
		server := params["server"]
		if args == "" {
			args = "SSH_Server_Key:" + server
		} else {
			args = args + ";SSH_Server_Key:" + server
		}
		body, err := ioutil.ReadAll(context.Request().Body)
		logs.Logger.Info("add new test task :", name, group, args, server, string(body))
		if err != nil {
			logs.Logger.Error(err)
			context.Write([]byte(err.Error()))
		} else {
			store.UpdateWithBucket(buildJobFullName(group, name), string(body), store.TestBucket)
			store.UpdateWithBucket(buildJobArgsFullName(group, name), args, store.TestBucket)
			context.Write(body)
		}
	})

	app.Post("/updateTestTask", func(context context.Context) {
		uri := context.Request().RequestURI
		params := param(uri)
		name := params["name"]
		group := params["group"]
		args := params["args"]
		if args != "" {
			store.UpdateWithBucket(buildJobArgsFullName(group, name), args, store.TestBucket)
		}
		body, err := ioutil.ReadAll(context.Request().Body)
		logs.Logger.Info("updat test task :", name, group, string(body))
		if err != nil {
			logs.Logger.Error(err)
			context.Write([]byte(err.Error()))
		} else {
			store.UpdateWithBucket(buildJobFullName(group, name), string(body), store.TestBucket)
			//store.UpdateWithBucket(buildJobArgsFullName(group,name),args , store.TestBucket)
			context.Write(body)
		}
	})

	app.Get("/testTask", func(context context.Context) {
		uri := context.Request().RequestURI
		params := param(uri)
		name := params["name"]
		group := params["group"]
		logs.Logger.Info("get task", name, group)
		args := store.SelectWithBucket(buildJobArgsFullName(group, name), store.TestBucket)
		body := store.SelectWithBucket(buildJobFullName(group, name), store.TestBucket)
		result := make(map[string]string)
		result["Id"] = name
		result["Task"] = body
		result["Args"] = args
		bs, err := json.Marshal(result)
		util.CheckErr(err, "json.Marshal error:")
		context.Write(bs)
	})

	app.Get("/deleteTestTask", func(context context.Context) {
		uri := context.Request().RequestURI
		params := param(uri)
		name := params["name"]
		group := params["group"]
		logs.Logger.Info("delete test task :", name, group)
		logs.Logger.Info(store.SelectWithBucket(buildJobFullName(group, name), store.TestBucket))
		store.DeleteWithBucket(buildJobFullName(group, name), store.TestBucket)
		logs.Logger.Info(store.SelectWithBucket(buildJobArgsFullName(group, name), store.TestBucket))
		store.DeleteWithBucket(buildJobArgsFullName(group, name), store.TestBucket)
		context.Write([]byte(name + " deleted ok"))
	})

	app.Get("/runTestTask", func(context context.Context) {
		uri := context.Request().RequestURI
		params := param(uri)
		name := params["name"]
		group := params["group"]
		sessionId := params["sessionId"]
		wsSesion, ok := cons.Load(sessionId)
		if ok {
			var server util.SSHServer
			args := store.SelectWithBucket(buildJobArgsFullName(group, name), store.TestBucket)
			logs.Logger.Info("execute test task :", name, group, args)
			body := store.SelectWithBucket(buildJobFullName(group, name), store.TestBucket)
			logs.Logger.Info(body)
			argsArray := strings.Split(args, ";")
			logs.Logger.Info(argsArray)
			for _, arg := range argsArray {
				kv := strings.Split(arg, ":")
				if len(kv) == 2 && kv[0] != "" && kv[1] != "" {
					body = strings.ReplaceAll(body, "${"+kv[0]+"}", kv[1])
				}
				if kv[0] == "SSH_Server_Key" {
					s := store.SelectWithBucket(kv[1], store.TestBucket)
					logs.Logger.Info("server:", s)
					err := json.Unmarshal([]byte(s), &server)
					if err != nil {
						logs.Logger.Error(err)
					}
				}
			}
			ws := wsSesion.(*session.WsSesion)
			util.ExecuteWithServer(ws, body, server)
			context.Write([]byte("执行成功"))
		}

	})
}
