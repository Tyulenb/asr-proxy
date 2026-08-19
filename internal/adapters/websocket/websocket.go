package infrawebsocket

import (
	"net/http"
	"time"

	"github.com/Tyulenb/asr-proxy/internal/service"
	"github.com/gorilla/websocket"
)

const (
	ReadLimit       = 3200
	ReadBufferSize  = 4096
	WriteBufferSize = 4096
	startWait       = 15 * time.Second
	pongWait        = 60 * time.Second
	pingPeriod      = pongWait * 9 / 10
	writeWait       = 10 * time.Second
)

type Upgrader struct{}

func (u *Upgrader) CreateSession(w http.ResponseWriter, r *http.Request) (service.Session, error) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  ReadBufferSize,
		WriteBufferSize: WriteBufferSize,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	conn.SetReadLimit(ReadLimit)
	sess := session{conn: conn}
	return &sess, nil
}
