package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Egor-Evsikov/crschback/src/db"
)

func UserLogin(w http.ResponseWriter, r *http.Request) {
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

func UserRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {

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
