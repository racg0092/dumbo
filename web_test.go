package dumbo

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"
)

func TestSimpleServer(t *testing.T) {
	mtx := http.NewServeMux()
	store, err := NewMongoStore("ttt", "sessions", os.Getenv("db"))
	if err != nil {
		t.Error(err)
	}

	Start(Options{
		HttpOnly: true,
		Secure:   true,
		MaxAge:   60 * 5,
	}, &store)

	mtx.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		session := Get(r, w, "session")
		var counter int
		cv, exists := session.Values["counter"]
		if !exists {
			session.Values["counter"] = 1
			counter = 1
		} else {
			v, ok := cv.(int)
			if !ok {
				t.Error("wrong type")
			}
			counter = int(v) + 1
			session.Values["counter"] = counter
		}
		if err := session.Save(w); err != nil {
			panic(err)
		}
		w.Write([]byte("<h1>Hello World " + strconv.Itoa(counter) + "</h1>"))
	})

	mtx.HandleFunc("/delete-session", func(w http.ResponseWriter, r *http.Request) {
		Delete(r, w, "session")
		w.Write([]byte("session deleted"))
	})

	srv := &http.Server{Addr: "127.0.0.1:8080", Handler: mtx}
	defer srv.Close()

	fmt.Println("server stared in port 8080")
	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}

}
