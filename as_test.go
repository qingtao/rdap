package rdap

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

var rdapAddr = `https://rdap.apnic.net`
var ip = `202.102.152.3`

func TestIP(t *testing.T) {
	as, err := GetJSON(rdapAddr, ip)
	if err != nil {
		t.Fatalf("%s\n", err)
	}
	t.Run("JSON", func(t *testing.T) {
		b, err := json.MarshalIndent(as, "", "  ")
		if err != nil {
			t.Fatalf("%s\n", err)
		}
		fmt.Printf("%s\n", b)
	})
	t.Run("IP", func(t *testing.T) {
		ia, err := GetAddr(as, ip)
		if err != nil {
			t.Fatalf("%s\n", err)
		}
		fmt.Printf("%s\n", ia)
	})
}

func TestGetSP(t *testing.T) {
	r, err := http.Get(`http://10.62.7.117:50053/admin/r/g`)
	if err != nil {
		t.Fatalf("%s\n", err)
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("%s\n", err)
	}
	var router = make([]*Router, 0)
	if err = json.Unmarshal(b, &router); err != nil {
		t.Fatalf("%s\n", err)
	}
	var str = make(chan string, len(router))
	go func() {
		for i := 0; i < len(router); i++ {
			go func(i int) {
				as, err := GetJSON(rdapAddr, router[i].Wanip)
				if err != nil {
					fmt.Printf("%s\n", err)
				}
				str <- fmt.Sprintf("%s: %s", router[i].Code, as.Name)
			}(i)
		}
	}()
	i := 1
	for {
		select {
		case s := <-str:
			fmt.Println(i, s)
			i++
		case <-time.After(time.Second * 10):
			if i >= len(router) {
				fmt.Println("done!")
				return
			}
		}
	}
}
