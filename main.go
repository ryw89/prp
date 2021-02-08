package main

import (
	"errors"
	"github.com/BurntSushi/toml"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port        int
	Servers     []string
	UseIncludes bool
	Timeout     int
}

// Global config
var confPath = "./config.toml"
var conf Config

func parseConfig(configPath string) (Config, error) {
	var conf Config
	_, err := toml.DecodeFile(configPath, &conf)
	return conf, err
}

func fetchHosts() []string {
	return conf.Servers
}

// Given a list of hosts and path, return the first host that can
// serve the request.
func checkHeadOk(host string, path string) (string, error) {
	res, _ := http.Head(host + path)
	if res.StatusCode != 200 && res.StatusCode != 302 {
		err := errors.New("Error: HEAD request returned non-OK status code.")
		return host + path, err
	}
	return host + path, nil
}

// Fetch a file over HTTP
func fetchFile(path string) (*http.Response, error) {
	// Based on: https://play.golang.org/p/v9IAu2Xu3_
	timeout := time.Duration(conf.Timeout) * time.Millisecond
	transport := &http.Transport{
		ResponseHeaderTimeout: timeout,
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, timeout)
		},
		DisableKeepAlives: true,
	}
	client := &http.Client{
		Transport: transport,
	}
	resp, err := client.Get(path)
	if err != nil {
		log.Println(err)
	}
	return resp, err
}

func serveReverseProxy(w http.ResponseWriter, r *http.Request) {
	var path string

	hosts := fetchHosts()

	// Check if this file exists on various hosts using HEAD request
	for i := 0; i < len(hosts); i++ {
		_path, err := checkHeadOk(hosts[i], r.URL.Path)
		if err == nil {
			log.Printf("Good HEAD response found from %s.\n", _path)
			path = _path
			break
		} else {
			log.Printf("Bad HEAD response found from %s.\n", _path)
		}

	}

	// Resource not found, so 404.
	if path == "" {
		log.Printf("Received: %s %s%s from %s; returning 404.\n",
			r.Method, r.Host, r.URL.Path, r.RemoteAddr)
		w.WriteHeader(404)
		return
	}

	resp, err := fetchFile(path)
	defer resp.Body.Close()
	if err != nil {
		log.Printf("Received: %s %s%s from %s; returning 500.\n",
			r.Method, r.Host, r.URL.Path, r.RemoteAddr)
		w.WriteHeader(500)
		return
	}

	// Copy the relevant headers.
	w.Header().Set("Content-Disposition", "attachment; filename="+r.URL.Path)
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", resp.Header.Get("Content-Length"))

	log.Printf("Received: %s %s%s from %s; proxied to %s.\n",
		r.Method, r.Host, r.URL.Path, r.RemoteAddr, path)

	// stream the body to the client without fully loading it into memory
	io.Copy(w, resp.Body)

}

func main() {
	// Fetch config
	if len(os.Args) > 1 {
		confPath = os.Args[1]
	}

	var err error
	conf, err = parseConfig(confPath)
	if err != nil {
		panic(err)
	}
	// Run server
	http.HandleFunc("/", serveReverseProxy)

	port := ":" + strconv.Itoa(conf.Port)
	log.Printf("Starting server on %s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		panic(err)
	}
}
