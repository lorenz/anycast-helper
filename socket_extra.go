package main

import (
	"encoding/binary"
	"errors"
	"net"

	"github.com/vishvananda/netlink"

	"github.com/vishvananda/netlink/nl"
	"golang.org/x/sys/unix"
)

const (
	sizeofSocketID      = 0x30
	sizeofSocketRequest = sizeofSocketID + 0x8
	sizeofSocket        = sizeofSocketID + 0x18
)

const (
	TCP_ESTABLISHED = iota + 1
	TCP_SYN_SENT
	TCP_SYN_RECV
	TCP_FIN_WAIT1
	TCP_FIN_WAIT2
	TCP_TIME_WAIT
	TCP_CLOSE
	TCP_CLOSE_WAIT
	TCP_LAST_ACK
	TCP_LISTEN
	TCP_CLOSING /* Now a valid state */
	TCP_NEW_SYN_RECV

	TCP_MAX_STATES /* Leave at the end! */

)

var (
	native       = nl.NativeEndian()
	networkOrder = binary.BigEndian
)

type socketRequest struct {
	Family   uint8
	Protocol uint8
	Ext      uint8
	pad      uint8
	States   uint32
	ID       netlink.SocketID
}

type writeBuffer struct {
	Bytes []byte
	pos   int
}

func (b *writeBuffer) Write(c byte) {
	b.Bytes[b.pos] = c
	b.pos++
}

func (b *writeBuffer) Next(n int) []byte {
	s := b.Bytes[b.pos : b.pos+n]
	b.pos += n
	return s
}

func (r *socketRequest) Serialize() []byte {
	b := writeBuffer{Bytes: make([]byte, sizeofSocketRequest)}
	b.Write(r.Family)
	b.Write(r.Protocol)
	b.Write(r.Ext)
	b.Write(r.pad)
	native.PutUint32(b.Next(4), r.States)
	networkOrder.PutUint16(b.Next(2), r.ID.SourcePort)
	networkOrder.PutUint16(b.Next(2), r.ID.DestinationPort)
	copy(b.Next(4), r.ID.Source.To4())
	b.Next(12)
	copy(b.Next(4), r.ID.Destination.To4())
	b.Next(12)
	native.PutUint32(b.Next(4), r.ID.Interface)
	native.PutUint32(b.Next(4), r.ID.Cookie[0])
	native.PutUint32(b.Next(4), r.ID.Cookie[1])
	return b.Bytes
}

func (r *socketRequest) Len() int { return sizeofSocketRequest }

// GetListenersOnPort returns the Socket identified by its local and remote addresses.
func HasListenersOnPort(port uint16) (bool, error) {
	s, err := nl.Subscribe(unix.NETLINK_INET_DIAG)
	if err != nil {
		return false, err
	}
	defer s.Close()
	req := nl.NewNetlinkRequest(nl.SOCK_DIAG_BY_FAMILY, unix.NLM_F_DUMP)
	req.AddData(&socketRequest{
		Family:   unix.AF_INET,
		Protocol: unix.IPPROTO_TCP,
		States:   1 << TCP_LISTEN,
		ID: netlink.SocketID{
			SourcePort:      port,
			DestinationPort: 0,
			Source:          net.IP{0, 0, 0, 0},
			Destination:     net.IP{0, 0, 0, 0},
			Cookie:          [2]uint32{0, 0},
		},
	})
	s.Send(req)
	msgs, err := s.Receive()
	if err != nil {
		return false, err
	}
	if len(msgs) == 0 {
		return false, errors.New("no message nor error from netlink")
	}
	if len(msgs) > 2 {
		return true, nil
	}
	return len(msgs[0].Data) >= sizeofSocket, nil
}

func HasListenersOnPortSimple(port uint16) bool {
	hasListeners, err := HasListenersOnPort(port)
	if err != nil {
		return false
	}
	return hasListeners
}
