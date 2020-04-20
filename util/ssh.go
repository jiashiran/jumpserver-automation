package util

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/kataras/iris/websocket"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"jumpserver-automation/logs"
	"jumpserver-automation/session"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

func Jump(user string, password string, ip string, port int, c websocket.Connection, wsSesion *session.WsSesion) (*ssh.Client, *session.JumpserverSession) {
	client, err := NewJumpserverClient(&JumpserverConfig{
		User:     user,
		Password: password,
		Ip:       ip,
		Port:     port,
	}, c, wsSesion)
	if err != nil {
		logs.Logger.Error("gt client err:", err)
		return nil, nil
	}

	jumpserverSession := NewSession(client, wsSesion)

	return client, jumpserverSession
}

/*
函数名：delete_extra_space(s string) string
功  能:删除字符串中多余的空格(含tab)，有多个空格时，仅保留一个空格，同时将字符串中的tab换为空格
参  数:s string:原始字符串
返回值:string:删除多余空格后的字符串
创建时间:2018年12月3日
修订信息:
*/
func delete_extra_space(s string) string {
	//删除字符串中的多余空格，有多个空格时，仅保留一个空格
	s1 := strings.Replace(s, "	", " ", -1)       //替换tab为空格
	regstr := "\\s{2,}"                          //两个及两个以上空格的正则表达式
	reg, _ := regexp.Compile(regstr)             //编译正则表达式
	s2 := make([]byte, len(s1))                  //定义字符数组切片
	copy(s2, s1)                                 //将字符串复制到切片
	spc_index := reg.FindStringIndex(string(s2)) //在字符串中搜索
	for len(spc_index) > 0 {                     //找到适配项
		s2 = append(s2[:spc_index[0]+1], s2[spc_index[1]:]...) //删除多余空格
		spc_index = reg.FindStringIndex(string(s2))            //继续在字符串中搜索
	}
	return string(s2)
}

func Execute(wsSesion *session.WsSesion, task string) {

	if wsSesion.Client != nil && wsSesion.Session == nil {
		wsSesion.Session = NewSession(wsSesion.Client, wsSesion)
	}

	commands := strings.Split(task, "\n")

	for i, m := range commands {
		m = delete_extra_space(m)
		logs.Logger.Info(i, m)

		if strings.Contains(m, "//") {
			//log.Logger.log.Logger.Logger.ln("注释：",m)
			continue
		}

		ms := strings.Split(m, " ")
		for i, m := range ms {
			ms[i] = strings.Replace(m, " ", "", -1)
		}
		if ms[0] == "LOGIN" {
			wsSesion.Session.SendCommand(ms[1])

		} else if ms[0] == "LOGOUT" {

			for atomic.LoadUint32(wsSesion.LoginServer) > 0 {
				logs.Logger.Info("loginServer:", wsSesion.LoginServer, wsSesion.ID)
				err := wsSesion.Session.SendCommand("exit")
				logs.Logger.Error("logout error:", err)
				if err != nil {
					logs.Logger.Error("LOGOUT SendCommand error:", err)
					break
				}
				time.Sleep(3 * time.Second)
			}

		} else if ms[0] == "SHELL" {

			logs.Logger.Info("shell")
			wsSesion.Session.SendCommand(strings.ReplaceAll(m, "SHELL", ""))

		} else if ms[0] == "LB" {
			ok, msg := OperatLb(m)
			if !ok {
				wsSesion.F.WriteString(msg + "\n")
				wsSesion.F.Flush()
				goto OUT
			} else {
				wsSesion.F.WriteString(m + " 操作成功\n")
				wsSesion.F.Flush()
			}

		} else if ms[0] == "LB-INFO" {

			msg := LbINFO(m)
			wsSesion.F.WriteString("LB实例信息：" + msg + "\n")
			wsSesion.F.Flush()

		} else if ms[0] == "CHECK" {

			check(wsSesion, ms[1])

		} else if ms[0] == "SLEEP" {

			second, err := time.ParseDuration(ms[1])
			if err != nil {
				logs.Logger.Error("parse int error :", err)
			}
			time.Sleep(second)

		} else if ms[0] == "UPLOAD" {

			UploadPath(wsSesion.Client, ms[1], ms[2])
		}
	}
OUT:
}

func ExecuteWithServer(wsSesion *session.WsSesion, task string, server SSHServer) {
	client, err := GetSSHClient(&server.Config)
	if err != nil {
		logs.Logger.Error(err)
	}
	defer client.Close()
	session, in, out, echo := NewSessionWithChan(client, wsSesion)
	defer session.Close()
	defer close(in)
	defer close(out)
	commands := strings.Split(task, "\n")
	start := time.Now().UnixNano()
	for i, m := range commands {
		m = delete_extra_space(m)
		logs.Logger.Info(i, m)
		if strings.Contains(m, "//") {
			continue
		}
		ms := strings.Split(m, " ")
		for i, m := range ms {
			ms[i] = strings.Replace(m, " ", "", -1)
		}
		if ms[0] == "SHELL" {
			//log.Logger.Info("shell")
			//wsSesion.Session.SendCommand(strings.ReplaceAll(m, "SHELL", ""))
			//ExecuteShellWithChan(client, strings.ReplaceAll(m, "SHELL", ""), wsSesion)
			command := strings.ReplaceAll(m, "SHELL", "")
			wsSesion.C.Emit("chat", command+"\n")
			in <- command
		} else if ms[0] == "SLEEP" {
			second, err := time.ParseDuration(ms[1])
			if err != nil {
				logs.Logger.Error("parse int error :", err)
			}
			time.Sleep(second)
		}
	}
	end := time.Now().UnixNano()
	logs.Logger.Info(end, start)
	t := 3000000000 + (end - start)
	d, _ := time.ParseDuration(fmt.Sprint(t) + "ns")
	echo(d)
	logs.Logger.Info("execute over")
}

func check(wsSesion *session.WsSesion, url string) {
	wsSesion.Session.CheckURL = url + " is 200ok"
	wsSesion.Session.CheckCommand = "echo `if [[ $curl_check == 200 ]]; then echo \"" + wsSesion.Session.CheckURL + "\"; fi`"
	atomic.StoreInt32(wsSesion.Session.CheckCount, 0)
	wsSesion.F.WriteString("开始健康监测\n")
	for atomic.StoreUint32(wsSesion.Session.Health, 0); atomic.LoadUint32(wsSesion.Session.Health) == 0; {
		//log.Logger.log.Logger.Logger.ln("check url:", url)
		wsSesion.Session.SendCommand("curl -I -m 10 -s " + url)
		wsSesion.Session.SendCommand(wsSesion.Session.CheckCommand)
		time.Sleep(10 * time.Second)
	}
}

type JumpserverConfig struct {
	User     string
	Password string
	Ip       string
	Port     int
}

func NewJumpserverClient(conf *JumpserverConfig, c websocket.Connection, wsSesion *session.WsSesion) (*ssh.Client, error) {
	var config ssh.ClientConfig
	var authMethods []ssh.AuthMethod
	authMethods = append(authMethods, ssh.Password(conf.Password))
	authMethods = append(authMethods, ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
		answers := make([]string, 0, len(questions))
		for i, q := range questions {
			logs.Logger.Info(q)
			c.Emit("chat", q)
			if echos[i] {
				/*scan := bufio.NewScanner(os.Stdin)
				if scan.Scan() {
					answers = append(answers, scan.Text())
				}
				err := scan.Err()
				if err != nil {
					return nil, err
				}*/
				MFA := <-wsSesion.IN
				logs.Logger.Info("MFA:", MFA)
				answers = append(answers, MFA)
			} else {
				b, err := terminal.ReadPassword(int(syscall.Stdin))
				if err != nil {
					return nil, err
				}
				answers = append(answers, string(b))
			}
		}
		return answers, nil
	}))
	config = ssh.ClientConfig{
		User: conf.User,
		Auth: authMethods,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: 10 * time.Second,
	}
	var err error = nil
	defer func() {
		if e := recover(); e != nil {
			logs.Logger.Error("ssh Dial error:", e)
			err = errors.New(fmt.Sprint(e))
		}
	}()
	client, err := ssh.Dial("tcp", conf.Ip+":"+strconv.Itoa(conf.Port), &config)
	if err != nil {
		logs.Logger.Error("Failed to dial: " + err.Error())
		return nil, err
	}

	return client, err
}

func NewSession(client *ssh.Client, wsSesion *session.WsSesion) *session.JumpserverSession {
	sshSession, err := client.NewSession()
	CheckErr(err, "create new session")
	in := &session.Input{make(chan string)}
	sshSession.Stdin = in
	out := &session.Output{wsSesion.Session} //&session.Output{wsSesion.OUT, wsSesion.Session} //todo
	sshSession.Stdout = out
	sshSession.Stderr = os.Stderr
	sshSession.Setenv("LANG", "zh_CN.UTF-8")
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	err = sshSession.RequestPty("xterm", 100, 200, modes)
	if err != nil {
		logs.Logger.Error(errors.New("unable request pty  " + err.Error()))
	}
	var checkCount int32 = 0
	var health uint32 = 0
	jumpserverSession := &session.JumpserverSession{sshSession, in, out, &health, "", wsSesion, &checkCount, ""}
	out.JumpserverSession = jumpserverSession
	go func(s *ssh.Session) {
		err = s.Shell()
		CheckErr(err, "session shell")
		err = s.Wait()
		CheckErr(err, "session wait")
		logs.Logger.Info("session over")
	}(sshSession)
	go func(ws *session.WsSesion) {
		buf := bufio.NewReader(ws.ReadLog)
		for {
			line, err := buf.ReadString('\n')
			//line = strings.Replace(line, "\a", "", -1)
			line = strings.TrimSpace(line)
			if err != nil && !strings.Contains(fmt.Sprint(err), "file already closed") {
				//fmt.Println("buf.ReadString:",line,err)
				time.Sleep(2 * time.Second)
				//ws.F.Flush()
				continue
			}
			ws.C.Emit("chat", line)
			if strings.Contains(line, "close channel session") || strings.Contains(fmt.Sprint(err), "file already closed") {
				goto CLOSE
			}
		}
	CLOSE:
		logs.Logger.Info("close channel session")

	}((wsSesion))

	return jumpserverSession
}

func CheckErr(err error, msg string) {
	if err != nil {
		logs.Logger.Error(msg+" err:", err)
	}
}
