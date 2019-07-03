package util

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/kataras/iris/websocket"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"jumpserver-automation/log"
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
		log.Logger.Error("gt client err:", err)
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
		log.Logger.Info(i, m)

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
				log.Logger.Info("loginServer:", wsSesion.LoginServer, wsSesion.ID)
				err := wsSesion.Session.SendCommand("exit")
				log.Logger.Error("logout error:", err)
				if err != nil {
					log.Logger.Error("LOGOUT SendCommand error:", err)
					break
				}
				time.Sleep(3 * time.Second)
			}

		} else if ms[0] == "SHELL" {

			log.Logger.Info("shell")
			wsSesion.Session.SendCommand(strings.ReplaceAll(m, "SHELL", ""))

		} else if ms[0] == "LB" {

			ok, msg := OperatLb(m)
			if !ok {
				wsSesion.OUT <- msg
				goto OUT
			} else {
				wsSesion.OUT <- m + " 操作成功"
			}

		} else if ms[0] == "LB-INFO" {

			msg := LbINFO(m)
			wsSesion.OUT <- "LB实例信息：" + msg

		} else if ms[0] == "CHECK" {

			check(wsSesion, ms[1])

		} else if ms[0] == "SLEEP" {

			second, err := time.ParseDuration(ms[1])
			if err != nil {
				log.Logger.Error("parse int error :", err)
			}
			time.Sleep(second)

		} else if ms[0] == "UPLOAD" {

			UploadPath(wsSesion.Client, ms[1], ms[2])
		}
	}
OUT:
}

func check(wsSesion *session.WsSesion, url string) {
	//command := "curl_check=`curl -I -m 10 -o /dev/null -s -w %{http_code} " + url + "`"
	//wsSesion.Session.SendCommand(command)
	wsSesion.Session.CheckURL = url + " is 200ok"
	wsSesion.Session.CheckCommand = "echo `if [[ $curl_check == 200 ]]; then echo \"" + wsSesion.Session.CheckURL + "\"; fi`"
	atomic.StoreInt32(wsSesion.Session.CheckCount, 0)
	wsSesion.OUT <- "开始健康监测\n"
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
			log.Logger.Info(q)
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
				log.Logger.Info("MFA:", MFA)
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
	}
	var err error = nil
	defer func() {
		if e := recover(); e != nil {
			log.Logger.Error("ssh Dial error:", e)
			err = errors.New(fmt.Sprint(e))
		}
	}()
	client, err := ssh.Dial("tcp", conf.Ip+":"+strconv.Itoa(conf.Port), &config)
	if err != nil {
		log.Logger.Error("Failed to dial: " + err.Error())
		return nil, err
	}

	return client, err
}

func NewSession(client *ssh.Client, wsSesion *session.WsSesion) *session.JumpserverSession {
	sshSession, err := client.NewSession()
	CheckErr(err, "create new session")
	in := &session.Input{make(chan string)}
	sshSession.Stdin = in
	out := &session.Output{wsSesion.OUT, wsSesion.Session} //todo
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
		log.Logger.Error(errors.New("unable request pty  " + err.Error()))
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
		log.Logger.Info("session over")
	}(sshSession)
	go func() {
		for {
			select {
			case msg, isOpen := <-wsSesion.OUT:
				if isOpen {
					wsSesion.C.Emit("chat", msg)
					if msg == "close channel session" {
						goto CLOSE
					}
					break
				} else {
					goto CLOSE
				}

			}
		}
	CLOSE:
		log.Logger.Info("close channel session")
	}()

	return jumpserverSession
}

/*func (session *SSHSession) Close() {
	close(session.In.Input)
	for {
		select {
		case <-session.out.out:
		case <-time.After(10 * time.Second):
			{

				close(session.out.out)
				session.Close()
			}

		}

	}
	close(session.out.out)

	session.Close()
}*/

func GetSftp(client *ssh.Client) *sftp.Client {
	sftp, err := sftp.NewClient(client)
	if err != nil {
		log.Logger.Error("GetSftp.error", err)
	}
	return sftp
}

const Separator = "/"

/**
上传目录
*/
func UploadPath(client *ssh.Client, localPath, remotePath string) {
	sftp := GetSftp(client)
	defer sftp.Close()
	uploadPath(localPath, sftp, remotePath)
}

/**
上传目录子方法
*/
func uploadPath(file string, sftp *sftp.Client, remotePath string) {
	inputFile, inputError := os.Open(file)
	if inputError != nil {
		log.Logger.Error(os.Stderr, "File Error: %s\n", inputError)
	}

	fileInfo, err := inputFile.Stat()
	if err != nil {
		log.Logger.Error("fileinfo err:", err)
	}
	defer inputFile.Close()

	if fileInfo.IsDir() {
		//mkdir
		path := remotePath + Separator + fileInfo.Name()
		sftp.Mkdir(path)
		log.Logger.Error(path)

		fileInfo, err := inputFile.Readdir(-1)
		if err == nil {
			for _, f := range fileInfo {

				uploadPath(file+Separator+f.Name(), sftp, path)

			}
		}

	} else {
		//copy file
		log.Logger.Info(remotePath + Separator + fileInfo.Name())
		uploadFile(sftp, file, remotePath+Separator+fileInfo.Name())
	}

}

/**
上传文件
*/
func uploadFile(sftp *sftp.Client, localFile, remotePath string) {
	log.Logger.Info(localFile, ",", remotePath)
	// leave your mark
	inputFile, inputError := os.Open(localFile)
	//fileInfo , err := inputFile.Stat();
	defer inputFile.Close()
	log.Logger.Info(remotePath)
	f, err := sftp.Create(remotePath)

	if err != nil {
		log.Logger.Error("sftp.Create.err", err)
	}

	if inputError != nil {
		log.Logger.Error("File Error: %s\n", inputError)
	}

	fileReader := bufio.NewReader(inputFile)
	counter := 0
	for {
		buf := make([]byte, 20480)
		n, err := fileReader.Read(buf)
		if err == io.EOF {
			break
		}
		counter++
		//fmt.log.Logger.Logger.f("%d,%s", n, string(buf))
		if n == 0 {
			break
		}
		//fmt.log.Logger.Logger.ln(string(buf))
		if _, err := f.Write(buf[0:n]); err != nil {
			log.Logger.Error(err)
		}

	}
	// check it's there
	fi, err := sftp.Lstat(remotePath)
	if err != nil {
		log.Logger.Error("sftp.Lstat.error", err)
	}
	log.Logger.Info(fi)

}

/**
删除文件
*/
func RemoveFile(remoateFile string, sftp *sftp.Client) {
	err := sftp.Remove(remoateFile)
	if err != nil {
		log.Logger.Error(err)
	}
}

/**
查看文件列表
*/
func ListPath(sftp *sftp.Client, remotePath string) {
	//defer sftp.Close()
	// walk a directory
	w := sftp.Walk(remotePath)
	for w.Step() {
		if w.Err() != nil {
			continue
		}
		log.Logger.Info(w.Path())
	}
}

func CheckErr(err error, msg string) {
	if err != nil {
		log.Logger.Error(msg+" err:", err)
	}
}
