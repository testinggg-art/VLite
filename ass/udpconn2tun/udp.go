package udpconn2tun

import (
	"github.com/xiaokangwang/VLite/interfaces"
	"io"
	"net"
	"sync"
	"time"
)

func NewUDPConn2Tun(LocalTxToTun chan interfaces.UDPPacket, LocalRxFromTun chan interfaces.UDPPacket) *UDPConn2Tun {
	Conn := &UDPConn2Tun{
		LocalTxToTun:      LocalTxToTun,
		LocalRxFromTun:    LocalRxFromTun,
		remoteConnTracker: sync.Map{},
	}

	go Conn.RxLoop()

	return Conn
}

type UDPConn2Tun struct {
	LocalTxToTun   chan interfaces.UDPPacket
	LocalRxFromTun chan interfaces.UDPPacket

	remoteConnTracker sync.Map //key: RemoteAddr#string() value:connImpl
}

func (u *UDPConn2Tun) RxLoop() {
	for {
		select {
		case pack := <-u.LocalTxToTun:
			v, ok := u.remoteConnTracker.Load(pack.Dest.String())
			if !ok {
				//We cannot process this packet and it have to be discarded
				continue
			}

			vn := v.(*connImpl)
			vn.readchan <- pack
		}
	}
}
func (u *UDPConn2Tun) DialUDP(ouraddr net.UDPAddr) net.PacketConn {
	imp := &connImpl{}
	imp.remoteAddr = &ouraddr
	imp.readchan = make(chan interfaces.UDPPacket, 8)
	imp.server = u
	u.remoteConnTracker.Store(ouraddr.String(), imp)
	return imp
}

type connImpl struct {
	server     *UDPConn2Tun
	remoteAddr net.Addr
	readchan   chan interfaces.UDPPacket
}

func (c connImpl) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	select {
	case by, more := <-c.readchan:
		if !more {
			return 0, nil, io.ErrClosedPipe
		}
		copy(b, by.Payload)
		return len(by.Payload), by.Source, nil
	case <-time.Tick(time.Second * 1200):
		return 0, nil, io.ErrClosedPipe
	}

}

func (c connImpl) WriteTo(b []byte, addr net.Addr) (n int, err error) {

	if addr.(*net.UDPAddr).IP.To4() == nil {
		pack := interfaces.UDPPacket{
			Source:  c.remoteAddr.(*net.UDPAddr),
			Dest:    addr.(*net.UDPAddr),
			Payload: b,
		}
		c.server.LocalRxFromTun <- pack
	} else {
		XSource := c.remoteAddr.(*net.UDPAddr)
		XSource.IP = net.IPv6zero
		pack := interfaces.UDPPacket{
			Source:  XSource,
			Dest:    addr.(*net.UDPAddr),
			Payload: b,
		}
		c.server.LocalRxFromTun <- pack
	}

	return len(b), nil
}

func (c connImpl) Close() error {
	c.server.remoteConnTracker.Delete(c.remoteAddr.String())
	close(c.readchan)
	return nil
}

func (c connImpl) LocalAddr() net.Addr {
	panic("implement me")
}

func (c connImpl) RemoteAddr() net.Addr {
	panic("implement me")
}

func (c connImpl) SetDeadline(t time.Time) error {
	return nil
}

func (c connImpl) SetReadDeadline(t time.Time) error {
	return nil
}

func (c connImpl) SetWriteDeadline(t time.Time) error {
	return nil
}
