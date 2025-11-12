package server

import (
	"net/http"

	"github.com/Egor-Evsikov/crschback/src/api"
	"github.com/Egor-Evsikov/crschback/src/db"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type RestApiServer struct {
	cfg      *ServerConfig
	logger   *logrus.Logger
	router   *chi.Mux
	handlers *api.HandlerDB
}

func NewServer(c *ServerConfig, d *db.Repo) *RestApiServer {
	return &RestApiServer{c, logrus.New(), chi.NewRouter(), api.NewHandlerDB(d)}
}

func (s *RestApiServer) Start() error {
	err := s.configureLogger()
	if err != nil {
		return err
	}
	err = s.configureRouter()
	if err != nil {
		return err
	}
	s.logger.Info("Сервер запущен на ", s.cfg.Addr)

	return http.ListenAndServe(s.cfg.Addr, s.router)
}

func (s *RestApiServer) configureLogger() error {
	level, err := logrus.ParseLevel(s.cfg.LogLever)
	if err != nil {
		return err
	}
	s.logger.SetLevel(level)
	return nil
}

func (s *RestApiServer) configureRouter() error {
	s.router.Post("/register", s.handlers.UserRegister)
	return nil
}
