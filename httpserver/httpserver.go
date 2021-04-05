package httpserver

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"github.com/gazercloud/sws/logger"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Host struct {
	Name string
}

type HttpServer struct {
	srv      *http.Server
	srvTLS   *http.Server
	r        *mux.Router
	rTLS     *mux.Router
	rootPath string
}

func CurrentExePath() string {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	return dir
}

func NewHttpServer() *HttpServer {
	var c HttpServer
	c.rootPath = CurrentExePath() + "/www"
	return &c
}

func (c *HttpServer) Start() {
	logger.Println("HttpServer start")
	go c.thListen()
	go c.thListenTLS()
}

func (c *HttpServer) thListen() {
	c.srv = &http.Server{
		Addr: ":80",
	}

	c.r = mux.NewRouter()
	c.r.NotFoundHandler = http.HandlerFunc(c.processFile)
	c.srv.Handler = c.r

	logger.Println("HttpServer thListen begin")
	err := c.srv.ListenAndServe()
	if err != nil {
		logger.Println("HttpServer thListen error: ", err)
	}
	logger.Println("HttpServer thListen end")
}

func (c *HttpServer) thListenTLS() {
	tlsConfig := &tls.Config{}
	tlsConfig.Certificates = make([]tls.Certificate, 0)
	dirs, _ := logger.GetDir(c.rootPath)
	for _, d := range dirs {
		if d.Dir {
			cert, err := tls.LoadX509KeyPair(d.Path+"/ssl/bundle.crt", d.Path+"/ssl/private.key")
			if err == nil {
				tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
			}
		}
	}

	c.srvTLS = &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
	}

	c.rTLS = mux.NewRouter()
	c.rTLS.NotFoundHandler = http.HandlerFunc(c.processFile)
	c.srvTLS.Handler = c.r

	logger.Println("HttpServerTLS thListen begin")
	listener, err := tls.Listen("tcp", ":443", tlsConfig)
	if err != nil {
		logger.Println("TLS Listener error:", err)
		return
	}

	err = c.srvTLS.Serve(listener)
	if err != nil {
		logger.Println("HttpServerTLS thListen error: ", err)
	}
	logger.Println("HttpServerTLS thListen end")
}

func (c *HttpServer) Stop() error {
	var err error

	{
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if err = c.srv.Shutdown(ctx); err != nil {
			logger.Println(err)
		}
	}

	{
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if err = c.srvTLS.Shutdown(ctx); err != nil {
			logger.Println(err)
		}
	}
	return err
}

func SplitRequest(path string) []string {
	return strings.FieldsFunc(path, func(r rune) bool {
		return r == '/'
	})
}

func (c *HttpServer) contentTypeByExt(ext string) string {
	var builtinTypesLower = map[string]string{
		".css":  "text/css; charset=utf-8",
		".gif":  "image/gif",
		".htm":  "text/html; charset=utf-8",
		".html": "text/html; charset=utf-8",
		".jpeg": "image/jpeg",
		".jpg":  "image/jpeg",
		".js":   "text/javascript; charset=utf-8",
		".mjs":  "text/javascript; charset=utf-8",
		".pdf":  "application/pdf",
		".png":  "image/png",
		".svg":  "image/svg+xml",
		".wasm": "application/wasm",
		".webp": "image/webp",
		".xml":  "text/xml; charset=utf-8",
	}

	logger.Println("Ext: ", ext)

	if ct, ok := builtinTypesLower[ext]; ok {
		return ct
	}
	return "text/plain"
}

func (c *HttpServer) processFile(w http.ResponseWriter, r *http.Request) {
	c.file(w, r, r.URL.Path)
}

func (c *HttpServer) file(w http.ResponseWriter, r *http.Request, urlPath string) {
	var err error
	var fileContent []byte
	var writtenBytes int

	realIP := getRealAddr(r)

	logger.Println("Real IP: ", realIP)
	logger.Println("HttpServer processFile: ", r.URL.String())

	if urlPath == "/" || urlPath == "" {
		urlPath = "/index.html"
	}

	url, err := c.fullpath(urlPath, r.Host)

	logger.Println("FullPath: " + url)

	if err != nil {
		w.WriteHeader(404)
		return
	}

	fileContent, err = ioutil.ReadFile(url)

	ext := filepath.Ext(url)
	if ext == ".html" {
		fileContent = c.processTemplate(fileContent, r.Host)
	}

	if err == nil {
		w.Header().Set("Content-Type", c.contentTypeByExt(filepath.Ext(url)))
		writtenBytes, err = w.Write(fileContent)
		if err != nil {
			logger.Println("HttpServer sendError w.Write error:", err)
		}
		if writtenBytes != len(fileContent) {
			logger.Println("HttpServer sendError w.Write data size mismatch. (", writtenBytes, " / ", len(fileContent))
		}
	} else {
		logger.Println("HttpServer processFile error: ", err)
		w.WriteHeader(404)
	}
}

func getRealAddr(r *http.Request) string {

	remoteIP := ""
	// the default is the originating ip. but we try to find better options because this is almost
	// never the right IP
	if parts := strings.Split(r.RemoteAddr, ":"); len(parts) == 2 {
		remoteIP = parts[0]
	}
	// If we have a forwarded-for header, take the address from there
	if xff := strings.Trim(r.Header.Get("X-Forwarded-For"), ","); len(xff) > 0 {
		addrs := strings.Split(xff, ",")
		lastFwd := addrs[len(addrs)-1]
		if ip := net.ParseIP(lastFwd); ip != nil {
			remoteIP = ip.String()
		}
		// parse X-Real-Ip header
	} else if xri := r.Header.Get("X-Real-Ip"); len(xri) > 0 {
		if ip := net.ParseIP(xri); ip != nil {
			remoteIP = ip.String()
		}
	}

	return remoteIP

}

func (c *HttpServer) sendError(w http.ResponseWriter, errorToSend error) {
	var err error
	var writtenBytes int
	var b []byte
	w.WriteHeader(500)
	b, err = json.Marshal(errorToSend.Error())
	if err != nil {
		logger.Println("HttpServer sendError json.Marshal error:", err)
	}
	writtenBytes, err = w.Write(b)
	if err != nil {
		logger.Println("HttpServer sendError w.Write error:", err)
	}
	if writtenBytes != len(b) {
		logger.Println("HttpServer sendError w.Write data size mismatch. (", writtenBytes, " / ", len(b))
	}
}

func (c *HttpServer) fullpath(url string, host string) (string, error) {
	result := ""

	result = c.rootPath + "/" + host + "/www/" + url

	fi, err := os.Stat(result)
	if err == nil {
		if fi.IsDir() {
			result += "/index.html"
		}
	}

	return result, err
}

func (c *HttpServer) redirect(w http.ResponseWriter, r *http.Request, url string) {
	w.Header().Set("Cache-Control", "no-cache, private, max-age=0")
	w.Header().Set("Expires", time.Unix(0, 0).Format(http.TimeFormat))
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Accel-Expires", "0")
	http.Redirect(w, r, url, 307)
}

func (c *HttpServer) processTemplate(tmp []byte, host string) []byte {
	tmpString := string(tmp)
	re := regexp.MustCompile(`\{#.*?#\}`)
	reResults := re.FindAllString(tmpString, 100)
	for _, reString := range reResults {
		filePath := strings.ReplaceAll(reString, "{#", "")
		filePath = strings.ReplaceAll(filePath, "#}", "")
		url, err := c.fullpath(filePath, host)
		if err != nil {
			logger.Println("processTemplate - c.fullpath(filePath) - ", err)
			continue
		}
		fileContent, err := ioutil.ReadFile(url)
		if err != nil {
			fileContent = []byte("-")
		} else {
			fileContent = c.processTemplate(fileContent, host)
		}
		tmpString = strings.ReplaceAll(tmpString, reString, string(fileContent))
	}
	return []byte(tmpString)
}
