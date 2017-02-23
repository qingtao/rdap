package rdap

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"log"
	"net/http"
)

type Router struct {
	Code  string `json:"code"`
	Name  string `json:"name,omitempty"`
	Wanip string `json:"wanip,omitempty"`
	Area  string `json:"area,omitempty"`
	SP    string `json:"service_provider,omitempty"`
	AU    int    `json:"auto_update"`
}

//连接数据库
func OpenMysql(host, port, user, password, database string) *sql.DB {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		user, password, host, port, database))
	if err != nil {
		log.Fatalf("open mysql failed: %s", err)
	}
	return db
}

func UpdateRouterSP(db *sql.DB, r *Router) error {
	r = ToUpper(r)
	_, err := db.Exec(`update routers set sp=? where code = ?`, r.SP, r.Code)
	return err
}

func UpdateSP(URL, ip) error {
	r, err := http.Get(`http://10.62.7.117:50053/admin/r/g`)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	var router = make([]*Router, 0)
	if err = json.Unmarshal(b, &router); err != nil {
		t.Fatalf("%s\n", err)
	}
	for i := 0; i < len(router); i++ {
		go func(i int) {
			as, err := GetJSON(rdapAddr, router[i].Wanip)
			if err != nil {
				fmt.Printf("%s\n", err)
			}

		}(i)
	}
	return nil
}
