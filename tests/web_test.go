package tests

import (
	"fmt"
	. "github.com/chapgx/assert"
	. "github.com/racg0092/dumbo"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestStartSession(t *testing.T) {
	db := os.Getenv("db")
	if db == "" {
		db = "mongodb://localhost:7171"
	}

	var cookies []*http.Cookie

	store, err := NewMongoStore("ttt", "sessions", db)
	AssertT(t, err == nil, err)

	Start(Options{
		HttpOnly: true,
		Secure:   true,
		MaxAge:   time.Minute * 30,
	}, &store)

	t.Run("create session with increment", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		CreateSessionInc(w, req)

		resp := w.Result()
		body := w.Body.String()

		AssertT(t, resp.StatusCode == http.StatusOK,
			fmt.Sprintf(
				"expected %d, got %d. %s",
				http.StatusOK, resp.StatusCode, resp.Status,
			),
		)
		AssertT(t, body == "1", fmt.Sprintf("expected %q got %q", "1", body))

		cookies = w.Result().Cookies()

		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		for _, c := range cookies {
			req2.AddCookie(c)
		}
		w2 := httptest.NewRecorder()

		CreateSessionInc(w2, req2)

		resp = w2.Result()
		body = w2.Body.String()

		AssertT(t, resp.StatusCode == http.StatusOK,
			fmt.Sprintf(
				"expected %d, got %d. %s",
				http.StatusOK, resp.StatusCode, resp.Status,
			),
		)

		AssertT(t, body == "2", fmt.Sprintf("expected %q got %q", "2", body))

	})

	t.Run("complex type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		for _, c := range cookies {
			req.AddCookie(c)
		}

		ComplexType(w, req)

		resp := w.Result()
		body := w.Body.String()

		AssertT(t, resp.StatusCode == http.StatusOK, fmt.Sprintf("expected %d got %d, %s", http.StatusOK, resp.StatusCode, resp.Status))
		AssertT(t, body == "Richard", fmt.Sprintf("expected %q got %q", "Richard", body))

		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		w2 := httptest.NewRecorder()
		for _, c := range cookies {
			req2.AddCookie(c)
		}
		ComplexType(w2, req2)

		resp = w2.Result()
		body = w2.Body.String()
		AssertT(t, resp.StatusCode == http.StatusOK, fmt.Sprintf("expected %d got %d, %s", http.StatusOK, resp.StatusCode, resp.Status))
		AssertT(t, body == "Richard Chapman", fmt.Sprintf("expected %q got %q", "Richard Chapman", body))

	})

	//TODO: test values when content is a pointer
	//

	t.Run("delete session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		for _, c := range cookies {
			req.AddCookie(c)
		}

		DeleteSession(w, req)

		resp := w.Result()

		AssertT(t, resp.StatusCode == http.StatusOK, fmt.Sprintf("expected %d got %d, %s", http.StatusOK, resp.StatusCode, resp.Status))
	})
}

func TestUpdateSessionExpiration(t *testing.T) {
	db := os.Getenv("mdb")
	if db == "" {
		db = "mongodb://localhost:7171"
	}

	var cookies []*http.Cookie

	store, e := NewMongoStore("ttt", "sessions", db)
	AssertT(t, e == nil, e)

	Start(Options{
		HttpOnly: true,
		Secure:   true,
		MaxAge:   time.Minute * 30,
	}, &store)

	t.Run("start session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		CoolSession(w, req)

		resp := w.Result()
		cookies = w.Result().Cookies()

		AssertT(t, resp.StatusCode == http.StatusOK, "status code is not 200 ok")

		chocolate := cookies[0]

		fmt.Println(chocolate.Expires.In(time.Local))

	})

	t.Run("update session expiration", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		for _, c := range cookies {
			req.AddCookie(c)
		}

		UpdateCoolSession(w, req)

		resp := w.Result()
		cookies = w.Result().Cookies()

		AssertT(t, resp.StatusCode == http.StatusOK, "status code is not 200 ok")

		chocolate := cookies[0]
		fmt.Println(chocolate.Expires.In(time.Local))

		time.Sleep(time.Second * 5)
	})

}

func CoolSession(w http.ResponseWriter, r *http.Request) {
	sess := Get(r, w, "cool")
	e := sess.SetValue(Value{"ping", "pong"}, w)
	if e != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(e.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func UpdateCoolSession(w http.ResponseWriter, r *http.Request) {
	e := UpdateExpiration(w, r, "cool", time.Hour*24*90)
	if e != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(e.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func CreateSessionInc(w http.ResponseWriter, r *http.Request) {
	session := Get(r, w, "session")
	var counter int
	e := session.GetDecode("counter", &counter)
	if e == ErrValIsNil {
		counter = 1
		session.SaveMustCompile("counter", counter, w)
	} else {
		counter = counter + 1
		session.SaveMustCompile("counter", counter, w)
	}

	time.Sleep(time.Second * 1) //NOTE: give enough time for go routines
	w.Write(fmt.Appendf([]byte(""), "%d", counter))
}

type I struct {
	Name string
	Age  int
}

func ComplexType(w http.ResponseWriter, r *http.Request) {
	sess := Get(r, w, "session")
	var i I
	e := sess.GetDecode("item", &i)
	if e == ErrValIsNil {
		i = I{
			Name: "Richard",
			Age:  33,
		}
		sess.SaveMustCompile("item", i, w)
	} else {
		i.Name = "Richard Chapman"
		sess.SaveMustCompile("item", i, w)
	}
	w.Write([]byte(i.Name))
	time.Sleep(time.Second * 1)
}

func DeleteSession(w http.ResponseWriter, r *http.Request) {
	Delete(r, w, "session")
	time.Sleep(time.Second * 1) //NOTE: give time for concurent functions to finish
}
