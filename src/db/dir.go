package db

type Dir struct {
	Name  string `json:"name"`
	Owner string `json:"login"`
}

func NewDir(name string, own string) Dir {
	return Dir{name, own}
}
