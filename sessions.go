package dumbo

import (
	"net/http"
	"sync"
	"time"
)

var manager *SessionManager

type Options struct {
	MaxAge   int
	Secure   bool
	HttpOnly bool
}

type Session struct {
	ID      string
	Values  map[string]interface{}
	IsNew   bool
	Expires time.Time
	store   Store
}

type SessionManager struct {
	sessions map[string]*Session
	lock     sync.Mutex
}

// Get Session manager
func startManager() *SessionManager {
	if manager == nil {
		//TODO: IF INITIALLIZE START CLEAN UP ROUTINE
		manager = &SessionManager{
			sessions: make(map[string]*Session),
		}
	}
	return manager
}

// TODO: FINISH GET FUNCTION
func Get(r http.Request, name string) *Session {
	if manager == nil {
		startManager()
	}

	cookie, err := r.Cookie(name)
	if err != nil {

	}
}

//TODO: DELETE FUNCTION SESSION
