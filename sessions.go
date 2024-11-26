package dumbo

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"sync"
	"time"
)

var manager *SessionManager
var options *Options
var store Store

type Options struct {
	MaxAge   int
	Secure   bool
	HttpOnly bool
}

type Session struct {
	ID      string                 `bson:"_id"`
	Values  map[string]interface{} `bson:"values"`
	IsNew   bool                   `bson:"isnew"`
	Expires time.Time              `bson:"expires"`
}

func (ses Session) Save() error {
	if store == nil {
		return errors.New("store is </nil>")
	}
	if store != nil {
		go store.Save(&ses)
	}
	return nil
}

type SessionManager struct {
	sessions map[string]*Session
	lock     sync.Mutex
}

func Start(opt Options, s Store) {
	options = &opt
	getManager()
	store = s
}

// Get Session manager
func getManager() *SessionManager {
	if manager == nil {
		go CleanUpExpiredSessions()
		manager = &SessionManager{
			sessions: make(map[string]*Session),
		}
	}
	return manager
}

// Retrieves session if not session is found a new one is created it
func Get(r *http.Request, w http.ResponseWriter, name string) *Session {
	mng := getManager()

	cookie, err := r.Cookie(name)
	if err != nil {
		if err == http.ErrNoCookie {
			sess, err := newSession(w, name)
			if err != nil {
				panic(err)
			}
			return sess
		}
	}

	//TODO: need to reverse to check store before checking memory
	mng.lock.Lock()
	defer mng.lock.Unlock()
	id := cookie.Value
	session, exists := mng.sessions[id]
	if !exists {
		var sess *Session
		if store != nil {
			sess, err = store.Read(id)
			if err == nil {
				mng.sessions[sess.ID] = sess
				return mng.sessions[sess.ID]
			}
		}
		sess, err := newSession(w, name)
		if err != nil {
			panic(err)
		}
		return sess
	}

	return session
}

// Creates a new session
func newSession(w http.ResponseWriter, name string) (*Session, error) {
	id, err := newSessionId()
	if err != nil {
		return nil, err
	}
	mng := getManager()
	session := &Session{
		ID:      id,
		Values:  make(map[string]interface{}),
		IsNew:   true,
		Expires: time.Now().Add(time.Second * time.Duration(options.MaxAge)),
	}
	mng.lock.Lock()
	defer mng.lock.Unlock()
	mng.sessions[id] = session
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    session.ID,
		HttpOnly: options.HttpOnly,
		Secure:   options.Secure,
		Expires:  time.Now().Add(time.Second * time.Duration(options.MaxAge)),
	})
	return session, nil
}

// Generates a new session id
func newSessionId() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func Delete(r *http.Request, w http.ResponseWriter, name string) {
	mng := getManager()

	cookie, err := r.Cookie(name)
	if err != nil {
		return
	} else {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Secure:   options.Secure,
			HttpOnly: options.HttpOnly,
			Expires:  time.Unix(0, 0),
			Path:     "/",
		})
	}

	mng.lock.Lock()
	defer mng.lock.Unlock()

	id := cookie.Value

	delete(mng.sessions, id)
	if store != nil {
		go store.Delete(id)
	}
}

// Clean up expired sessions
func CleanUpExpiredSessions() {
	for {
		time.Sleep(time.Duration(options.MaxAge) * time.Second)
		mng := getManager()
		mng.lock.Lock()
		now := time.Now()
		for k, v := range mng.sessions {
			if v.Expires.Before(now) {
				delete(mng.sessions, k)
				if store != nil {
					go store.Delete(k)
				}
			}
		}
		mng.lock.Unlock()
	}
}
