package main
 
import (
   "fmt"
   "socketconn"
   "time"

)

type mainSlave struct {
    sessionString string
}

func (ms *mainSlave) HeartbeatBreak(){
    ms.sessionString = ""
}

func (ms *mainSlave) HeartbeatStart() {
//------------------------------- 
//TCP        
    config := &socketconn.ClientConfig{}
    
    if len( ms.sessionString) < 2{

        config.RequestType = socketconn.REQUEST_TYPE_GETSESSION         

        config.ClientServiceStart()
        ms.sessionString = config.ClientServiceGetSession() 
  
        fmt.Println("sessionString",ms.sessionString)
    }


//-------------------------------
//UDP    
    config.RequestType = socketconn.REQUEST_TYPE_HEARTBEAT
    config.ClientServiceStart()
    config.ClientServiceHeartbeat(ms.sessionString) 
 
}

func main() {

    var input string
 
    var hs mainSlave

    config := &socketconn.Config{
        ServiceType:    socketconn.SERVICE_TYPE_TCP_HEARTBEAT,
        AcceptTimeout:          50 * time.Second,
        ReadTimeout:            240 * time.Second,
        WriteTimeout:           240 * time.Second,
        PacketSizeLimit:        2048,
        PacketSendChanLimit:    20,
        PacketReceiveChanLimit: 20,

        Hb: &hs,
    }

    socketconn.ServerService(config)
 
 
    fmt.Scanln(&input)
 
}

