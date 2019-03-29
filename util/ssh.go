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
	"jumpserver-automation/session"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (

	//jumpserverSession *JumpserverSession
)

func Jump(user string, password string, ip string, port int, c websocket.Connection,wsSesion *session.WsSesion) (*ssh.Client,*session.JumpserverSession) {
	client, err := NewJumpserverClient(&JumpserverConfig{
		User:     user,
		Password: password,
		Ip:       ip,
		Port:     port,
	}, c,wsSesion)
	if err != nil {
		log.Fatal("gt client err:", err)
	}

	jumpserverSession := NewSession(client,wsSesion)

	return client,jumpserverSession
}

func Execute(wsSesion session.WsSesion,task string) {
	commands := strings.Split(task, "\n")

	for i, m := range commands {
		log.Println(i, m)
		ms := strings.Split(m, " ")
		if ms[0] == "LOGIN" {
			wsSesion.Session.SendCommand(ms[1])
			time.Sleep(3 * time.Second)
		} else if ms[0] == "LOGOUT" {
			for wsSesion.LoginServer == true {
				log.Println("loginServer:", wsSesion.LoginServer)
				wsSesion.Session.SendCommand("exit")
				time.Sleep(3 * time.Second)
			}
		} else if ms[0] == "SHELL" {
			log.Println("shell")
			wsSesion.Session.SendCommand(strings.ReplaceAll(m, "SHELL", ""))
			time.Sleep(3 * time.Second)
			/*err := jumpserverSession.Signal(ssh.SIGINT)
			if err != nil {
				log.Println("SIGINT:", err)
			}*/
		} else if ms[0] == "LB" {

		} else if ms[0] == "CHECK" {
			check(wsSesion,ms[1])
		}
	}
}

func check(wsSesion session.WsSesion,url string) {
	command := "curl_check=`curl -I -m 10 -o /dev/null -s -w %{http_code} "+url+"`"
	wsSesion.Session.SendCommand(command)
	wsSesion.Session.CheckURL = url + " is 200ok"
	wsSesion.Session.CheckCommand = "echo `if [ $curl_check == 200 ]; then echo \""+wsSesion.Session.CheckURL+"\"; fi`"
	for wsSesion.Session.Health = false ; !wsSesion.Session.Health;  {
		log.Println("check url:",url)
		wsSesion.Session.SendCommand("curl -I -m 10 -s "+url)
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

func NewJumpserverClient(conf *JumpserverConfig, c websocket.Connection,wsSesion *session.WsSesion) (*ssh.Client, error) {
	var config ssh.ClientConfig
	var authMethods []ssh.AuthMethod
	authMethods = append(authMethods, ssh.Password(conf.Password))
	authMethods = append(authMethods, ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
		answers := make([]string, 0, len(questions))
		for i, q := range questions {
			fmt.Print(q)
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
				MFA := <- wsSesion.IN
				fmt.Println("MFA:", MFA)
				answers = append(answers, MFA)
			} else {
				b, err := terminal.ReadPassword(int(syscall.Stdin))
				if err != nil {
					return nil, err
				}
				fmt.Println("aaa")
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

	client, err := ssh.Dial("tcp", conf.Ip+":"+strconv.Itoa(conf.Port), &config)
	if err != nil {
		panic("Failed to dial: " + err.Error())
		return nil, err
	}

	return client, nil
}

func NewSession(client *ssh.Client,wsSesion *session.WsSesion) *session.JumpserverSession {
	sshSession, err := client.NewSession()
	CheckErr(err, "create new session")
	in := &session.Input{make(chan string)}
	sshSession.Stdin = in
	out := &session.Output{wsSesion.OUT,wsSesion.Session}//todo
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
		log.Println(errors.New("unable request pty  " + err.Error()))
	}
	jumpserverSession := &session.JumpserverSession{sshSession, in, out,true,"",wsSesion,0,""}
	out.JumpserverSession = jumpserverSession
	go func(s *ssh.Session) {
		err = s.Shell()
		CheckErr(err, "session shell")
		err = s.Wait()
		CheckErr(err, "session wait")
		log.Println("session over")
		wsSesion.OUT <- "close channel session"
	}(sshSession)
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
		log.Println("GetSftp.error", err)
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
		log.Println(os.Stderr, "File Error: %s\n", inputError)
	}

	fileInfo, err := inputFile.Stat()
	if err != nil {
		log.Println("fileinfo err:", err)
	}
	defer inputFile.Close()

	if fileInfo.IsDir() {
		//mkdir
		path := remotePath + Separator + fileInfo.Name()
		sftp.Mkdir(path)
		//log.Println(path)

		fileInfo, err := inputFile.Readdir(-1)
		if err == nil {
			for _, f := range fileInfo {

				uploadPath(file+Separator+f.Name(), sftp, path)

			}
		}

	} else {
		//copy file
		//log.Println(remotePath + Separator + fileInfo.Name())
		uploadFile(sftp, file, remotePath+Separator+fileInfo.Name())
	}

}

/**
上传文件
*/
func uploadFile(sftp *sftp.Client, localFile, remotePath string) {
	log.Println(localFile, ",", remotePath)
	// leave your mark
	inputFile, inputError := os.Open(localFile)
	//fileInfo , err := inputFile.Stat();
	defer inputFile.Close()

	f, err := sftp.Create(remotePath)

	if err != nil {
		log.Println("sftp.Create.err", err)
	}

	if inputError != nil {
		log.Println("File Error: %s\n", inputError)
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
		//fmt.Printf("%d,%s", n, string(buf))
		if n == 0 {
			break
		}
		//fmt.Println(string(buf))
		if _, err := f.Write(buf[0:n]); err != nil {
			log.Println(err)
		}

	}
	// check it's there
	fi, err := sftp.Lstat(remotePath)
	if err != nil {
		log.Println("sftp.Lstat.error", err)
	}
	log.Println(fi)

}

/**
删除文件
*/
func RemoveFile(remoateFile string, sftp *sftp.Client) {
	err := sftp.Remove(remoateFile)
	if err != nil {
		log.Println(err)
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
		log.Println(w.Path())
	}
}

func CheckErr(err error, msg string) {
	if err != nil {
		log.Println(msg+" err:", err)
	}
}
