package main

import (
	_ "embed"
	"fmt"
	"strings"
	"flag"
	"log"
	"net/http"
	"database/sql"
	"encoding/json"
	"time"
	"os"

	_ "modernc.org/sqlite"
)

var addr = flag.String("addr", ":8080", "http service address")

//go:embed cards.txt
var cardText []byte
var cards = []*Card{}

var parser = NewCardParser()

func init() {
	for _, txt := range strings.Split(string(cardText), "\n\n") {
		card, err := parser.Parse(txt)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println(card)
		cards = append(cards, card)
	}
}

type LoginUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Cors struct {
	handler http.Handler
}
func (c *Cors) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	c.handler.ServeHTTP(w, r)
}

type Logger struct {
    handler http.Handler
	logger *log.Logger
}
func (l *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    l.handler.ServeHTTP(w, r)
    l.logger.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
}

func main() {
	flag.Parse()

	db, err := sql.Open("sqlite", "./db.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repo := NewRepository(db)
	broker := NewMemoryBroker()
	wsServer := NewWebsocketServer(broker, repo)
	go wsServer.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		ServeWs(wsServer, w, r)
	}))
	
	mux.HandleFunc("/login", func (w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var user LoginUser
			err := json.NewDecoder(r.Body).Decode(&user)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			dbUser := repo.FindUserByName(user.Username)
			if dbUser == nil {
				log.Println("user not found")
				w.Write([]byte("{\"status\": \"error\"}"))
				return
			}
			if !dbUser.Password.Valid {
				log.Println("password not set")
				w.Write([]byte("{\"status\": \"error\"}"))
				return
			}
			if ok, err := ValidatePassword(user.Password, dbUser.Password.String); !ok || err != nil {
				log.Println(err)
				w.Write([]byte("{\"status\": \"error\"}"))
				return
			}
			token, err := CreateJWTToken(dbUser)
			if err != nil {
				log.Println(err)
				w.Write([]byte("{\"status\": \"error\"}"))
				return
			}
			data, _ := json.Marshal(token)
			w.Write(data)
		}
	})

	mux.HandleFunc("/register", func (w http.ResponseWriter, r *http.Request) {
		var user LoginUser
		var dbUser *User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		dbUser = repo.FindUserByName(user.Username)
		w.Header().Set("Content-Type", "application/json")
		if dbUser == nil {
			if dbUser, err = repo.AddUser(user.Username); err != nil {
				log.Println(err)
				w.Write([]byte("{\"status\": \"error\"}"))
				return
			}
		}
		hash, err := GeneratePassword(user.Password)
		if err != nil {
			log.Println(err)
			w.Write([]byte("{\"status\": \"error\"}"))
			return
		}
		if err := repo.SetPassword(dbUser, hash); err != nil {
			log.Println(err)
			w.Write([]byte("{\"status\": \"error\"}"))
			return
		}
		w.Write([]byte("{\"status\": \"ok\"}"))
	})

	fs := http.FileServer(http.Dir("./public"))
	mux.Handle("/", fs)

	logger := log.New(os.Stderr, "[http]: ", log.LstdFlags)
	wrapped := &Logger{&Cors{mux}, logger}
	log.Printf("http server started on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, wrapped))
}