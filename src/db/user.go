package db

type User struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func NewUser(log string, pass string) User {
	return User{
		log,
		pass,
	}
}
