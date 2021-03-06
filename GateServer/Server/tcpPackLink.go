package Server

import (
	. "GameServer/GateServer/config"
	"GameServer/Pack"
	gLog "GameServer/gameLog"
	"GameServer/utils"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

type tcpPackLink struct {
	sid        SocketIdType
	server     *TcpPackServer
	conn       *net.TCPConn
	wtSyncChan chan []byte
}

func NewPackLink(sid SocketIdType, svr *TcpPackServer, co *net.TCPConn) *tcpPackLink {
	lk := new(tcpPackLink)
	lk.sid = sid
	lk.server = svr
	lk.conn = co
	lk.wtSyncChan = make(chan []byte, WRITE_PACK_SYNC_CHAN_SIZE)
	return lk
}

func (lk *tcpPackLink) Close() {
	defer func() {
		gLog.Info(fmt.Sprintf("disconnected: %s sid: %d mapCount: %d ", lk.conn.RemoteAddr().String(), lk.sid, len(lk.server.linkMap)))
	}()
	defer utils.PrintPanicStack()
	////////////////////////////////////////////////////////////////////

	lk.conn.Close()
	close(lk.wtSyncChan)
}

func (lk *tcpPackLink) PutBytes(b []byte) error {
	select {
	case lk.wtSyncChan <- b:
		return nil
	case <-time.After(time.Second * WRITE_PACK_SYNC_CHAN_TIMEOUT):
		return errors.New("put wtSyncChan timeout.")
	}
}

func (lk *tcpPackLink) StartReadPack() {
	defer func() {
		lk.server.RemoveLink(lk.sid)
	}()
	defer utils.PrintPanicStack()
	////////////////////////////////////////////////////////////////////

	sizeBuf := make([]byte, PACK_DATA_SIZE_TYPE_LEN)
	var dataSize uint32
	var sizeBufIdx uint32
	var dataBufIdx uint32
	for {
		sizeBufIdx = 0
		for {
			err := lk.conn.SetReadDeadline(time.Now().Add(TCP_READ_TIMEOUT * time.Second))
			if err != nil {
				panic(err)
			}
			n, err := lk.conn.Read(sizeBuf[sizeBufIdx:])
			if err != nil {
				if err == io.EOF {
					gLog.Info(fmt.Sprintf("tcp read EOF. sid: %d ", lk.sid))
					return
				} else {
					panic(err)
				}
			}
			sizeBufIdx += uint32(n)
			if sizeBufIdx == PACK_DATA_SIZE_TYPE_LEN {
				break
			} else if sizeBufIdx > PACK_DATA_SIZE_TYPE_LEN || sizeBufIdx < 0 {
				panic("read pack data size error.")
			}
		}

		dataSize = binary.BigEndian.Uint32(sizeBuf)
		if dataSize > MAX_INBOUND_PACK_DATA_SIZE {
			panic("read pack data out of limit.")
		} else if dataSize == 0 {
			panic("read pack data size equals 0.")
		}

		dataBufIdx = 0
		data := make([]byte, dataSize)
		for {
			err := lk.conn.SetReadDeadline(time.Now().Add(TCP_READ_TIMEOUT * time.Second))
			if err != nil {
				panic(err)
			}
			n, err := lk.conn.Read(data[dataBufIdx:])
			if err != nil {
				if err == io.EOF {
					gLog.Info(fmt.Sprintf("tcp read EOF. sid: %d ", lk.sid))
					return
				} else {
					panic(err)
				}
			}

			dataBufIdx += uint32(n)

			if dataBufIdx == dataSize {
				lk.server.RoutePackIn(Pack.NewPack(lk.sid, data))
				break
			} else if dataBufIdx > dataSize || dataBufIdx < 0 {
				panic("read pack data error.")
			}
		}

	}
}

func (lk *tcpPackLink) StartWritePack() {
	defer func() {
		lk.server.RemoveLink(lk.sid)
	}()
	defer utils.PrintPanicStack()
	////////////////////////////////////////////////////////////////////

	var dataSize uint32
	var sizeBufIdx uint32
	var dataBufIdx uint32
	for rawData := range lk.wtSyncChan {
		n := len(rawData)
		if n > MAX_OUTBOUND_PACK_DATA_SIZE {
			panic("write pack data out of limit.")
		} else if n == 0 {
			panic("write pack data size equals 0.")
		}

		dataSize = uint32(n)
		sizeBytes := make([]byte, PACK_DATA_SIZE_TYPE_LEN)
		binary.BigEndian.PutUint32(sizeBytes, uint32(n))

		sizeBufIdx = 0
		for {
			err := lk.conn.SetWriteDeadline(time.Now().Add(TCP_WRITE_TIMEOUT * time.Second))
			if err != nil {
				panic(err)
			}

			n, err := lk.conn.Write(sizeBytes[sizeBufIdx:])
			if err != nil {
				panic(err)
			}

			sizeBufIdx += uint32(n)
			if sizeBufIdx == PACK_DATA_SIZE_TYPE_LEN {
				break
			} else if sizeBufIdx > PACK_DATA_SIZE_TYPE_LEN {
				panic("write pack data size error.")
			}
		}

		dataBufIdx = 0
		for {
			err := lk.conn.SetWriteDeadline(time.Now().Add(TCP_WRITE_TIMEOUT * time.Second))
			if err != nil {
				panic(err)
			}

			n, err := lk.conn.Write(rawData)
			if err != nil {
				panic(err)
			}

			dataBufIdx += uint32(n)

			if dataBufIdx == dataSize {
				break
			} else if dataBufIdx > dataSize || dataBufIdx < 0 {
				panic("write pack data error.")
			}
		}
	}
}
