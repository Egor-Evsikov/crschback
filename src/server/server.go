package server

import (
	"log"
	"net/http"

	"github.com/Egor-Evsikov/crschback/src/api"
	"github.com/Egor-Evsikov/crschback/src/db"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type RestApiServer struct {
	cfg      *ServerConfig
	router   *chi.Mux
	handlers *api.HandlerDB
}

func NewServer(c *ServerConfig, d *db.Repo) *RestApiServer {
	return &RestApiServer{c, chi.NewRouter(), api.NewHandlerDB(d)}
}

func (s *RestApiServer) Start() error {

	err := s.configureRouter()
	if err != nil {
		return err
	}
	log.Print("Сервер запущен на ", s.cfg.Addr)

	return http.ListenAndServe(s.cfg.Addr, s.router)
}

func (s *RestApiServer) configureRouter() error {
	s.router.Use(middleware.Logger)
	s.router.Use(api.AuthMiddleware)
	s.router.Post("/login", s.handlers.UserLogin)
	s.router.Post("/register", s.handlers.UserRegister)
	s.router.Post("/directories", s.handlers.Mkdir)
	s.router.Get("/directories", s.handlers.GetDirs)
	s.router.Delete("/directories", s.handlers.DeleteDir)
	s.router.Post("/directories/{id}/users", s.handlers.AddUserToDir)
	s.router.Delete("/directories/{id}/users/{login}", s.handlers.RemoveUserFromDir)
	s.router.Get("/directories/{id}/medicines", s.handlers.GetMeds)
	s.router.Post("/directories/{id}/medicines", s.handlers.AddMedicine)
	s.router.Get("/directories/{id}/medicines/{med_id}", s.handlers.GetMedicine)
	s.router.Put("/directories/{id}/medicines/{med_id}", s.handlers.UpdateMedicine)
	s.router.Delete("/directories/{id}/medicines/{med_id}", s.handlers.DeleteMedicine)
	s.router.Get("/directories/{id}/users", s.handlers.GetUsersInDir)
	return nil
}
