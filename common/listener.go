package common

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/anacrolix/utp"
	"github.com/meshbird/meshbird/network/protocol"
	"net"
	"os"
)

type ListenerService struct {
	BaseService

	localNode *LocalNode
	socket    *utp.Socket

	logger *log.Logger
}

func (l ListenerService) Name() string {
	return "listener"
}

func (l *ListenerService) Init(ln *LocalNode) error {
	// TODO: Add prefix
	l.logger = log.New()
	log.SetOutput(os.Stderr)
	log.SetLevel(ln.config.Loglevel)

	l.logger.Info(fmt.Sprintf("Listening on port: %d", ln.State().ListenPort+1))
	socket, err := utp.NewSocket("udp4", fmt.Sprintf("0.0.0.0:%d", ln.State().ListenPort+1))
	if err != nil {
		return err
	}

	l.localNode = ln
	l.socket = socket
	return nil
}

func (l *ListenerService) Run() error {
	for {
		conn, err := l.socket.Accept()
		if err != nil {
			break
		}

		l.logger.Debug("Has new connection: %s", conn.RemoteAddr().String())

		if err = l.process(conn); err != nil {
			l.logger.Error("Error on process: %s", err)
		}
	}
	return nil
}

func (l *ListenerService) Stop() {
	l.SetStatus(StatusStopping)
	l.socket.Close()
}

func (l *ListenerService) process(c net.Conn) error {
	//defer c.Close()

	handshakeMsg, errHandshake := protocol.ReadDecodeHandshake(c)
	if errHandshake != nil {
		return errHandshake
	}

	l.logger.Debug("Processing hansdhake...")

	if !protocol.IsMagicValid(handshakeMsg.Bytes()) {
		return fmt.Errorf("Invalid magic bytes")
	}

	l.logger.Debug("Magic bytes are correct. Preparing reply...")

	if err := protocol.WriteEncodeOk(c); err != nil {
		return err
	}
	if err := protocol.WriteEncodePeerInfo(c, l.localNode.State().PrivateIP); err != nil {
		return err
	}

	peerInfo, errPeerInfo := protocol.ReadDecodePeerInfo(c)
	if errPeerInfo != nil {
		return errPeerInfo
	}

	l.logger.Debug("Processing PeerInfo...")

	rn := NewRemoteNode(c, handshakeMsg.SessionKey(), peerInfo.PrivateIP())

	l.logger.Debug("Adding remote node from listener...")

	l.localNode.NetTable().AddRemoteNode(rn)

	return nil
}
