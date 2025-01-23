# DUMBO

Simple session manager



## Quick Start

This is an example of using the `dumbo` session manager with an http server. In the main function initialize you session manager and a database storage. Database storage being optional, if none is specified the session manager will use only the process memory

```go 
package main

import (
  "net/http"
  "github.com/racg0092/dumbo"
  "os"
  "fmt"
)

func main() {
  port := os.Getenv("PORT")
  if port == "" {
    port = "8080"
  }

  // this is an optional step
  // note: only mongo db is supported at the moment but anyone can implement the storage interface
  store, err := dumbo.NewMongoStore("db", "collection", os.Getenv("db_conn"))
  if err != nil {
    panic(err)
  }

  // starts the session
  dumbo.Start(dumbo.Options{Secure: true, HttpOnly: true, MaxAge: 60 *30}, store)


  mux := http.NewServerMux()


  mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    // get the session if none has been stared one will be initialize
    sess := dumbo.Get(r, w, "sess")
    if sess.Values["count"] == nil {
      var initial int32 = 0
      sesss.Values["count"] = initial
    }
    sess.Values["count"] = sess.Values["count"].(int32) + 1
    err := sess.Save(w)
    if err != nil {
      w.WriteHeader(http.StatusInternalServerError)
      return
    }
    w.Write([]byte("Count is " + sess.Values["count"]))
  })


  server := http.Server{Addr: ":" + port, Handler: mux}

  fmt.Println("Server running on :", port)

  if err := server.ListenAndServe(); err != nil {
    panic(err)
  }
}
```

