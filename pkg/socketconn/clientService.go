package socketconn

import (
    "fmt" 
    "net"
    "time"
    "strings"
   // "strconv"
)

type ClientConfig struct {
    RequestType   REQUESTTYPE
    connTCP       *net.TCPConn  
    connUDP       *net.UDPConn 
    SlaveIP string
}

 
func ( clientConfig *ClientConfig) ClientServiceGetSession() string {
    
    var sessionString string
    sessionString=""
    
    if clientConfig.RequestType == REQUEST_TYPE_GETSESSION{
        cc := &ConnecterConfig{}    
        clientConfig.connTCP.Write(NewPackagePacket([]byte("session"),REQUEST_TYPE_GETSESSION, false).Serialize())
        p, err := cc.ReadPackagePacket(clientConfig.connTCP, 1024)
        if err == nil {
            packPacket := p.(*PackagePacket)
            fmt.Printf("Server reply:[%v] [%v]\n", packPacket.GetPackagePacketLength(), string(packPacket.GetPackagePacketBody()))
            sessionString = string(packPacket.GetPackagePacketBody())
        }
   
    }

   defer clientConfig.ClientServiceClose()

   return sessionString
} 

func ( clientConfig *ClientConfig) csHeartbeat(mark string){
   
    //i := 0
    for {
        // msg := strconv.Itoa(i)
       // i++
     
        buf := []byte(mark)
        _,err := clientConfig.connUDP.Write(buf)
        if err != nil {
            fmt.Println(mark, err)
        }
  

        time.Sleep(time.Second * 1)
    }

    time.Sleep(5 * time.Second)
 
    defer clientConfig.ClientServiceClose()
}

func ( clientConfig *ClientConfig) ClientServiceHeartbeat(mark string) {
    if clientConfig.RequestType == REQUEST_TYPE_HEARTBEAT{
        go clientConfig.csHeartbeat(mark)
    }

} 

func ( clientConfig *ClientConfig) ClientServiceHeartbeatStart(errorChannelIP chan string) {

    if clientConfig.RequestType == REQUEST_TYPE_HEARTBEAT_START{
 
        clientConfig.connTCP.Write([]byte("0"))

        reply := make([]byte, 4)
        clientConfig.connTCP.Read(reply)
 
        if strings.Contains(string(reply), "1") == true {
            fmt.Println("ClientServiceHeartbeatStart OK")
        }else{     
        //no feedback        
            errorChannelIP <- clientConfig.SlaveIP
        }
  
    }

    defer clientConfig.ClientServiceClose()

}

func ( clientConfig *ClientConfig) ClientServiceCheckStatus() {
    if clientConfig.RequestType == REQUEST_TYPE_CHECKSTATUS{
        
    }
} 

func ( clientConfig *ClientConfig) ClientServiceSendFile(file string) {
 
    if clientConfig.RequestType == REQUEST_TYPE_SENDFILE{
         go FileSend( clientConfig.connTCP, "sendFile",  file)
    }
     
}
 
func ( clientConfig *ClientConfig) ClientServiceReceiveFile() {
   
    if clientConfig.RequestType == REQUEST_TYPE_RECEIVEFILE{
  
        res, err := FileReceive(clientConfig.connTCP,"./")
        if err != nil {
            fmt.Println(err)
        } else {
            fmt.Println("Name f-> ",res["Name"])
        }
    
    }
}

 
func ( clientConfig *ClientConfig) ClientServiceClose(){
    switch clientConfig.RequestType { 
    case REQUEST_TYPE_HEARTBEAT_START,REQUEST_TYPE_CHECKSTATUS,REQUEST_TYPE_SENDFILE,REQUEST_TYPE_GETSESSION:  
        clientConfig.connTCP.Close()
    case REQUEST_TYPE_HEARTBEAT:    
        clientConfig.connUDP.Close()
    default:    
        fmt.Println("ERROR: RequestType")
    }    
}

func ( clientConfig *ClientConfig) ClientServiceStart() error{

    var err error
    
    switch clientConfig.RequestType { 
    case REQUEST_TYPE_CHECKSTATUS,REQUEST_TYPE_GETSESSION:    
    // connect to server
        tcpAddr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:8080")
        checkError(err)
        conn, err := net.DialTCP("tcp", nil, tcpAddr)
        checkError(err)

        clientConfig.connTCP = conn
        fmt.Printf("Server reply:")
  

    case REQUEST_TYPE_HEARTBEAT:
     

        udpAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:8082")
        checkError(err)
        conn, err := net.DialUDP("udp", nil, udpAddr)
        checkError(err)
 
        clientConfig.connUDP = conn

    case REQUEST_TYPE_HEARTBEAT_START:    
 
        tcpAddr, err := net.ResolveTCPAddr("tcp4", clientConfig.SlaveIP)
        if err != nil {
           return err
        }
        conn, err := net.DialTCP("tcp", nil, tcpAddr)
        if err != nil {
           return err
        }


        clientConfig.connTCP = conn    
        
    case REQUEST_TYPE_SENDFILE:    
    // connect to server
        tcpAddr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:8084")
        checkError(err)
        conn, err := net.DialTCP("tcp", nil, tcpAddr)
        checkError(err)

        clientConfig.connTCP = conn

 
    case REQUEST_TYPE_RECEIVEFILE:    
    // connect to server
        tcpAddr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:8086")
        checkError(err)
        conn, err := net.DialTCP("tcp", nil, tcpAddr)
        checkError(err)

        clientConfig.connTCP = conn

    default:    
        fmt.Println("ERROR: ServiceType")
    
    }


    return err

} 


