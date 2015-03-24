package main

import (
	"socketconn"
	"backuper"
    "fmt"
    "time"
)

type SlaveInfo struct {
    Name  string 
}

type mainMaster struct {
    errorChannelIP  chan string
    slaveTable map[string]SlaveInfo 
    heartbeatTable map[string]SlaveInfo
}

var gMM mainMaster

func (mm *mainMaster) HeartbeatErrorHandle(){

    for {
        select {
            case  ip := <-mm.errorChannelIP:
                fmt.Println("HeartbeatErrorHandle: ",ip)
                //try connecting again, Slave no return
                var s SlaveInfo
                s.Name = "UUID" 
                mm.heartbeatTable[ip] = s
            default:
        }

        time.Sleep(time.Second)
    }
    
}

func (mm *mainMaster) HeartbeatBreak(){
//----------------------
//----------------------
	var b backuper.Backup
    b.BackupRun()
//----------------------    
}

func (mm *mainMaster) HeartbeatStart() {
//TCP     
//connect to slave from the slave table
//
    configC := &socketconn.ClientConfig{
        RequestType:    socketconn.REQUEST_TYPE_HEARTBEAT_START,
    }

    for {
        for k,_ := range mm.heartbeatTable {
  
            configC.SlaveIP = k //"127.0.0.1:8088"
            err := configC.ClientServiceStart()
            if err == nil {
                //require Slave to start heartbeat
                configC.ClientServiceHeartbeatStart(mm.errorChannelIP) 
                //suppose Slave should be connected, remove Slave's ip from table
                delete(mm.heartbeatTable, k)
            }else{
                //try connecting again, Slave not ready yet
                fmt.Println("HeartbeatStart Connecting!!!!!!!! ",k)
            }
        }   
        time.Sleep(time.Second)
    }


    
}

func (mm *mainMaster) SlaveTableAdd(ip string , name string){
 
   var s SlaveInfo
   s.Name = name  
   mm.slaveTable[ip] = s
   mm.heartbeatTable[ip] = s
 
}

func (mm *mainMaster) SlaveTableDel(ip string){
 
    delete(mm.slaveTable, ip)
    delete(mm.heartbeatTable, ip)
 
}

func main() {
	
    gMM.errorChannelIP = make(chan string)
    gMM.slaveTable = make(map[string]SlaveInfo) 
    gMM.heartbeatTable = make(map[string]SlaveInfo)
//-------------------------------
	// initialize server params
    config := &socketconn.Config{
    	ServiceType:    socketconn.SERVICE_TYPE_TCP,
        AcceptTimeout:          50 * time.Second,
        ReadTimeout:            240 * time.Second,
        WriteTimeout:           240 * time.Second,
        PacketSizeLimit:        2048,
        PacketSendChanLimit:    20,
        PacketReceiveChanLimit: 20,

        Hb: &gMM,
    }

    socketconn.ServerService(config)
//-------------------------------
   
    gMM.SlaveTableAdd("127.0.0.1:8088" , "UUID")   
    go gMM.HeartbeatErrorHandle()
    go gMM.HeartbeatStart()
//-------------------------------

    var input string
    fmt.Scanln(&input)
}