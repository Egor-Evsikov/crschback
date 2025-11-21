package db

type Dir struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Owner string `json:"login"`
}

func NewDir(id int, name string, own string) Dir {
	return Dir{
		Id:    id,
		Name:  name,
		Owner: own,
	}
}
