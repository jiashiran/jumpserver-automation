package util

import (
	"bufio"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
	"strings"
)

const Separator = "/"

/**
上传目录
*/
func UploadPath(client *ssh.Client, localPath, remotePath string) {
	sftp := GetSftpClient(client)
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
func ListPath(sftp *sftp.Client, remotePath string) []string {
	//defer sftp.Close()
	// walk a directory
	files := make([]string, 0)
	w := sftp.Walk(remotePath)
	for w.Step() {
		if w.Err() != nil {
			continue
		}
		files = append(files, w.Path())
		log.Println(w.Path())
	}
	return files
}

func DownloadFile(sftp *sftp.Client, remoteFile string, localPath string) {
	f, err := sftp.Open(remoteFile)
	if err != nil {
		log.Println(err)
	}
	bs := make([]byte, 20480)
	i, err := f.Read(bs)
	if i > 0 || err != io.EOF {
		ss := strings.Split(f.Name(), `/`)

		localFile, err := os.Create(localPath + "/" + ss[len(ss)-1])
		if err != nil {
			log.Println("createFile err:", err)
		}
		for i > 0 || err != io.EOF {
			localFile.Write(bs)
			i, err = f.Read(bs)
		}
	}

}
