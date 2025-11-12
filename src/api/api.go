package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Egor-Evsikov/crschback/src/db"
)

type HandlerDB struct {
	dtbs *db.Repo
}

func NewHandlerDB(d *db.Repo) *HandlerDB {
	return &HandlerDB{d}
}

func (h *HandlerDB) UserLogin() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var user db.User
			err := json.NewDecoder(r.Body).Decode(&user)
			if err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			if user == db.NewUser("aaa", "bbb") {
				fmt.Fprintln(w, "success", http.StatusAccepted)
			}
		}

	}

}

func (h *HandlerDB) UserRegister() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var user db.User
			err := json.NewDecoder(r.Body).Decode(&user)
			if err != nil {
				http.Error(w, "Invalid JSON", http.StatusUnsupportedMediaType)
			}

			h.dtbs.SaveUser(user.Login, user.Password)

		}
	}
}

func GetDirs(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {

	}

}

func Mkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {

	}
}

func ChangePass(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPut {

	}
}

func DeleteDir(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {

	}
}
