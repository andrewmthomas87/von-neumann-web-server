package game

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
	"golang.org/x/sync/errgroup"
)

type Server interface {
	ID() string
	Run(ctx context.Context) error
	Connect(sd *webrtc.SessionDescription) (*webrtc.SessionDescription, error)
}

type message struct {
	ID      string `json:"id,omitempty"`
	Payload []byte `json:"payload,omitempty"`
}

type connectRequest struct {
	sd      *webrtc.SessionDescription
	res     chan *webrtc.SessionDescription
	errored chan error
}

type server struct {
	id           string
	c            *websocket.Conn
	in           chan []byte
	out          chan []byte
	pending      map[string]chan []byte
	connectQueue chan connectRequest
}

func (s *server) ID() string {
	return s.id
}

func (s *server) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		for {
			_, msg, err := s.c.ReadMessage()
			if err != nil {
				return err
			}

			s.in <- msg
		}
	})

	g.Go(func() error {
		for msg := range s.out {
			if err := s.c.WriteMessage(websocket.TextMessage, msg); err != nil {
				return err
			}
		}
		return nil
	})

	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				close(s.in)
				close(s.out)
				close(s.connectQueue)
				s.pending = make(map[string]chan []byte)
				return nil
			case msg := <-s.in:
				var m message
				if err := json.Unmarshal(msg, &m); err != nil {
					return err
				}

				res, ok := s.pending[m.ID]
				if !ok {
					fmt.Println("WARN: received message with unknown id")
					continue
				}
				delete(s.pending, m.ID)

				res <- m.Payload
			case cr := <-s.connectQueue:
				id := uuid.NewString()
				res := make(chan []byte, 1)

				b, err := marshalMessage(id, cr.sd)
				if err != nil {
					cr.errored <- err
					continue
				}
				s.out <- b

				s.pending[id] = res

				go func() {
					select {
					case <-ctx.Done():
						return
					case b := <-res:
						var sd webrtc.SessionDescription
						if err := json.Unmarshal(b, &sd); err != nil {
							cr.errored <- err
							return
						}
						cr.res <- &sd
					}
				}()
			}
		}
	})

	return g.Wait()
}

func (s *server) Connect(sd *webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	res := make(chan *webrtc.SessionDescription, 1)
	errored := make(chan error, 1)
	s.connectQueue <- connectRequest{sd, res, errored}

	select {
	case err := <-errored:
		return nil, err
	case sd := <-res:
		return sd, nil
	}
}

func NewServer(c *websocket.Conn) Server {
	return &server{
		id:           uuid.NewString(),
		c:            c,
		in:           make(chan []byte),
		out:          make(chan []byte),
		pending:      make(map[string]chan []byte),
		connectQueue: make(chan connectRequest),
	}
}

func marshalMessage(id string, payload interface{}) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	m := message{id, b}
	b, err = json.Marshal(m)
	if err != nil {
		return nil, err
	}

	return b, nil
}
