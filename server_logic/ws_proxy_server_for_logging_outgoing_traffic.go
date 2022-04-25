package server_logic

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"

	"github.com/gorilla/mux"

	"github.com/gorilla/websocket"
)

const (
	incomingBufferSize = 1024 * 100
	outgoingBufferSize = 1024 * 100
)

var wsUpgrader = &websocket.Upgrader{
	ReadBufferSize:  incomingBufferSize,
	WriteBufferSize: outgoingBufferSize,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var wsDialer = &websocket.Dialer{
	ReadBufferSize:  outgoingBufferSize,
	WriteBufferSize: incomingBufferSize,
}

func HTTPProxyHandler(res http.ResponseWriter, req *http.Request) {
	log.Printf("Received proxy HTTP req:: %v", req.URL.Path)
	// startChromeInstance()

	origin, _ := url.Parse("http://localhost:9222/")

	director := func(req *http.Request) {
		req.Header.Add("X-Forwarded-Host", req.Host)
		req.Header.Add("X-Origin-Host", origin.Host)
		req.URL.Scheme = "http"
		req.URL.Host = origin.Host
	}

	proxy := &httputil.ReverseProxy{Director: director}

	proxy.ServeHTTP(res, req)
}

func ProxyHandler(res http.ResponseWriter, req *http.Request) {

	var upgrader = websocket.Upgrader{} // use default options

	log.Println("req method - ", req.Method)
	urlParams := req.URL.Query()
	// urlParamsString, _ := json.Marshal(urlParams["capabilities"][0])
	// log.Println("urlParamString: - ", string(urlParamsString))

	urlEncodedURL := urlParams.Encode()
	// log.Println("urlEncodedURL --> ", urlEncodedURL)

	// wsurl := "wss://stage-cdp.lambdatest.com/playwright?capabilities=" + urlEncodedURL
	wsurl := "ws://asad-cdp.dev.lambdatest.io:31333/playwright?" + urlEncodedURL
	log.Println("urlEncodedURL --> ", wsurl)
	// return

	clientAndProxyWSConnection, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		msg := fmt.Sprintf("could not upgrade websocket from %s, got: %v", req.RemoteAddr, err)
		log.Println(msg)
		http.Error(res, msg, http.StatusInternalServerError)
		return
	}

	defer clientAndProxyWSConnection.Close()

	var wsDebugURL string
	wsDebugURL = wsurl
	// wsDebugURL = `wss://stage-cdp.lambdatest.com/playwright?capabilities=${encodeURIComponent(JSON.stringify(capabilities))}`

	// wsDebugURL = "ws://127.0.0.1:9222/devtools/browser/e9b28929-ee14-4506-bb40-383926ba45fa"

	proxyToBrowserWSConnection, _, err := wsDialer.Dial(wsDebugURL, nil)
	if err != nil {
		msg := fmt.Sprintf("could not connect to browser WS debug url %s, got: %v", wsDebugURL, err)
		log.Println(msg)
		http.Error(res, msg, http.StatusInternalServerError)
		return
	}
	log.Println("connected to browser debug url ", wsDebugURL)

	var wg sync.WaitGroup
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer func() {
			wg.Add(1)
			go CloseCDPSocketsAndUpdateTest(wg, clientAndProxyWSConnection, proxyToBrowserWSConnection)
			wg.Done()
		}()
		for {
			log.Println("Inside 1st goroutine for loop")
			_, message, err := proxyToBrowserWSConnection.ReadMessage()
			if err != nil {
				log.Println("read from proxyToBrowserWSConnection err: ", err)
				log.Println("WS connection between proxy service and browser closed")
				break
			}
			log.Printf("received from proxyToBrowserWSConnection: %s", message)
			err = clientAndProxyWSConnection.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Println("write: clientAndProxyWSConnection - err: ", err)
				break
			}
			log.Println("message sent to clientAndProxyWSConnection")
		}
		log.Println("Returning from 1st goroutine")
		return
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer func() {
			wg.Add(1)
			go CloseCDPSocketsAndUpdateTest(wg, clientAndProxyWSConnection, proxyToBrowserWSConnection)
			wg.Done()
		}()
		for {
			log.Println("Inside 2nd goroutine for loop")
			_, message, err := clientAndProxyWSConnection.ReadMessage()
			if err != nil {
				log.Println("read from clientAndProxyWSConnection err : ", err)
				log.Println("WS connection between client and proxy service closed")
				break
			}
			log.Printf("received from clientAndProxyWSConnection : %s", message)
			err = proxyToBrowserWSConnection.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Println("write into proxyToBrowserWSConnection - err : ", err)
				break
			}
			log.Println("message sent to proxyToBrowserWSConnection")
		}
		log.Println("Returning from 2nd goroutine")
		return

	}(&wg)

	wg.Wait()
	log.Println("=========TEST ENDED==========")
}

func CloseCDPSocketsAndUpdateTest(wg *sync.WaitGroup, clientToProxyConn *websocket.Conn, proxyToBrowserConn *websocket.Conn) {
	defer wg.Done()

	log.Println("cdp-close-session: CloseCDPSocketsAndUpdateTest START")
	err := clientToProxyConn.Close()
	if err != nil {
		log.Println("cdp-create-session: Error in closing clientToProxyConn, error: %v", err)
	} else {
		log.Println("cdp-close-session: clientToProxyConn close success")
	}

	err = proxyToBrowserConn.Close()
	if err != nil {
		log.Println("cdp-create-session: Error in closing proxyToBrowserConn, error: %v", err)
	} else {
		log.Println("cdp-close-session: proxyToBrowserConn close success")
	}

	log.Println("===========DEFER ENDED============")
}

func SetupRouteAndStartServer(portNumberRequested string) {

	f, err := os.OpenFile("proxy_server_logs.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	defer f.Close()

	wrt := io.MultiWriter(os.Stdout, f)
	log.SetOutput(wrt)
	log.Println("log file created")

	log.Println("creating endpoint mapping")

	r := mux.NewRouter()
	// r.HandleFunc("/ws_endpoint/{session}", ProxyHandler)
	r.HandleFunc("/ws_endpoint", ProxyHandler)
	// r.HandleFunc("/devtools/page/{ID}/{session}", ProxyHandler)
	r.HandleFunc("/devtools/page", ProxyHandler)
	r.HandleFunc("/json/protocol", HTTPProxyHandler)
	r.HandleFunc("/json/list", HTTPProxyHandler)
	r.HandleFunc("/json/version", HTTPProxyHandler)
	r.HandleFunc("/json", HTTPProxyHandler)
	r.HandleFunc("/json/new", HTTPProxyHandler)
	r.HandleFunc("/json/activate/{targetId}", HTTPProxyHandler)
	r.HandleFunc("/json/close/{targetId}", HTTPProxyHandler)
	r.HandleFunc("/devtools/inspector.html", HTTPProxyHandler)

	log.Println("server starting at port ", portNumberRequested)
	err = http.ListenAndServe(fmt.Sprintf("localhost:%s", portNumberRequested), r)
	if err != nil {
		log.Println("server start failed. Err ->", err)
		return
	}
	log.Println("server started ")
}
