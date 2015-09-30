package main

import (
    "fmt"
    "flag"
    "os"
    "net/http"
    "net/url"
    "io"
    "golang.org/x/net/html"
    "strings"
    "path/filepath"
    //"time"
)

var start_url *url.URL
var visited = make(map[string]bool)
var goCount = 0

func createPaths (parsed_url *url.URL) *os.File{
  var dir,file string
  i := strings.LastIndex(parsed_url.Path, "/")
  if(i >=0 && strings.Index(parsed_url.Path[i:], ".") >= 0){
   dir, file = filepath.Split(parsed_url.Path)
  } else {
   dir  = parsed_url.Path + "/"
   file = "index.html"
  }
  if(len(dir)>0){
    err := os.MkdirAll(parsed_url.Host + dir, 0777)
    if(err != nil){
        fmt.Println("Directory Create Error: ",dir, err)
        os.Exit(1)
    }
  }
  fileWriter, err := os.Create(parsed_url.Host + dir + file)
  if(err != nil){
      fmt.Println("File Open Error: ",err)
      os.Exit(1)
  }
  return fileWriter
}

//Converts relative links to absolute links

func fixUrl(href string, baseUrl *url.URL) (string, string) {
    link, err := url.Parse(href)
    if err!= nil{
        fmt.Println("Parsing relative link Error: ", err)
        return "", "" //ignoring invalid urls
    }
    uri := baseUrl.ResolveReference(link)
    return uri.String(), uri.Host
}

func generateLinks(resp_reader io.Reader,  uri *url.URL) {
  z := html.NewTokenizer(resp_reader)
  countLinks := 0
  for{
      tt := z.Next();
      switch{
          case tt==html.ErrorToken:
              fmt.Println("Number of links: ", countLinks)
              return
          case tt==html.StartTagToken:
              t := z.Token()

              if t.Data == "a"{
                  for _, a := range t.Attr{
                      if a.Key == "href" && !strings.Contains(a.Val, "#"){
                          link, link_host := fixUrl(a.Val, uri)
                          if link != "" {
                            if link_host == start_url.Host{
                              link = strings.TrimSuffix(link, "/")
                              _, ok := visited[link]
                              if !ok {
                                countLinks++
                                visited[link] = false
                              }
                            }
                          }
                      }
                  }
              }
        }
    }
}

func retrieveDone(syncChan chan int) {
    <-syncChan
    goCount--
}

func retrieve(uri string, syncChan chan int){
    defer retrieveDone(syncChan)

    parsed_url, err := url.Parse(uri)
    if err!= nil{
        fmt.Println("Parsing link Error: ", err)
        os.Exit(1)
    }
    fmt.Println("Fetching:  ", uri)
    resp, err := http.Get(uri)

    if(err != nil){
        fmt.Println("Http Transport Error: ", uri, "     ", err)
        return
    }
    defer resp.Body.Close()

    fileWriter := createPaths(parsed_url)
    resp_reader := io.TeeReader(resp.Body, fileWriter)
    defer fileWriter.Close()

    generateLinks(resp_reader, parsed_url)
}

func main(){
    flag.Parse()
    args := flag.Args()

    if(len(args)<1){
        fmt.Println("Specify a start page")
        os.Exit(1)
     }
     var err error
     start_url, err = url.Parse(args[0])
     if err!= nil{
         fmt.Println("Parsing Start Url Error: ",err)
         os.Exit(1)
     }
     args[0] = strings.TrimSuffix(args[0], "/")
     syncChan:= make(chan int, 10)
     visited[args[0]] = false
     for {
        allVisited := true
        for uri, done := range visited {
            if done == true {
                continue
            }
            syncChan <- 1
            visited[uri] = true
            goCount++
            go retrieve(uri, syncChan)
            allVisited = false
            break
        }
        if allVisited == true && goCount == 0 {
            break;
        }
     }
}