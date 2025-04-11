package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
) 
type Request struct {
    Method          string
    Target          string
    Proto           string
    Header          map[string]string

    Body           []byte
}
type Response struct {
    Proto           string
    StatCode        int 
    StatText        string
    Header          map[string]string

    Body            []byte
}
type ResponseWriter interface{
    Write([]byte)(int, error)
    Read(p []byte)(int, error)
}
func main(){
    l, err := net.Listen("tcp", ":3490")
    if err != nil {
        log.Fatal(err) 
    }
    defer l.Close()
    fmt.Println("listening...")

    for {
        c, err := l.Accept()
        if err != nil {
            log.Println("unable to accept:", err)
            break
        }
        serve(c)
    }
}
func serve(c net.Conn){
    defer c.Close()
    buf := bufio.NewReader(c)
    
    req, err := readRequest(buf)
    if err != nil {
        log.Println(err)
    }

    serveHTTP(c, req)

}
func serveHTTP(w ResponseWriter, r *Request){
    
    switch r.Method{
        case "GET":
            var statCode int 
            var statText string

            if r.Header["Transfer-Encoding"] == "chunked"{
                resp := &Response{
                    Proto: "HTTP/1.1", StatCode: 200, StatText: "OK",
                    Header: map[string]string{
                        "Content-Type": "text/html",
                        "Date": time.Now().Format(time.RFC1123),
                        "Transfer-Encoding": "chunked",
                        "Connection": "keep-alive",
                    },
                }
                writeLineAndHeaders(w, resp)
                chunkedTransfer(r.Target, w)
            }

            file, err := getFile(r.Target)
            if err != nil {
                statCode = 404
                statText = "Not Found"
            }else {
                statCode = 200
                statText = "OK"
            }

            resp := &Response{
                Proto: "HTTP/1.1", StatCode: statCode, StatText: statText,
                Header: map[string]string{
                    "Content-Type": "text/html",
                    "Date": time.Now().Format(time.RFC1123),
                    "Content-Length": strconv.Itoa(len(file)),
                },
                Body: file,
            }
            writeLineAndHeaders(w, resp)
            w.Write(resp.Body)

        case "HEAD":
            file, err := getFile(r.Target)
            if err != nil {
                log.Print(err)
            }
            resp := &Response{
                Proto: "HTTP/1.1", StatCode: 200, StatText: "OK",
                Header: map[string]string{
                    "Content-Type": "text/html",
                    "Date": time.Now().Format(time.RFC1123),
                    "Content-Length": strconv.Itoa(len(file)),
                },
            }
            writeLineAndHeaders(w, resp)

        case "POST":
            resp := &Response{
                Proto: "HTTP/1.1",
                StatCode: 201,
                StatText: "Created",
                Header: map[string]string{
                    "Content-Type": r.Header["Content-Type"],
                    "Date": time.Now().Format(time.RFC1123),
                    "Content-Length": r.Header["Content-Length"],
                },
                Body: r.Body,
            }
            writeLineAndHeaders(w, resp)
            w.Write(resp.Body)
        }

}
func chunkedTransfer(path string, w ResponseWriter){
    path = fmt.Sprintf("./resources%s", path)
    file, err := os.Open(path)
    if err != nil {
        log.Println(err)
    }
    defer file.Close()


    buf := make([]byte, 30)
    for {
        n, err := file.Read(buf)
        if err != nil {
            break
        }
        chunkSize := fmt.Sprintf("%x\r\n", n)
        w.Write([]byte(chunkSize))
        w.Write(buf[:n])
        w.Write([]byte("\r\n"))
        time.Sleep(2 * time.Second)
    }
    finalLine := fmt.Sprintf("%x\r\n\r\n", 0)
    w.Write([]byte(finalLine))
}
func getFile(path string)([]byte, error){
    path = fmt.Sprintf("./resources/%s", path)
    file, err := os.ReadFile(path)
    if err != nil {
        notFoundFile, _ := os.ReadFile("./resources/404.html")
        return notFoundFile, err
    }
    return file, nil
}
func writeLineAndHeaders(w ResponseWriter, resp *Response){
    startLine := fmt.Sprintf("%s %d %s\n", resp.Proto, resp.StatCode, resp.StatText)
    w.Write([]byte(startLine))

    for key, val := range resp.Header{
        h := fmt.Sprintf("%s: %s\n", key, val)
        w.Write([]byte(h))
    }
    w.Write([]byte("\r\n"))
}
func readRequest(buf *bufio.Reader) (r *Request, error error){
    // PARSE Request Line
    line0, err := buf.ReadString('\n')
    if err != nil {
        return nil, err
    }
    startLine := strings.Split(line0, " ")

    // PARSE Headers
    var headers = make(map[string]string)
    
    for {
        h, err := buf.ReadString('\n')
        if err != nil {
            return nil, err 
        }
        // if an empty line appears we are now in the body
        if h == "\r\n"{
            break
        }
        
        header := strings.Split(h, ":")
        headers[header[0]] = strings.TrimSpace(header[1])
    }
    
    req := &Request{
        Method: startLine[0],
        Target: startLine[1],
        Proto: startLine[2],
        Header: headers,
    }

    // PARSE Body
    switch req.Method {
        case "GET": 
            break
        case "POST":
            length, err := strconv.Atoi(req.Header["Content-Length"])
            if err != nil {
                return nil, err
            }
            data, err := buf.Peek(length)
            if err != nil {
                return nil, err
            }
            req.Body = data
        case "HEAD": 
            break
    }
    return req, nil
}
