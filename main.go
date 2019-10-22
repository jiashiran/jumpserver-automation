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

	service()

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
