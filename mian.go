package main

import (
	"bufio"
	"github.com/kataras/golog"
	"io"
	"jumpserver-automation/ws"
	"os"
	"strings"
)

func main() {

	//util.OperatLb("LB aliyun slb in i-uf68a1l5ulumn2tizgw5 lb-uf6lfi06q95h5bcyxb87e 8089")
	//util.OperatLb("LB aws alb in i-0e41029513647e310 arn:aws-cn:elasticloadbalancing:cn-north-1:147022339119:targetgroup/sip-sbc-302/1dfa6510b0676035 8089")
	//util.OperatLb("LB aws elb in i-00722409683af6fab lbaaa 8089")

	service()

	/*ReadLine("/Users/jiashiran/Documents/all_host.txt", func(s string) {
		fmt.Println("LOGIN "+s)
		fmt.Println("SHELL  cat /etc/resolv.conf | grep nameserver")
		fmt.Println("SLEEP 3s")
		fmt.Println("LOGOUT")
	})*/
}

func ReadLine(fileName string, handler func(string)) error {
	f, err := os.Open(fileName)
	if err != nil {
		golog.Info(err)
		return err
	}
	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		line = strings.TrimSpace(line)
		handler(line)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
	return nil
}

func service() {
	ws.Service()
}

/*func WaitStop()  {
	// Go signal notification works by sending `os.Signal`
	// values on a channel. We'll create a channel to
	// receive these notifications (we'll also make one to
	// notify us when the program can exit).
	sigs := make(chan os.Signal)
	done := make(chan bool, 1)
	// `signal.Notify` registers the given channel to
	// receive notifications of the specified signals.
	signal.Notify(sigs, syscall.SIGILL, syscall.SIGTRAP, syscall.SIGABRT, syscall.SIGBUS, syscall.SIGFPE, syscall.SIGKILL,syscall.SIGSEGV,syscall.SIGPIPE,syscall.SIGALRM,syscall.SIGTERM)
	// This goroutine executes a blocking receive for
	// signals. When it gets one it'll print it out
	// and then notify the program that it can finish.
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		store.Close()
		done <- true
	}()
	// The program will wait here until it gets the
	// expected signal (as indicated by the goroutine
	// above sending a value on `done`) and then exit.
	fmt.Println("awaiting signal")
	<-done
	fmt.Println("exiting")
}*/
