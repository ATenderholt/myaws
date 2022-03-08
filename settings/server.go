package settings

import (
	"strconv"
)

type Server struct {
	Protocol string
	Host     string
	Port     int
}

func (server *Server) BuildUrl(path string) string {
	return server.Protocol + "://" + server.Host + ":" + strconv.Itoa(server.Port) + path
}

func NewLocalhostServer(port int) *Server {
	return &Server{Protocol: "http", Host: "localhost", Port: port}
}
