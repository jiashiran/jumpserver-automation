package util

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func GetDirList(path string) []os.FileInfo {
	//以只读的方式打开目录
	f, err := os.OpenFile(path, os.O_RDONLY, os.ModeDir)
	if err != nil {
		fmt.Println(err.Error())
	}
	//延迟关闭目录
	defer f.Close()
	fileInfo, _ := f.Readdir(-1)
	//操作系统指定的路径分隔符
	separator := string(os.PathSeparator)
	_ = separator

	return fileInfo
}

func ReadDir(path string) {
	//以只读的方式打开目录
	f, err := os.OpenFile(path, os.O_RDONLY, os.ModeDir)
	if err != nil {
		fmt.Println(err.Error())
	}
	//延迟关闭目录
	defer f.Close()
	fileInfo, _ := f.Readdir(-1)
	//操作系统指定的路径分隔符
	separator := string(os.PathSeparator)
	_ = separator
	for _, info := range fileInfo {
		//判断是否是目录
		if info.IsDir() {
			//fmt.Println(path + separator + info.Name())
			//readDir(path + separator + info.Name())
		} else {
			if strings.Contains(info.Name(), ".csv") {
				//fmt.Println("文件：" + info.Name())
				ReadLine(info.Name(), func(s string) {

				})
			}
		}
	}
}

func ReadLine(fileName string, handler func(string)) error {
	f, err := os.Open(fileName)
	if err != nil {
		log.Println("open err:", err)
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
