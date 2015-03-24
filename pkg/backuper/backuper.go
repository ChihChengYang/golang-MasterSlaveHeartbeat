package backuper

import (
    "fmt"
    "log"
 	"net/mail"
    "encoding/base64"
    "net/smtp"
    "strings" 
)
 


type Backup struct {
 
}

func (b *Backup) BackupRun(){
    
    go sendMail()
    fmt.Println("BackupRun:")

}
  

func encodeRFC2047(String string) string{
    // use mail's rfc2047 to encode any string
    addr := mail.Address{String, ""}
    return strings.Trim(addr.String(), " <>")
} 

func sendMail(){
	// Set up authentication information.
    smtpServer := "smtp-mail.outlook.com"
    auth := smtp.PlainAuth(
            "",
            "govasystem@outlook.com",
            "go_123456",
            smtpServer,
        )
 
    from := mail.Address{"VA System", "govasystem@outlook.com"}
    to := mail.Address{"Sir", "govasystem@outlook.com"}
    title := "System Error!!!!"
 
    body := "Slave IP :" + "127.0.0.8";
 
    header := make(map[string]string)
    header["From"] = from.String()
    header["To"] = to.String()
    header["Subject"] = encodeRFC2047(title)
    header["MIME-Version"] = "1.0"
    header["Content-Type"] = "text/plain; charset=\"utf-8\""
    header["Content-Transfer-Encoding"] = "base64"
 
    message := ""
    for k, v := range header {
        message += fmt.Sprintf("%s: %s\r\n", k, v)
    }
    message += "\r\n" + base64.StdEncoding.EncodeToString([]byte(body))
 
    // Connect to the server, authenticate, set the sender and recipient,
    // and send the email all in one step.
    err := smtp.SendMail(
        smtpServer + ":587",
        auth,
        from.Address,
        []string{to.Address},
        []byte(message),
        //[]byte("This is the email body."),
        )
    if err != nil {
        log.Fatal(err)
    } 
}
 