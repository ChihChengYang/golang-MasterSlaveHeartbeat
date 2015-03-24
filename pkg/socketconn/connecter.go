package socketconn
 

import (
    "errors"
    "net"
    "sync"
    "sync/atomic"
    "time"
    "io"
    "log"
   
    "fmt"
    "crypto/rand"
    //"io"
)

type SERVICETYPE uint16
const (
    SERVICE_TYPE_TCP SERVICETYPE = 0
    SERVICE_TYPE_UDP_HEARTBEAT SERVICETYPE = 1
    SERVICE_TYPE_TCP_RECEIVE_FILE SERVICETYPE = 2
    SERVICE_TYPE_TCP_SEND_FILE SERVICETYPE = 3
    SERVICE_TYPE_TCP_HEARTBEAT SERVICETYPE = 4
)

type REQUESTTYPE uint16
const (
    REQUEST_TYPE_CHECKSTATUS REQUESTTYPE = 0 //SERVICE_TYPE_TCP
    REQUEST_TYPE_HEARTBEAT REQUESTTYPE = 1 //SERVICE_TYPE_UDP
    REQUEST_TYPE_SENDFILE REQUESTTYPE = 2 //SERVICE_TYPE_TCP
    REQUEST_TYPE_RECEIVEFILE REQUESTTYPE = 3 //SERVICE_TYPE_TCP
    REQUEST_TYPE_GETSESSION REQUESTTYPE = 4 //SERVICE_TYPE_TCP
    REQUEST_TYPE_HEARTBEAT_START REQUESTTYPE = 5 //SERVICE_TYPE_TCP

)
 
type PackagePacket struct {
    buffer []byte
}

type Packet interface {
    Serialize() []byte
}

type ConnecterConfig struct {
    basic             *basicSrv
    conn              *net.TCPConn // the raw connection
    extraData         interface{}  // save the extra data
    closeOnce         sync.Once    // close the conn, once, per instance
    closeFlag         int32
    closeChan         chan struct{}
    packetSendChan    chan Packet // packet send queue
    packetReceiveChan chan Packet // packeet receive queue
}

// ConnCallback is an interface of methods that are used as callbacks on a connection
type ConnCallback interface {
    // OnConnect is called when the connection was accepted,
    // If the return value of false is closed
    OnConnect(*ConnecterConfig) bool

    // OnMessage is called when the connection receives a packet,
    // If the return value of false is closed
    OnMessage(*ConnecterConfig, Packet) bool

    // OnClose is called when the connection closed
    OnClose(*ConnecterConfig)
} 

// Error type
var (
    ErrConnClosing   = errors.New("use of closed network connection")
    ErrWriteBlocking = errors.New("write packet was blocking")
    ErrReadBlocking  = errors.New("read packet was blocking")
)

func checkError(err error) {
    if err != nil {
        log.Fatal(err)
    }
} 

func newTCPConn(conn *net.TCPConn, basic *basicSrv) *ConnecterConfig {
    return &ConnecterConfig{
        basic:             basic,
        conn:              conn,
        closeChan:         make(chan struct{}),
        packetSendChan:    make(chan Packet, basic.config.PacketSendChanLimit),
        packetReceiveChan: make(chan Packet, basic.config.PacketReceiveChanLimit),
    }
}

// GetExtraData gets the extra data from the Conn
func (c *ConnecterConfig) GetExtraData() interface{} {
    return c.extraData
}

// PutExtraData puts the extra data with the Conn
func (c *ConnecterConfig) PutExtraData(data interface{}) {
    c.extraData = data
}

// GetRawConn returns the raw net.TCPConn from the Conn
func (c *ConnecterConfig) GetRawConn() *net.TCPConn {
    return c.conn
}

// Do it
func (c *ConnecterConfig) Run() {
    if !c.basic.callback.OnConnect(c) {
        return
    }

    c.basic.waitGroup.Add(3)
    go c.handleLoop()
    go c.readLoop()
    go c.writeLoop()
}

// Close closes the connection
func (c *ConnecterConfig) Close() {
    c.closeOnce.Do(func() {
        atomic.StoreInt32(&c.closeFlag, 1)
        close(c.closeChan)
        c.conn.Close()
        c.basic.callback.OnClose(c)
    })
}

// IsClosed indicates whether or not the connection is closed
func (c *ConnecterConfig) IsClosed() bool {
    return atomic.LoadInt32(&c.closeFlag) == 1
}


// AsyncWritePacket async writes a packet, this method will never block
func (c *ConnecterConfig) AsyncWritePacket(p Packet, timeout time.Duration) error {
    if c.IsClosed() {
        return ErrConnClosing
    }

    if timeout == 0 {
        select {
        case c.packetSendChan <- p:
            return nil

        default:
            return ErrWriteBlocking
        }

    } else {
        select {
        case c.packetSendChan <- p:
            return nil

        case <-c.closeChan:
            return ErrConnClosing

        case <-time.After(timeout):
            return ErrWriteBlocking
        }
    }
}

func (c *ConnecterConfig) readLoop() {
    defer func() {
        recover()
        c.Close()
        c.basic.waitGroup.Done()
    }()

    for {
        select {
        case <-c.basic.exitChan:
            return

        case <-c.closeChan:
            return

        default:
        }

        c.conn.SetReadDeadline(time.Now().Add(c.basic.config.ReadTimeout))
        recPacket, err := c.ReadPackagePacket(c.conn, c.basic.config.PacketSizeLimit)
        if err != nil {
            return
        }

        c.packetReceiveChan <- recPacket
    }
}

func (c *ConnecterConfig) writeLoop() {
    defer func() {
        recover()
        c.Close()
        c.basic.waitGroup.Done()
    }()

    for {
        select {
        case <-c.basic.exitChan:
            return

        case <-c.closeChan:
            return

        case p := <-c.packetSendChan:
            c.conn.SetWriteDeadline(time.Now().Add(c.basic.config.WriteTimeout))
            if _, err := c.conn.Write(p.Serialize()); err != nil {
                return
            }
        }
    }
}

func (c *ConnecterConfig) handleLoop() {
    defer func() {
        recover()
        c.Close()
        c.basic.waitGroup.Done()
    }()

    for {
        select {
        case <-c.basic.exitChan:
            return

        case <-c.closeChan:
            return

        case p := <-c.packetReceiveChan:
            if !c.basic.callback.OnMessage(c, p) {
                return
            }
        }
    }
}


func (this *PackagePacket) Serialize() []byte {
    return this.buffer
}

func (this *PackagePacket) GetPackagePacketLength() uint32 {
    return BytesToUint32(this.buffer[0:4])
}

func (this *PackagePacket) GetPackagePacketType() uint16 {
    return BytesToUint16(this.buffer[4:6])
}


func (this *PackagePacket) GetPackagePacketBody() []byte {
    return this.buffer[6:]
}

/*每個數據包的組成： 包長 + 類型 + 數據
–> 4字節 + 2字節 + 數據
–> uint32 + uint16 + []byte
*/
func NewPackagePacket(buffer []byte, req REQUESTTYPE,  lenFieldFlag bool) *PackagePacket {
    pac := &PackagePacket{}

    if lenFieldFlag {
        pac.buffer = buffer

    } else {
        pac.buffer = make([]byte, 6+len(buffer))
        copy(pac.buffer[0:4], Uint32ToBytes(uint32(len(buffer))))
        copy(pac.buffer[4:6], Uint16ToBytes(uint16(req)))
        copy(pac.buffer[6:], buffer)
    }

    return pac
}

func (c *ConnecterConfig) ReadPackagePacket(r io.Reader, packetSizeLimit uint32) (Packet, error) {
    var (
        lengthBytes []byte = make([]byte, 4)
        typeBytes []byte = make([]byte, 2)
        length      uint32
        packetType      uint16
    )

    // read length
    if _, err := io.ReadFull(r, lengthBytes); err != nil {
        return nil, ErrReadPacket
    }
    
    if _, err := io.ReadFull(r, typeBytes); err != nil {
        return nil, ErrReadPacket
    }

    if length = BytesToUint32(lengthBytes); length > packetSizeLimit {
        return nil, ErrPacketTooLarger
    }

    packetType  = BytesToUint16(typeBytes)

 
    // read body ( buffer = lengthBytes + body )
    buffer := make([]byte, 6+length)
    copy(buffer[0:4], lengthBytes)
    copy(buffer[4:6], typeBytes)
    
    if _, err := io.ReadFull(r, buffer[6:]); err != nil {
        return nil, ErrReadPacket
    }


    return NewPackagePacket(buffer, REQUESTTYPE(packetType), true), nil
}


// newUUID generates a random UUID according to RFC 4122
func newUUID() (string, error) {
    uuid := make([]byte, 16)
    n, err := io.ReadFull(rand.Reader, uuid)
    if n != len(uuid) || err != nil {
        return "", err
    }
    // variant bits; see section 4.1.1
    uuid[8] = uuid[8]&^0xc0 | 0x80
    // version 4 (pseudo-random); see section 4.1.3
    uuid[6] = uuid[6]&^0xf0 | 0x40
    return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}