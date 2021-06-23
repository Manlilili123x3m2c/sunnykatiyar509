package core

import (
	"context"
	"sync"
)

var Manager = &manager{
	container: new(sync.Map),
}

type manager struct {
	container *sync.Map
}

func (m *manager) add(uHome SessionHome) {
	m.container.Store(uHome.SessionID(), uHome)

}

func (m *manager) delete(roomID string) {
	m.container.Delete(roomID)

}

func (m *manager) search(roomID string) (SessionHome, bool) {
	if uHome, ok := m.container.Load(roomID); ok {
		return uHome.(SessionHome), ok
	}
	return nil, false
}

func (m *manager) JoinShareRoom(roomID string, uConn Conn) {
	if userHome, ok := m.search(roomID); ok {
		userHome.AddConnection(uConn)
	}
}

func (m *manager) ExitShareRoom(roomID string, uConn Conn) {
	if userHome, ok := m.search(roomID); ok {
		userHome.RemoveConnection(uConn)
	}

}

func (m *manager) Switch(ctx context.Context, userHome SessionHome, pChannel ProxyChannel) error {
	m.add(userHome)
	defer m.delete(userHome.SessionID())

	subCtx, cancelFunc := context.WithCancel(ctx)
	userSendRequestStream := userHome.SendRequestChannel(subCtx)
	userReceiveStream := userHome.ReceiveResponseChannel(subCtx)
	nodeRequestChan := pChannel.ReceiveRequestChannel(subCtx)
	nodeSendResponseStream := pChannel.SendResponseChannel(subCtx)

	for userSendRequestStream != nil || nodeSendResponseStream != nil {
		select {
		case buf1, ok := <-userSendRequestStream:
			if !ok {
				log.Warn("userSendRequestStream close")
				userSendRequestStream = nil
				continue
			}
			nodeRequestChan <- buf1
		case buf2, ok := <-nodeSendResponseStream:
			if !ok {
				log.Warn("nodeSendResponseStream close")
				nodeSendResponseStream = nil
				close(userReceiveStream)
				cancelFunc()
				continue
			}
			userReceiveStream <- buf2
		case <-ctx.Done():
			return nil
		}
	}
	log.Info("switch end")
	return nil
}
