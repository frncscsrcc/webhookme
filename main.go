package main

import (
	"bytes"
	"flag"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// GLOBALS

var sessionTTL int
var port string
var basePath string
var requestCounter int = 0
var globalMaxRequestPerMinute int = 60
var maxBodySizeBytes int = 500000
var linkLenght int = 8

// DATA TYPES

type Response struct {
	Time   string
	Method string
	Path   string
	Header map[string][]string
	Query  map[string][]string
	Body   string
}

type Session struct {
	Id        string
	Expires   string
	Responses []Response
}

type DataForTemplates struct {
	Initialized        bool
	BasePath           string
	Session            Session
	Error              string
	RequestsLastMinute int
}

// IN MEMORY STORAGE

var sessions map[string]*Session

func init() {
	sessions = make(map[string]*Session)
	rand.Seed(time.Now().UTC().UnixNano())
}

// HANDLERS

func Home(w http.ResponseWriter, r *http.Request) {
	if len(strings.TrimPrefix(r.URL.Path, "/")) > 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	session := newSession()

	data := DataForTemplates{
		Initialized:        false,
		BasePath:           basePath,
		Session:            session,
		RequestsLastMinute: requestCounter,
	}

	t := template.Must(template.ParseFiles("./templates/index.html"))
	err := t.Execute(w, data)
	if err != nil {
		panic(err)
	}
}

func Collect(w http.ResponseWriter, r *http.Request) {

	if tooManyRequests() {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/send/")
	sessionId := strings.Split(path, "/")[0]

	if _, ok := sessions[sessionId]; !ok {
		w.WriteHeader(http.StatusNotFound)
	}

	var bodyBuffer bytes.Buffer
	if r.Body != nil {
		defer r.Body.Close()
		io.Copy(&bodyBuffer, r.Body)
	}
	if len(bodyBuffer.Bytes()) > maxBodySizeBytes {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	query := make(map[string][]string)
	for key, values := range r.URL.Query() {
		query[key] = values
	}

	body := bodyBuffer.String()

	response := Response{
		Time:   time.Now().Format(time.RFC3339),
		Header: r.Header,
		Method: r.Method,
		Path:   path,
		Query:  query,
		Body:   body,
	}

	if _, ok := sessions[sessionId]; ok {
		sessions[sessionId].Responses = append(sessions[sessionId].Responses, response)
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("OK"))
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

func Show(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/listen/")
	sessionId := strings.Split(path, "/")[0]

	t := template.Must(template.ParseFiles("./templates/index.html"))

	session, sessionExists := sessions[sessionId]

	var data DataForTemplates

	if !sessionExists {
		data.Error = "Session expired"
	} else {
		data.Initialized = true
		data.Session = *session
		data.BasePath = basePath
	}

	t.Execute(w, data)
}

func Delete(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/delete/")
	sessionId := strings.Split(path, "/")[0]

	_, sessionExists := sessions[sessionId]

	if sessionExists {
		delete(sessions, sessionId)
	}

	w.Write([]byte("OK"))
}

// MAIN

func main() {
	flag.StringVar(&basePath, "base", "http://localhost", "Base path to build the links")
	flag.StringVar(&port, "port", "8080", "Port to expose")
	flag.IntVar(&sessionTTL, "ttl", 300, "Sessions TTL (sec)")
	flag.IntVar(&globalMaxRequestPerMinute, "global-max-request-per-minute", 60, "how many request per minute the server can accept")
	flag.IntVar(&maxBodySizeBytes, "max-body-size", 500000, "Max size of the body in bytes")
	flag.IntVar(&linkLenght, "link-lenght", 8, "Size of the link's random token")

	flag.Parse()

	http.HandleFunc("/", Home)
	http.HandleFunc("/send/", Collect)
	http.HandleFunc("/listen/", Show)
	http.HandleFunc("/delete/", Delete)

	log.Printf("Listening on %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// HELPERS

func newSession() Session {
	ttl := time.Duration(sessionTTL) * time.Second
	s := Session{
		Id:        randomString(linkLenght),
		Expires:   time.Now().Add(ttl).Format(time.RFC3339),
		Responses: make([]Response, 0),
	}

	sessions[s.Id] = &s
	log.Printf("Session %s created", s.Id)

	go func() {
		time.Sleep(ttl)
		log.Printf("Session %s deleted", s.Id)
		delete(sessions, s.Id)
	}()

	return s
}

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func tooManyRequests() bool {
	if requestCounter > globalMaxRequestPerMinute {
		log.Print("Too many requests")
		return true
	}
	requestCounter += 1
	go func() {
		time.Sleep(time.Minute)
		requestCounter -= 1
	}()
	return false
}
