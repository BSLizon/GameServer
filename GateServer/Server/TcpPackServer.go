package Server

import (
	"GameServer/GateServer/config"
	g "GameServer/GateServer/socketIdGenerator"
	gLog "GameServer/gameLog"
	"GameServer/utils"
	"errors"
	"fmt"
	"net"
	"sync"
)

type TcpPackServer struct {
	sync.RWMutex
	linkMap map[config.SocketIdType]*tcpPackLink
}

func NewTcpPackServer() *TcpPackServer {
	svr := new(TcpPackServer)
	svr.linkMap = make(map[config.SocketIdType]*tcpPackLink)
	return svr
}

func (svr *TcpPackServer) PutLink(i config.SocketIdType, lk *tcpPackLink) error {
	if len(svr.linkMap) > config.MAX_TCP_CONN {
		return errors.New("tcp conn limit")
	}

	if _, ok := svr.GetLink(i); ok {
		return errors.New("sid conflict")
	}

	svr.Lock()
	defer func() {
		svr.Unlock()
	}()
	svr.linkMap[i] = lk

	return nil
}

func (svr *TcpPackServer) GetLink(i config.SocketIdType) (*tcpPackLink, bool) {
	svr.RLock()
	defer func() {
		svr.RUnlock()
	}()
	c, ok := svr.linkMap[i]
	return c, ok
}

// 会关闭连接
func (svr *TcpPackServer) RemoveLink(i config.SocketIdType) {
	lk, ok := svr.GetLink(i)
	if ok {
		svr.Lock()
		defer func() {
			svr.Unlock()
		}()
		delete(svr.linkMap, i)
		gLog.Info(fmt.Sprintf("has been removed: %s sid: %d mapCount: %d ", lk.conn.RemoteAddr().String(), lk.sid, len(lk.server.linkMap)))
		lk.Close()
	}
}

// 复制一份linkMap，用于广播
func (svr *TcpPackServer) GetLinkMapCopy() map[config.SocketIdType]*tcpPackLink {
	linkMap := make(map[config.SocketIdType]*tcpPackLink)
	svr.RWMutex.Lock()
	defer func() {
		svr.RWMutex.Unlock()
	}()
	for k, v := range svr.linkMap {
		linkMap[k] = v
	}
	return linkMap
}

func (svr *TcpPackServer) Start() {
	tcpAddr, err := net.ResolveTCPAddr("tcp", ":"+config.EXTERNAL_LISTEN_PORT)
	if err != nil {
		gLog.Fatal(err)
	}

	tcpListener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		gLog.Fatal(err)
	}
	defer tcpListener.Close()

	gLog.Info("listen on: " + config.EXTERNAL_LISTEN_PORT)

	for {
		tcpConn, err := tcpListener.AcceptTCP()
		if err != nil {
			gLog.Warn(err)
			continue
		}

		gLog.Info(fmt.Sprintf("connected: %s mapCount: %d ", tcpConn.RemoteAddr().String(), len(svr.linkMap)))
		go svr.handleTcpConn(tcpConn)
	}
}

func (svr *TcpPackServer) handleTcpConn(tcpConn *net.TCPConn) {
	defer utils.PrintPanicStack()
	////////////////////////////////////////////////////////////////////

	sid := g.Get()

	lk := NewPackLink(sid, svr, tcpConn)
	err := svr.PutLink(sid, lk)
	if err != nil {
		lk.Close()
		gLog.Warn(fmt.Sprintf("%s disconnected: %s sid: %d mapCount: %d ", err.Error(), lk.conn.RemoteAddr().String(), lk.sid, len(lk.server.linkMap)))
		return
	}

	go lk.StartReadPack()
	go lk.StartWritePack()
	gLog.Info(fmt.Sprintf("serving: %s sid: %d mapCount: %d ", tcpConn.RemoteAddr().String(), lk.sid, len(lk.server.linkMap)))
}
