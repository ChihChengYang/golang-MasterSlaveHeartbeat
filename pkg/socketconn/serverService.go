package socketconn

import (
    "fmt" 
    "net"
    "sync"
    "time"
    "strings"
)

//----------------------------
type HealthChecker interface {
    HealthCheckRun(hr HealthReporter, setOvertime int64)
    HealthCheckAdd(clientName string, clientIP net.IP)
    HealthCheckDel(clientName string)
}
 

type Heartbeater interface {
    HeartbeatBreak()
    HeartbeatStart()
}
//---------------------------- 


type Callback struct{}

type Server struct {
    basic *basicSrv
    hc HealthChecker
}

type Config struct {
    ServiceType   SERVICETYPE
    AcceptTimeout          time.Duration // connection accepted timeout
    ReadTimeout            time.Duration // connection read timeout
    WriteTimeout           time.Duration // connection write timeout
    PacketSizeLimit        uint32        // the limit of packet size
    PacketSendChanLimit    uint32        // the limit of packet send channel
    PacketReceiveChanLimit uint32        // the limit of packet receive channel

    Hb Heartbeater
}

type basicSrv struct {
    config    *Config         // server configuration
    callback  ConnCallback    // message callbacks in connection
  
    exitChan  chan struct{}   // notify all goroutines to shutdown
    waitGroup *sync.WaitGroup // wait for all goroutines
}

func (this *Callback) OnConnect(c *ConnecterConfig) bool {
 
    addr := c.GetRawConn().RemoteAddr()
    c.PutExtraData(addr)
    fmt.Println("OnConnect:", addr)

    return true
}

func (this *Callback) OnMessage(c *ConnecterConfig, p Packet) bool {
 
    packPacket := p.(*PackagePacket)
 
    switch REQUESTTYPE(packPacket.GetPackagePacketType()) { 

        case REQUEST_TYPE_GETSESSION:  
            
            fmt.Printf("OnMessage:[%v] [%v] [%v]\n", packPacket.GetPackagePacketLength(),
                packPacket.GetPackagePacketType(), string(packPacket.GetPackagePacketBody()))
        
            uuid, err := newUUID()
            if err != nil {
                fmt.Printf("error: %v\n", err)
                return false
            }
        
            fmt.Println("OnConnect:",  c.GetExtraData(),uuid)

            c.AsyncWritePacket(NewPackagePacket([]byte(uuid), REQUEST_TYPE_GETSESSION, false), time.Second)
    
        case REQUEST_TYPE_CHECKSTATUS:

        default:    
            fmt.Println("ERROR: RequestType")
            return false
    }    

    return true
}

func (this *Callback) OnClose(c *ConnecterConfig) {
    fmt.Println("OnClose:", c.GetExtraData())
}

 
func newBasicSrv(config *Config, callback ConnCallback) *basicSrv {
    return &basicSrv{
        config:    config,
        callback:  callback,
        
        exitChan:  make(chan struct{}),
        waitGroup: &sync.WaitGroup{},
    }
}

// NewServer creates a new Server
func NewServer(config *Config, callback ConnCallback) *Server {
    basic := newBasicSrv(config, callback)
    return &Server{basic,nil}
}

// Start starts server
func (s *Server) StartTCP(listener *net.TCPListener) {
    s.basic.waitGroup.Add(1)
    defer func() {
        listener.Close()
        s.basic.waitGroup.Done()
    }()

    for {
        select {
        case <-s.basic.exitChan:
            return

        default:
        }

        listener.SetDeadline(time.Now().Add(s.basic.config.AcceptTimeout))

        conn, err := listener.AcceptTCP()
        if err != nil {
            continue
        }

        go newTCPConn(conn, s.basic).Run()
    }
}
 
func (s *Server) HealthReport(clientName string) {
    
    s.hc.HealthCheckDel(clientName)
  
    s.basic.config.Hb.HeartbeatBreak()
 
}

func (s *Server) StartUDP(listener *net.UDPConn) {
 
    buf := make([]byte, 1024)
    
    for {
        select {
        case <-s.basic.exitChan:
            return

        default:
        }
 
        n,addr,err := listener.ReadFromUDP(buf)
       
        fmt.Println("Received ",string(buf[0:n]), " from ",addr)
 
        if err != nil {
            fmt.Println("Error: ",err)
        } 
        
        s.hc.HealthCheckAdd(string(buf[0:n]), addr.IP  ) 
         
    }
  
}

func  StartTCP_File_R(listener *net.TCPListener) {
 
    defer func() {
        listener.Close()
    }()

    for {
  
        conn, err := listener.AcceptTCP()
        if err != nil {
            continue
        }
    
        res, err := FileReceive(conn,"./")
        if err != nil {
            fmt.Println(err)
        } else {
            fmt.Println("Name f-> ",res["Name"])
        }
    }
}

func  StartTCP_File_S(listener *net.TCPListener) {
 
    defer func() {
        listener.Close()
    }()

    for {
  
        conn, err := listener.AcceptTCP()
        if err != nil {
            continue
        }
  
        FileSend( conn, "sendFile",  "/home/jeff/TestProCom.c")
         
    }
}

func (s *Server) StartTCP_Heartbeat(listener *net.TCPListener) {
// for Slave
    defer func() {
        listener.Close()
    }()

    for {
 
        conn, err := listener.AcceptTCP()
        if err != nil {
            continue
        }
 
        reply := make([]byte, 4)
        conn.Read(reply)
 
        _, err2 := conn.Write([]byte("1"))
        if err2 != nil {
           continue
        }
 
        if strings.Contains(string(reply), "0") == true {
            s.basic.config.Hb.HeartbeatStart()
        }
 
    }
}



func ServerService(config *Config) {
 
    switch config.ServiceType { 
    
    case SERVICE_TYPE_TCP:
        tcpAddr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:8080")
        checkError(err)
        listener, err := net.ListenTCP("tcp", tcpAddr)
        checkError(err)

        srv := NewServer(config, &Callback{})

        // start server
        go srv.StartTCP(listener)
        fmt.Println("listening:", listener.Addr())
    
    case SERVICE_TYPE_TCP_RECEIVE_FILE:
 
        tcpAddr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:8084")
        checkError(err)
        listener, err := net.ListenTCP("tcp", tcpAddr)
        checkError(err)
         // start server
        go StartTCP_File_R(listener)
        fmt.Println("listening:", listener.Addr())
    
    case SERVICE_TYPE_TCP_SEND_FILE:

        tcpAddr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:8086")
        checkError(err)
        listener, err := net.ListenTCP("tcp", tcpAddr)
        checkError(err)
         // start server
        go StartTCP_File_S(listener)
        fmt.Println("listening:", listener.Addr())

    case SERVICE_TYPE_TCP_HEARTBEAT:

        tcpAddr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:8088")
        checkError(err)
        listener, err := net.ListenTCP("tcp", tcpAddr)
        checkError(err)

        srv := NewServer(config, &Callback{})
 
        go srv.StartTCP_Heartbeat(listener)

        fmt.Println("listening:", listener.Addr())

    case SERVICE_TYPE_UDP_HEARTBEAT:
    
        udpAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:8082")
        checkError(err)
        listener, err := net.ListenUDP("udp", udpAddr)
        checkError(err)

        srv := NewServer(config, &Callback{})

        go srv.StartUDP(listener)
        fmt.Println("listening:", listener.LocalAddr())

//-------------------------------  
        var hc HealthCheck        
        srv.hc = &hc          
        go srv.hc.HealthCheckRun(srv, 5)    
//-------------------------------
    
    default:    
        fmt.Println("ERROR: ServiceType")
    
    }



}

