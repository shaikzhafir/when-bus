package handlers

import (
	"net/http"

	store "github.com/shaikzhafir/go-htmx-starter/shared"

	"github.com/dghubble/gologin/v2/google"
	"github.com/dghubble/sessions"
)

type authHandler struct {
	store sessions.Store[string]
}

type AuthHandler interface {
	LoginCallback() http.HandlerFunc
	Logout() http.HandlerFunc
}

func NewAuthHandler(store sessions.Store[string]) AuthHandler {
	return &authHandler{
		store: store,
	}
}

func (a *authHandler) LoginCallback() http.HandlerFunc {
	// handle the request
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		googleUser, err := google.UserFromContext(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// 2. Implement a success handler to issue some form of session
		session := a.store.New(store.SessionName)
		session.Set(store.SessionUserKey, googleUser.Id)
		session.Set(store.SessionUsername, googleUser.Name)
		session.Set("googleEmail", googleUser.Email)
		session.Set("googleAvatar", googleUser.Picture)
		if err := session.Save(w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/profile", http.StatusFound)
	}
}

func (a *authHandler) Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			a.store.Destroy(w, store.SessionName)
		}
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
