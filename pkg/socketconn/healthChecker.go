package socketconn
 
import (
    "time"
    "fmt"
    "net"
)
 
type HealthReporter interface {
    HealthReport(clientName string)
}
 
type healthClientInfo struct {
    time int64
    ip net.IP
} 

type HealthCheck struct {
    ClientMap map[string]healthClientInfo
}

func (h *HealthCheck) HealthCheckRun(hr HealthReporter, setOvertime int64){
    
    var timeNow int64
    for{
        time.Sleep(1000 * time.Millisecond)
        timeNow = time.Now().Unix()
        for key, value := range h.ClientMap {
            diff := timeNow - value.time
            if diff > setOvertime{
                hr.HealthReport(key)
            }            
                
        }

        
    }
}
 
func (h *HealthCheck) HealthCheckAdd(clientName string, clientIP net.IP){
    
    if(h.ClientMap==nil){
        h.ClientMap = make(map[string]healthClientInfo)
        fmt.Println("Create ClientMap")
    }
    
    var hc healthClientInfo
    hc.time = time.Now().Unix()
    hc.ip = clientIP
    h.ClientMap[clientName] = hc
 
}

func (h *HealthCheck) HealthCheckDel(clientName string){
 
    delete(h.ClientMap, clientName)
}