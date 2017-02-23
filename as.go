package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	//响应类型
	jsonType = "application/rdap+json"

	rdapAddr = `https://rdap.apnic.net`

	UNICOM   = "联通"
	CHINANET = "电信"
	UNKNOWN  = "未知"
)

func FindSP(s string) string {
	switch {
	case strings.HasPrefix(s, "UNICOM"):
		return UNICOM
	case strings.HasPrefix(s, "CHINANET"):
		return CHINANET
	}
	return UNKNOWN
}

type AS struct {
	Handle       string `json:"handle,omitempty"`
	StartAddress string `json:"startAddress,omitempty"`
	EndAddress   string `json:"endAddress,omitempty"`
	IpVersion    string `json:"ipVersion,omitempty"`
	Name         string `json:"name,omitempty"`
	Type         string `json:"type,omitempty"`
	Country      string `json:"country,omitempty"`
	ParentHandle string `json:"parentHandle,omitempty"`
	//ObjectClassName:
	//1. domains
	//2. nameservers
	//3. entities
	//4. IP networks
	//5. Autonomous System numbers
	Lang            string `json:"lang,omitempty"`
	ObjectClassName string `json:"objectClassName,omitempty"`

	Entities []*Entity `json:"entities,omitempty"`

	Remarks []*Remark `json:"remarks,omitempty"`

	Links []*Link `json:"links,omitempty"`

	Roles []string `json:"roles,omitempty"`

	Events []*Event `json:"events,omitempty"`

	RdapConformance []string `json:"rdapConformance"`

	Notices   []*Notice   `json:"notices,omitempty"`
	PublicIds []*PublicId `json:"publicId,omitempty"`
	Status    []string    `json:"status,omitempty"`

	//whois服务器地址
	Port43 string `json:"port43,omitempty"`
	//
	Title       string   `json:"title,omitempty"`
	ErrorCode   int      `json:"errorCode,omitempty"`
	Description []string `json:"description,omitempty"`
}

type Entity struct {
	Handle          string      `json:"handle,omitempty"`
	VcardArray      interface{} `json:"vcardArray,omitempty"`
	Entities        []*Entity   `json:"entities,omitempty"`
	Roles           []string    `json:"roles,omitempty"`
	Remarks         []*Remark   `json:"remarks,omitempty"`
	Events          []*Event    `json:"events,omitempty"`
	AsEventActor    string      `json:"asEventActor,omitempty"`
	ObjectClassName string      `json:"objectClassName"`
	Links           []*Link     `json:"links,omitempty"`
	PublicIds       []*PublicId `json:"publicId,omitempty"`
	Status          []string    `json:"status,omitempty"`
}

type PublicId struct {
	Type       string `json:"type,omitempty"`
	Identifier string `json:"identifier,omitempty"`
}

type Link struct {
	Value    string `json:"value"`
	Rel      string `json:"self,omitempty"`
	Href     string `json:"href"`
	Hreflang string `json:"hreflang,omitempty"`
	Title    string `json:"tilte,omitempty"`
	Media    string `json:"media,omitempty"`
	Type     string `json:"type,omitempty"`
}

type Remark struct {
	Title       string   `json:"title,omitempty"`
	Type        string   `json:"type,omitempty"`
	Description []string `json:description,omitempty"`
	Links       []*Link  `json:"links,omitempty"`
}

type Event struct {
	EventAction string  `json:"eventAction,omitempty"`
	EventActor  string  `json:"eventActor,omitempty"`
	EventDate   string  `json:"eventDate,omitempty"`
	Links       []*Link `json:"links,omitempty"`
}

type Notice struct {
	Title       string   `json:"title,omitempty"`
	Description []string `json:"description,omitempty"`
	Links       []*Link  `json:"links,omitempty"`
}

type IPAddr struct {
	IP      string `json:"ip"`
	Addr    string `json:"address"`
	Country string `json:"country"`
}

func GetJSON(rawUrl, ip string) (*AS, error) {
	n := strings.Index(ip, "/")
	if n > 0 {
		_, _, err := net.ParseCIDR(ip)
		if err != nil {
			return nil, err
		}
	}
	if IP := net.ParseIP(ip); IP == nil {
		return nil, errors.New("ip invalid")
	}

	resp, err := http.Get(fmt.Sprintf("%s/ip/%s", rawUrl, ip))
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var as AS
	if err = json.Unmarshal(b, &as); err != nil {
		return nil, err
	}
	return &as, nil
}

func GetAddr(as *AS, ip string) (*IPAddr, error) {
	addr := as.Name
	if len(as.Remarks) > 0 {
		for i := 0; i < len(as.Remarks); i++ {
			rk := as.Remarks[i]
			if rk.Title == "description" && len(rk.Description) > 0 {
				for j := 0; j < len(rk.Description); j++ {
					addr += fmt.Sprintf(", %s", rk.Description[j])
				}
			}
		}
	} else {
		return nil, errors.New("unknown")
	}
	return &IPAddr{IP: ip, Country: as.Country, Addr: addr}, nil
}

func UpdateSP(ctx context.Context, db *sql.DB, ch chan<- error, done chan<- bool, addr string) {
	client := &http.Client{Timeout: time.Second * 10}
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/admin/r/g", addr), nil)
	if err != nil {
		ch <- err
		return
	}
	r, err := client.Do(req)
	if err != nil {
		ch <- err
		return
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ch <- err
		return
	}
	var routers = make([]*Router, 0)
	if err = json.Unmarshal(b, &routers); err != nil {
		ch <- err
		return
	}
	var wg sync.WaitGroup

	for i := 0; i < len(routers); i++ {
		wg.Add(1)
		router := routers[i]
		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(30*time.Second))
		defer cancel()
		go func(ctx context.Context, r *Router, ch chan<- error) {
			as, err := GetJSON(rdapAddr, r.Wanip)
			if err != nil {
				select {
				case <-ctx.Done():
					err = ctx.Err()
				default:
				}
			} else {
				r.SP = FindSP(as.Name)
				if err = UpdateRouterSP(db, r); err != nil {
					select {
					case <-ctx.Done():
						err = ctx.Err()
					default:
					}
				}
			}
			if err != nil {
				ch <- errors.New(fmt.Sprintf("update sp of %s: %s\n", r.Code, err))
			}
			wg.Done()

		}(ctx, router, ch)
	}
	wg.Wait()
	done <- true

}
