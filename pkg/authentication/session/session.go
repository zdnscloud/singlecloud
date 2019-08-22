package session

import (
	"net/http"
	"sync"

	"github.com/zdnscloud/cement/uuid"
)

type SessionMgr struct {
	key      string
	lock     sync.Mutex
	sessions map[string]string
}

func New(key string) *SessionMgr {
	return &SessionMgr{
		key:      key,
		sessions: make(map[string]string),
	}
}

func (mgr *SessionMgr) GetSession(r *http.Request) (string, error) {
	c, err := r.Cookie(mgr.key)
	if err != nil {
		return "", nil
	}

	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	session, ok := mgr.sessions[c.Value]
	if ok == false {
		return "", nil
	} else {
		return session, nil
	}
}

func (mgr *SessionMgr) AddSession(w http.ResponseWriter, r *http.Request, value string) {
	sessionKey := uuid.MustGen()
	c := &http.Cookie{
		Name:     mgr.key,
		Value:    sessionKey,
		Path:     "/",
		HttpOnly: true,
	}
	r.AddCookie(c)
	http.SetCookie(w, c)

	mgr.lock.Lock()
	mgr.sessions[sessionKey] = value
	mgr.lock.Unlock()
}

func (mgr *SessionMgr) ClearSession(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(mgr.key)
	if err != nil {
		return
	}

	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	delete(mgr.sessions, c.Value)
	c.MaxAge = -1
	http.SetCookie(w, c)
}
