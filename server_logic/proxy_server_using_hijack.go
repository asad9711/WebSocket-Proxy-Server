package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/gorilla/mux"
)

// utility to copy data from one connection to another
func forwardDataBetweenPorts(src, dest net.Conn) {
	var err error

	defer src.Close()
	defer dest.Close()
	_, err = io.Copy(src, dest)
	if err != nil {
		log.Println("ERR in copy - ", err.Error())
		return
	}
}

// 2 way comm b/w client and server
func connectToRemoteAndCopy(c net.Conn, req *http.Request, requestData []byte) {
	var remote net.Conn
	var err error

	// TODO: read this from config
	var proxyTargetURL string
	remote, err = net.Dial("tcp", proxyTargetURL)
	if err != nil {
		log.Println("Unable to create tcp connection to " + proxyTargetURL)
		return
	}

	_, err = remote.Write(requestData)
	if err != nil {
		log.Println("ERROR writing req to remote")
		return
	} else {
		log.Println("req written")
	}

	go forwardDataBetweenPorts(c, remote)
	go forwardDataBetweenPorts(remote, c)
}

func HijackedProxyEndpoint(res http.ResponseWriter, req *http.Request) {

	var err error
	// TODO: read from config
	var proxyTargetPath string
	receivedRequestBytes, err := httputil.DumpRequest(req, false)
	stringifiedRequest := string(receivedRequestBytes)

	// update request to contain the target websocket endpoint
	stringifiedRequest = strings.Replace(stringifiedRequest, "/websocket", proxyTargetPath, 1)

	requestData := []byte(stringifiedRequest)
	log.Println("updated req - ", stringifiedRequest)
	if err != nil {
		log.Println("err in dumping req")
		return
	}

	conn, _, err := res.(http.Hijacker).Hijack()
	if err != nil {
		log.Println("Hijack failed - reason", err.Error())
		return
	}
	log.Println("Hijack success")
	connectToRemoteAndCopy(conn, req, requestData)

}

func main() {

	f, err := os.OpenFile("proxy_server_logs.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	defer f.Close()

	wrt := io.MultiWriter(os.Stdout, f)
	log.SetOutput(wrt)
	log.Println("log file created")

	r := mux.NewRouter()
	r.HandleFunc("/websocket", HijackedProxyEndpoint)

	err = http.ListenAndServe("localhost:8080", r)
	if err != nil {
		log.Println("server start failed. %v", err)
		return
	}
	log.Println("server started ")
}
