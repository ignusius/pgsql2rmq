package main

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"regexp"
	"strings"

	"pgfake/pgsrv"

	pg_query "github.com/lfittl/pg_query_go/nodes"

	_ "github.com/lib/pq"
)

var (
	s      pgsrv.Server
	mock   *rows
	config configuration
)

type configuration struct {
	Address      []string
	Pgsql        []string
	BehavQuerys  [][]interface{}
	SendFilter   []string
	ShowLog      bool
	ShowSendData bool
}

type rows struct {
	cols []string
	rows [][]driver.Value
}

func (*rows) Close() error {
	panic("not implemented")

}

func (rows *rows) Columns() []string {
	return rows.cols
}

func (rows *rows) Next(dest []driver.Value) error {
	if len(rows.rows) == 0 {
		return io.EOF
	}

	for i, v := range rows.rows[0] {
		dest[i] = v
	}

	rows.rows = rows.rows[1:]
	return nil
}

func (rows *rows) AddCol(name string) {
	rows.cols = append(rows.cols, name)
}

func (rows *rows) AddRows(v []string) {
	//var err error
	row := make([]driver.Value, len(v))
	for i := 0; i < len(v); i++ {
		row[i], _ = driver.String.ConvertValue(v[i])
	}

	rows.rows = append(rows.rows, row)

}

func (rows *rows) AddRowsInt(v []int32) {
	//var err error
	row := make([]driver.Value, len(v))
	for i := 0; i < len(v); i++ {
		row[i], _ = driver.Int32.ConvertValue(v[i])
	}

	rows.rows = append(rows.rows, row)

}

func (rows *rows) Query(ctx context.Context, node pg_query.Node) (driver.Rows, error) {

	if config.ShowLog {

		fmt.Println("Q-->", ctx.Value(pgsrv.SqlCtxKey).(string), "<--Q")
	}

	valid := regexp.MustCompile((config.BehavQuerys[0][0]).(string))
	if valid.MatchString(ctx.Value(pgsrv.SqlCtxKey).(string)) {
		for _, col := range config.BehavQuerys[0][1].([]interface{}) {
			mock.AddCol(col.(string))
		}

		for _, rowsarr := range config.BehavQuerys[0][2].([]interface{}) {
			mock.AddRows(interfaceStringArr(rowsarr))
		}

	}

	return mock, nil
}

func interfaceStringArr(slice interface{}) []string {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		panic("InterfaceSlice() given a non-slice type")
	}

	ret := make([]string, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface().(string)
	}

	return ret
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if strings.EqualFold(a, e) {
			return true
		}
	}
	return false
}

func main() {

	file, _ := os.Open("config.json")
	decoder := json.NewDecoder(file)
	config = configuration{}
	err := decoder.Decode(&config)
	if err != nil {
		fmt.Println("!-->error configuration file:", err, "<--!")
	}

	//mock.Next([)

	ln, err := net.Listen("tcp", config.Address[0]+":"+config.Address[1])
	if err != nil {
		fmt.Println("net.Listen error")
	}

	go func() {
		for {

			mock = &rows{}
			s = pgsrv.New(mock)

			conn, err := ln.Accept()
			if err != nil {
				fmt.Println("Accept error")
			}
			defer conn.Close()

			go s.Serve(conn)
			if err != nil {
				fmt.Println("s.Serve error")
			}

		}
	}()
	for {
		data := <-pgsrv.GlobCtx
		if config.ShowLog {
			fmt.Println("S-->", data["Session"], "<--S")
			fmt.Println("Q-->", data["SQL"], "<--Q")
		}
		splitarr := strings.Split(data["SQL"].(string), " ")

		for _, filter := range config.SendFilter {
			if contains(splitarr, filter) {
				if config.ShowSendData {
					fmt.Println("==>", data["SQL"].(string))
				}
			}
		}

	}

}
