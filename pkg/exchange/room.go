package exchange

import (
	"container/ring"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/jumpserver/koko/pkg/common"
	"github.com/jumpserver/koko/pkg/logger"
)

type RoomManager interface {
	Add(s *Room)
	Delete(s *Room)
	Get(sid string) *Room
}

var (
	_ RoomManager = (*localRoomManager)(nil)
	_ RoomManager = (*redisRoomManager)(nil)
)

func CreateRoom(id string, inChan chan *RoomMessage) *Room {
	s := &Room{
		Id:             id,
		userInputChan:  inChan,
		broadcastChan:  make(chan *RoomMessage),
		subscriber:     make(chan *Conn),
		unSubscriber:   make(chan *Conn),
		exitSignal:     make(chan struct{}),
		done:           make(chan struct{}),
		recentMessages: ring.New(5),
	}
	return s
}

type Room struct {
	Id string

	userInputChan chan *RoomMessage

	broadcastChan chan *RoomMessage

	subscriber chan *Conn

	unSubscriber chan *Conn

	exitSignal chan struct{}

	done chan struct{}

	once sync.Once

	recentMessages *ring.Ring
}

func (r *Room) run() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	defer r.closeOnce()
	connMaps := make(map[string]*Conn)
	for {
		select {
		case <-ticker.C:
			if len(connMaps) == 0 {
				logger.Infof("Room %s has no connection now and exit", r.Id)
				return
			}
			select {
			case <-r.done:
				for k := range connMaps {
					_ = connMaps[k].Close()
				}
			default:
			}
		case con := <-r.subscriber:
			connMaps[con.Id] = con
			r.recentMessages.Do(func(value interface{}) {
				if msg, ok := value.(*RoomMessage); ok {
					switch msg.Event {
					case DataEvent:
						_, _ = con.Write(msg.Body)
					}
				}
			})
			logger.Debugf("Room %s current connections count: %d", r.Id, len(connMaps))
		case con := <-r.unSubscriber:
			delete(connMaps, con.Id)
			logger.Debugf("Room %s current connections count: %d", r.Id, len(connMaps))
		case msg := <-r.broadcastChan:
			userConns := make([]*Conn, 0, len(connMaps))
			for k := range connMaps {
				userConns = append(userConns, connMaps[k])
			}
			switch msg.Event {
			case DataEvent:
				r.recentMessages.Value = msg
				r.recentMessages = r.recentMessages.Next()
			}
			r.broadcastMessage(userConns, msg)

		case <-r.exitSignal:
			for k := range connMaps {
				_ = connMaps[k].Close()
			}
		}
	}
}

func (r *Room) Subscribe(conn *Conn) {
	r.subscriber <- conn

}

func (r *Room) UnSubscribe(conn *Conn) {
	r.unSubscriber <- conn
}

func (r *Room) Broadcast(msg *RoomMessage) {
	select {
	case <-r.done:
	case r.broadcastChan <- msg:
	}
}

func (r *Room) Receive(msg *RoomMessage) {
	select {
	case <-r.done:
	case r.userInputChan <- msg:
	}
}

func (r *Room) broadcastMessage(conns userConnections, msg *RoomMessage) {
	// ????????????goroutine?????????
	if len(conns) == 0 {
		return
	}
	if len(conns) == 1 {
		conns[0].handlerMessage(msg)
		return
	}

	// ?????? goroutine ????????????
	sort.Sort(conns)
	var wg sync.WaitGroup
	for i := range conns {
		wg.Add(1)
		go func(con *Conn) {
			defer wg.Done()
			con.handlerMessage(msg)
		}(conns[i])
	}
	wg.Wait()
}

func (r *Room) Done() <-chan struct{} {
	return r.done
}

func (r *Room) stop() {
	select {
	case <-r.done:
		return
	case r.exitSignal <- struct{}{}:
	}
	r.closeOnce()
}

func (r *Room) closeOnce() {
	r.once.Do(func() {
		close(r.done)
	})
}

func WrapperUserCon(stream io.WriteCloser) *Conn {
	return &Conn{
		Id:          common.UUID(),
		WriteCloser: stream,
		created:     time.Now(),
	}
}

type Conn struct {
	Id string
	io.WriteCloser
	created time.Time
}

func (c *Conn) handlerMessage(msg *RoomMessage) {
	switch msg.Event {
	case DataEvent:
		_, _ = c.Write(msg.Body)
	case PingEvent:
		_, _ = c.Write(nil)
	}
}

var _ sort.Interface = (userConnections)(nil)

type userConnections []*Conn

func (l userConnections) Less(i, j int) bool {
	return l[i].created.Before(l[j].created)
}

func (l userConnections) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l userConnections) Len() int {
	return len(l)
}
