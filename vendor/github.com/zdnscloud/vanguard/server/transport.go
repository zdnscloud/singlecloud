package server

import (
	"net"
	"sync/atomic"

	g53util "github.com/zdnscloud/g53/util"
	"github.com/zdnscloud/vanguard/config"
	"github.com/zdnscloud/vanguard/logger"
	"github.com/zdnscloud/vanguard/util"
)

const (
	maxConcurrentTCPConn = 512
	maxQueryLen          = 512
	udpReceiveBuf        = 1024 * maxQueryLen
	maxBufferFullCount   = 5
)

type Transport struct {
	udpConns        []*net.UDPConn
	tcpListeners    []*net.TCPListener
	tcpConnCount    int32
	udpBufPool      *util.BytePool
	bufferFullCount int
}

func newTransport(conf *config.VanguardConf, handlerCount int) (*Transport, error) {
	t := &Transport{}
	if err := t.openUDP(conf); err != nil {
		t.Close()
		return nil, err
	}

	if err := t.openTCP(conf); err != nil {
		t.Close()
		return nil, err
	}

	t.udpBufPool = util.NewBytePool(handlerCount, maxQueryLen)
	return t, nil
}

func (t *Transport) openUDP(conf *config.VanguardConf) error {
	var udpAddrs []*net.UDPAddr
	for _, addr := range conf.Server.Addrs {
		udpAddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return err
		}

		if udpAddr.IP.IsUnspecified() {
			for _, ip := range getAllIPs(udpAddr.IP.To4() != nil) {
				udpAddrs = append(udpAddrs, &net.UDPAddr{
					IP:   ip,
					Port: udpAddr.Port,
					Zone: udpAddr.Zone,
				})
			}
		} else {
			udpAddrs = append(udpAddrs, udpAddr)
		}
	}

	return t.bindUPDAddresses(udpAddrs)
}

func (t *Transport) bindUPDAddresses(addrs []*net.UDPAddr) error {
	conns := []*net.UDPConn{}
	for _, addr := range addrs {
		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			return err
		}
		conn.SetReadBuffer(udpReceiveBuf)
		conns = append(conns, conn)
	}
	t.udpConns = conns
	return nil
}

func (t *Transport) openTCP(conf *config.VanguardConf) error {
	if conf.Server.EnableTCP == false {
		return nil
	}

	var tcpAddrs []*net.TCPAddr
	for _, addr := range conf.Server.Addrs {
		tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			return err
		}

		if tcpAddr.IP.IsUnspecified() {
			for _, ip := range getAllIPs(tcpAddr.IP.To4() != nil) {
				tcpAddrs = append(tcpAddrs, &net.TCPAddr{
					IP:   ip,
					Port: tcpAddr.Port,
					Zone: tcpAddr.Zone,
				})
			}
		} else {
			tcpAddrs = append(tcpAddrs, tcpAddr)
		}
	}

	return t.bindTCPAddresses(tcpAddrs)
}

func (t *Transport) bindTCPAddresses(addrs []*net.TCPAddr) error {
	tcpListeners := []*net.TCPListener{}
	for _, addr := range addrs {
		listener, err := net.ListenTCP("tcp", addr)
		if err != nil {
			return err
		}
		tcpListeners = append(tcpListeners, listener)
	}
	t.tcpListeners = tcpListeners
	return nil
}

func (t *Transport) run(messageChan chan<- message) {
	t.runTCP(messageChan)
	t.runUDP(messageChan)
}

func (t *Transport) runTCP(messageChan chan<- message) {
	for _, l := range t.tcpListeners {
		go func(listener *net.TCPListener) {
			for {
				conn, err := listener.AcceptTCP()
				if err != nil {
					return
				}

				if atomic.LoadInt32(&t.tcpConnCount) < maxConcurrentTCPConn {
					atomic.AddInt32(&t.tcpConnCount, 1)
					go t.handleTCPConn(conn, messageChan)
				} else {
					conn.Close()
				}
			}
		}(l)
	}
}

func (t *Transport) handleTCPConn(conn *net.TCPConn, messageChan chan<- message) {
	buf, err := g53util.TCPRead(conn)
	if err != nil {
		t.releaseConn(conn)
		return
	}

	messageChan <- message{
		usingTCP: true,
		addr:     conn.RemoteAddr(),
		destAddr: conn.LocalAddr(),
		conn:     conn,
		buf:      buf,
	}
}

func (t *Transport) releaseConn(conn *net.TCPConn) {
	conn.Close()
	atomic.AddInt32(&t.tcpConnCount, -1)
}

func (t *Transport) runUDP(messageChan chan<- message) {
	for _, conn := range t.udpConns {
		go func(conn_ *net.UDPConn) {
			for {
				buf := t.udpBufPool.Get()
				n, addr, err := conn_.ReadFromUDP(buf)
				if err == nil && n > 0 && n < maxQueryLen {
					select {
					case messageChan <- message{
						usingTCP: false,
						addr:     addr,
						destAddr: conn_.LocalAddr(),
						conn:     conn_,
						buf:      buf[0:n],
					}:
					default:
						logger.GetLogger().Warn("!!!udp buffer is full")
						t.udpBufPool.Put(buf[:maxQueryLen])
					}
				} else {
					t.udpBufPool.Put(buf[:maxQueryLen])
				}
			}
		}(conn)
	}
}

func (t *Transport) Close() {
	for _, conn := range t.udpConns {
		conn.Close()
	}

	for _, l := range t.tcpListeners {
		l.Close()
	}
}

func (t *Transport) SendResponse(q *message, response []byte) {
	if q.usingTCP {
		g53util.TCPWrite(response, q.conn.(*net.TCPConn))
		t.releaseConn(q.conn.(*net.TCPConn))
	} else {
		q.conn.(*net.UDPConn).WriteTo(response, q.addr)
	}
}

func (t *Transport) FinishQuery(q *message) {
	if q.usingTCP == false {
		t.udpBufPool.Put(q.buf[:maxQueryLen])
	}
}

func getAllIPs(isV4 bool) []net.IP {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		panic("get address failed:" + err.Error())
	}

	var ips []net.IP
	for _, addr := range addrs {
		var ip net.IP
		if isV4 {
			if ip = addr.(*net.IPNet).IP.To4(); ip == nil || ip.IsLoopback() {
				continue
			}
		} else {
			if ip = addr.(*net.IPNet).IP; ip.To4() != nil ||
				ip.IsGlobalUnicast() == false ||
				ip.IsLoopback() {
				continue
			}
		}
		ips = append(ips, ip)
	}

	return ips
}
