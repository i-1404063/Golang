package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Coaster struct {
	Name         string `json:"name"`
	Id           string `json:"id"`
	Manufacturer string `json:"manufacturer"`
	InPark       string `json:"inpark"`
}

// for data storage
type coasterHandlers struct {
	sync.Mutex
	store map[string]Coaster
}

func (h *coasterHandlers) coasters(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Method)
	switch r.Method {
	case "GET":
		h.get(w, r)
		return

	case "POST":
		h.post(w, r)
		return

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}
}

func (h *coasterHandlers) get(w http.ResponseWriter, r *http.Request) {
	coasters := make([]Coaster, len(h.store))

	h.Lock()
	i := 0
	for _, coaster := range h.store {
		coasters[i] = coaster
		i++
	}
	h.Unlock()

	jsonBytes, err := json.Marshal(coasters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (h *coasterHandlers) post(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	var coaster Coaster
	err = json.Unmarshal(bodyBytes, &coaster)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	ct := r.Header.Get("content-type")
	if ct != "application/json" {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		w.Write([]byte(fmt.Sprintf("need content type 'application/json' but got '%s'", ct)))
		return
	}

	coaster.Id = strconv.Itoa(rand.Intn(10000))

	h.Lock()
	h.store[coaster.Id] = coaster
	defer h.Unlock()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("successfully created the coaster"))
}

// implementing redirect
func (h *coasterHandlers) randomHandler(w http.ResponseWriter, r *http.Request) {
	ids := make([]string, len(h.store))

	i := 0
	h.Lock()
	for id := range h.store {
		ids[i] = id
		i++
	}
	defer h.Unlock()

	var target string

	if len(ids) == 0 {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("no coaster in the database"))
		return
	} else if len(ids) == 1 {
		target = ids[0]
	} else {
		rand.Seed(time.Now().UnixNano())
		target = ids[rand.Intn(len(ids))]
	}

	w.Header().Add("location", fmt.Sprintf("/coasters/%s", target))
	w.WriteHeader(http.StatusFound)
}

func (h *coasterHandlers) getCoaster(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(r.URL.String(), "/")
	if len(path) != 3 {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("data with the specific id not exists"))
		return
	}

	if path[2] == "random" {
		h.randomHandler(w, r)
		return
	}

	h.Lock()
	coaster, ok := h.store[path[2]]
	h.Unlock()

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("data with the specific id not exists"))
		return
	}

	jsonBytes, err := json.Marshal(coaster)
	if err != nil {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func newCoasterHandlers() *coasterHandlers {
	return &coasterHandlers{
		store: map[string]Coaster{},
	}
}

type adminPortal struct {
	password string
}

func newAdminPortal() *adminPortal {
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		panic("required env var ADMIN_PASSWORD is not set")
	}

	return &adminPortal{password: password}
}

func (a adminPortal) handler(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || user != "admin" || pass != a.password {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - unAuthorized"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Congrats! you are successfully logged in."))
}

func main() {
	os.Setenv("ADMIN_PASSWORD", "secret")
	admin := newAdminPortal()
	coasterHanlder := newCoasterHandlers()
	http.HandleFunc("/coasters", coasterHanlder.coasters)
	http.HandleFunc("/coasters/", coasterHanlder.getCoaster)
	http.HandleFunc("/admin", admin.handler)
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		panic(err)
	}
}
