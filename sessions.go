// Package dumbo implements a simple session manager to keep state
// of an application
package dumbo

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"reflect"
	"sync"
	"time"
)

var manager *SessionManager
var options *Options
var store Store

type Options struct {
	MaxAge   time.Duration // max age
	Secure   bool          // secure cookie
	HttpOnly bool          // http  only cookie
}

// Session value
type Value struct {
	Key     string
	Content any
}

func (v Value) IsNil() bool {
	return v.Content == nil
}

type Session struct {
	ID      string                 `bson:"_id" json:"id" sql:"column:id;type:varchar(500)"`
	Name    string                 `bson:"name" json:"name" sql:"column:name;type:varchar(500)"`
	Values  map[string]interface{} `bson:"values" json:"values" sql:"column:values"`
	IsNew   bool                   `bson:"isnew" json:"isnew" sql:"column:isnew;type:bit"`
	Expires time.Time              `bson:"expires" json:"expires" sql:"column:expires;type:datetime"`
}

// Saves changes to session
func (ses *Session) Save(w http.ResponseWriter) error {
	ses.Expires = time.Now().Add(options.MaxAge)
	if store != nil {
		go store.Save(ses)
	}
	touch(w, ses)
	return nil
}

// Set [key] and [val] and saves session
func (ses *Session) SaveMustCompile(key string, val any, w http.ResponseWriter) {
	if key != "" && val != nil {
		ses.Values[key] = val
	}
	if err := ses.Save(w); err != nil {
		panic(err)
	}
}

// Set values from [Value]
func (ses *Session) SetValue(val Value, w http.ResponseWriter) error {
	if !val.IsNil() {
		//BUG: Undeterministic bejhavior
		//When saving an struct in memory the struct type is retain
		//when session is loaded from database in this case mongo driver it changes to a
		//map[string]interface creating undeterministic behavior
		ses.Values[val.Key] = val.Content
	}
	return ses.Save(w)
}

func (ses *Session) SetValueMustCompile(val Value, w http.ResponseWriter) {
	if !val.IsNil() {
		ses.Values[val.Key] = val.Content
	}

	if err := ses.Save(w); err != nil {
		panic(err)
	}
}

// Get stored value resturns nil if none found
func (ses *Session) Get(key string) Value {
	v, exists := ses.Values[key]
	if !exists {
		return Value{key, nil}
	}
	return Value{key, v}
}

// Get value from session and decodes it into [val]
func (ses *Session) GetDecode(key string, val any) error {
	v := ses.Get(key)
	if v.IsNil() {
		return ErrValIsNil
	}

	rv := reflect.ValueOf(val)
	if rv.Kind() != reflect.Pointer {
		return ErrValIsNotPointer
	}
	elem := rv.Elem()
	if !elem.CanSet() {
		return ErrCantSetValue
	}

	source := reflect.ValueOf(v.Content)
	var source_elem reflect.Value
	if source.Kind() == reflect.Pointer {
		source_elem = source.Elem()
	} else {
		source_elem = source
	}

	if elem.Kind() != source_elem.Kind() && elem.Kind() != reflect.Struct {
		return ErrNoTypeMatch
	}

	elem.Set(source_elem)

	return nil
}

// Changes the expiration time for a session
func touch(w http.ResponseWriter, sess *Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     sess.Name,
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: options.HttpOnly,
		Secure:   options.Secure,
		Expires:  time.Now().Add(options.MaxAge),
	})
}

type SessionManager struct {
	sessions map[string]*Session
	lock     sync.Mutex
}

// Starts a session manager with specifies opt options and s store mechanism
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
	//not sure what the above comment means

	id := cookie.Value
	session, exists := mng.sessions[id]
	if !exists {
		var sess *Session
		if store != nil {
			sess, err = store.Read(id)
			if err == nil {
				mng.lock.Lock()
				mng.sessions[sess.ID] = sess
				mng.lock.Unlock()
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

// Retrieves session if not session is found a new oner is created. You can specify duration via dur which will
// apply a duration seperate to the global setting
func GetWithDuration(w http.ResponseWriter, r *http.Request, name string, dur time.Duration) *Session {
	mng := getManager()

	cookie, err := r.Cookie(name)
	if err != nil {
		if err == http.ErrNoCookie {
			sess, err := newSessionWithDuration(w, name, dur)
			if err != nil {
				panic(err)
			}
			return sess
		}
	}

	//TODO: need to reverse to check store before checking memory
	//not sure what the above comment means

	id := cookie.Value
	session, exists := mng.sessions[id]
	if !exists {
		var sess *Session
		if store != nil {
			sess, err = store.Read(id)
			if err == nil {
				mng.lock.Lock()
				mng.sessions[sess.ID] = sess
				mng.lock.Unlock()
				return mng.sessions[sess.ID]
			}
		}
		sess, err := newSessionWithDuration(w, name, dur)
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
		Name:    name,
		Values:  make(map[string]interface{}),
		IsNew:   true,
		Expires: time.Now().Add(options.MaxAge),
	}
	mng.lock.Lock()
	defer mng.lock.Unlock()
	mng.sessions[id] = session
	touch(w, session)
	return session, nil
}

// Creates a new session with custom duration
func newSessionWithDuration(w http.ResponseWriter, name string, dur time.Duration) (*Session, error) {
	id, err := newSessionId()
	if err != nil {
		return nil, err
	}

	if dur == 0 {
		dur = options.MaxAge
	}

	mng := getManager()
	session := &Session{
		ID:      id,
		Name:    name,
		Values:  make(map[string]interface{}),
		IsNew:   true,
		Expires: time.Now().Add(dur),
	}

	mng.lock.Lock()
	defer mng.lock.Unlock()
	mng.sessions[id] = session
	touch(w, session)
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

// Deletes session by name
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
		//OPTIMIZE: Probably need to take a look at when to run the clean up better
		// sleep should not be based on expiration intervals
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
		time.Sleep(options.MaxAge)
	}
}
