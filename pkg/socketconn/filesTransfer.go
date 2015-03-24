package socketconn

import (
    "fmt"
    "crypto/md5"
    "strconv"
    "hash"
    "bufio"
    "net" 
    "os" 
    "path/filepath" 
    "strings"   
    "io" 
    "errors"
    "path"
    "encoding/json"  
)


type FileHead struct {
    Ip    string
    Type  string
    Name  string
    Md5   string
}

var (
	ErrFileSend  = errors.New("ERROR - No file to send")
)

 
func Exists(name string) bool {
  if _, err := os.Stat(name); err != nil {
    if os.IsNotExist(err) {
      return false
    }
  }
  return true
}

func MD5(ph string) string {
  return Finger(md5.New(), ph)
}
 
func Finger(h hash.Hash, ph string) string {
 
  f, err := os.Open(ph)
  if err != nil {
    return ""
  }
  defer f.Close()
 
  io.Copy(h, bufio.NewReader(f))
 
  return fmt.Sprintf("%x", h.Sum(nil))
}

func IsSpace(c byte) bool {
  if c >= 0x00 && c <= 0x20 {
    return true
  }
  return false
}
 
func IsBlank(s string) bool {
  for i := 0; i < len(s); i++ {
    b := s[i]
    if !IsSpace(b) {
      return false
    }
  }
  return true
}

func ToInt(s string, dft int) int {
  var re, err = strconv.Atoi(s)
  if err != nil {
    return dft
  }
  return re
}

func Ph(ph string) string {
  if IsBlank(ph) {
    return ""
  }
  if strings.HasPrefix(ph, "~") {
    home := os.Getenv("HOME")
    if IsBlank(home) {
      panic(fmt.Sprintf("can not found HOME in envs, '%s' AbsPh Failed!", ph))
    }
    ph = fmt.Sprint(home, string(ph[1:]))
  }
  s, err := filepath.Abs(ph)
  if nil != err {
    panic(err)
  }
  return s
}

func FileR(ph string) *os.File {
  ph = Ph(ph)
  f, err := os.Open(ph)
  if nil != err {
    return nil
  }
  return f
}

//---------------------------------
func FcheckParents(aph string) {
  pph := path.Dir(aph)
  err := os.MkdirAll(pph, os.ModeDir|0755)
  if nil != err {
    panic(err)
  }
}
 
func FileW(ph string) *os.File {
  ph = Ph(ph)
 
  FcheckParents(ph)
 
  f, err := os.OpenFile(ph, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
  if nil != err {
    panic(err)
  }
  return f
}
//---------------------------------

func FileSend(conn net.Conn, types string, file string) error {
   
    if !Exists(file) {
    	return ErrFileSend
    }

    fh := make([]FileHead,1)
    x := strings.LastIndex(fmt.Sprintf("%s", conn.LocalAddr()), ":")
 
    fh[0] = FileHead{ Ip: fmt.Sprintf("%s", conn.LocalAddr())[:x], Type:types, Name:file, Md5:MD5(file) }
   
    b, _  := json.Marshal(fh)
    bb := string(b) + "\n"
    _, err := conn.Write([]byte(bb))
    if err != nil {
       return err
    }
 
    // [0] need the file to send
    // [1] no need the file to send
    line, _ := bufio.NewReader(conn).ReadString('\n')
    if ToInt(strings.TrimSpace(line), 1) == 0 {
        f := FileR(file)  
        if f != nil {         
            rd := bufio.NewReader(f) 
            wd := bufio.NewWriter(conn)
            _, err := io.Copy(wd, rd)
            if err != nil {
                return err
            }             
            wd.Flush() 
            f.Close()
        
        }
    }
 
    defer conn.Close()
 
    return nil
}
 
func FileReceive(conn net.Conn, rootPath string) (map[string]string, error) {
 
    var res map[string]string = make(map[string]string) 
    rd := bufio.NewReader(conn)
 
    for { 
        data, err := rd.ReadString('\n')
        if err == io.EOF {  
            break
        } else if err != nil {  
            return nil, err
        }
 
        fh := make([]FileHead,1)
        er := json.Unmarshal([]byte(data), &fh)
        if er != nil {
            return nil, err   
        }

        fmt.Println("Name: ", fh[0].Name)
        fmt.Println("Md5: ", fh[0].Md5)
    
        res["Ip"] = fh[0].Ip
        res["Type"] = fh[0].Type
        res["Name"] = fh[0].Name
        res["Md5"] = fh[0].Md5
 
 
        fname := rootPath + fh[0].Name
        fmd5 := fh[0].Md5
 
        if !Exists(fname) || MD5(fname) != fmd5 {
        //need the file
            conn.Write([]byte("0\n"))
            f := FileW(fname)
            wd := bufio.NewWriter(f)
            _, err := io.Copy(wd, rd)
            if err != nil {
                return nil, err
            }
            wd.Flush() 
            f.Close()
 
        } else {
        //no need the file
            conn.Write([]byte("1\n"))
            break
        }
 
        if Exists(fname) && MD5(fname) == fmd5 {
            break
        } else {
            return nil, fmt.Errorf("file MD5 fail")
        }
  
    }
 
    return res, nil
}  
//---------------------------------
 