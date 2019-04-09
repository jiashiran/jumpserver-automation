package session

import (
	"github.com/kataras/iris/websocket"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"strings"
	"sync/atomic"
)

type WsSesion struct {
	ID          string
	Client      *ssh.Client
	Session     *JumpserverSession
	OUT         chan string //           = make(chan string, 100)
	IN          chan string //           = make(chan string)
	LoginServer *uint32

	C websocket.Connection
}

type JumpserverSession struct {
	*ssh.Session
	In           *Input
	Out          *Output
	Health       *uint32
	CheckURL     string
	WebSesion    *WsSesion
	CheckCount   *int32
	CheckCommand string
}

func (s *JumpserverSession) SendCommand(command string) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("SendCommand error:", err)
		}
	}()
	s.In.In <- command
	log.Println("send command:", command)
}

type Input struct {
	In chan string
}

func (in *Input) Read(p []byte) (n int, err error) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Read error:", err)
			close(in.In)
		}
	}()
	log.Println("wait read...")
	str, isOpen := <-in.In
	if !isOpen {
		return 0, io.EOF
	}
	if strings.Index(str, "\n") <= 0 {
		str = str + "\n"
	}
	//log.Println("receive command:", str)
	if str == io.EOF.Error() {
		return 0, io.EOF
	}
	if str == "" {
		return 0, nil
	}
	bytes := []byte(str)
	for i, b := range bytes {
		p[i] = b
	}
	return len(bytes), nil
}

type Output struct {
	Out               chan string
	JumpserverSession *JumpserverSession
}

func (out *Output) Write(p []byte) (n int, err error) {
	/*defer func() {
		if err := recover(); err != nil {
			log.Println("Write error:", err)
			close(out.Out)
		}
	}()*/
	/*if len(p) == 0 {
		log.Println("session close")
		return -1, io.EOF
	}*/
	output := string(p)
	outputs := strings.Split(output, "\n")
	for _, output = range outputs {
		if strings.Contains(output, "Opt>") {
			atomic.StoreUint32(out.JumpserverSession.WebSesion.LoginServer, 0)
		}
		if atomic.LoadUint32(out.JumpserverSession.WebSesion.LoginServer) == 0 && (strings.Contains(output, "$") || strings.Contains(output, "#")) {
			atomic.StoreUint32(out.JumpserverSession.WebSesion.LoginServer, 1)
		}
		if out.JumpserverSession.CheckURL != "" && strings.Contains(output, out.JumpserverSession.CheckURL) && !strings.Contains(output, out.JumpserverSession.CheckCommand) {
			atomic.AddInt32(out.JumpserverSession.CheckCount, 1)
			log.Println("健康检查", atomic.LoadInt32(out.JumpserverSession.CheckCount))
			if atomic.LoadInt32(out.JumpserverSession.CheckCount) >= 2 {
				atomic.StoreUint32(out.JumpserverSession.Health, 1)
				out.JumpserverSession.WebSesion.OUT <- "健康监测成功"
			}
		}
		/*if out.JumpserverSession.CheckURL != "" && out.JumpserverSession.CheckCommand != "" && !strings.Contains(output, out.JumpserverSession.CheckCommand){

		}*/
		out.JumpserverSession.WebSesion.OUT <- output
		/*if strings.Contains(output,"nameserver") || strings.Contains(output,"B_") || strings.Contains(output,"VLINK_"){
			out.JumpserverSession.WebSesion.OUT <- output
		}*/
		//log.Println("output:", output)
	}

	return len(p), nil
}
