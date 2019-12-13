package main

import (
	"bytes"
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

	"pgsql2rmq/pgsrv"

	pg_query "github.com/lfittl/pg_query_go/nodes"
	"github.com/streadway/amqp"

	_ "github.com/lib/pq"
)

var (
	s      pgsrv.Server
	mock   *rows
	config configuration
)

type configuration struct {
	Address      []string
	RabbitMQ     []string
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

func (rows *rows) Close() error {
	rows.cols = []string{}
	rows.rows = [][]driver.Value{}
	return nil

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

func (rows *rows) AddRows(v []interface{}) {
	//var err error
	row := make([]driver.Value, len(v))
	for i := 0; i < len(v); i++ {
		row[i], _ = driver.DefaultParameterConverter.ConvertValue(v[i])

	}

	rows.rows = append(rows.rows, row)

}

func (rows *rows) Query(ctx context.Context, node pg_query.Node) (driver.Rows, error) {

	rows.Close()

	fmt.Println("-------------------->", mock)
	fmt.Println("---------------r----->", rows)

	if config.ShowLog {

		fmt.Println("Q-->", ctx.Value(pgsrv.SqlCtxKey).(string), "<--Q")
	}

	valid := regexp.MustCompile((config.BehavQuerys[0][0]).(string))
	if valid.MatchString(ctx.Value(pgsrv.SqlCtxKey).(string)) {
		for _, col := range config.BehavQuerys[0][1].([]interface{}) {
			rows.AddCol(col.(string))
		}
		for _, rowsarr := range config.BehavQuerys[0][2].([]interface{}) {
			rows.AddRows(interfaceArr(rowsarr))
		}

	}

	return rows, nil
}

func interfaceArr(slice interface{}) []interface{} {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		panic("InterfaceSlice() given a non-slice type")
	}

	ret := make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret
}

func contains(s []string, e string) bool {
	str := bytes.Trim([]byte(e), "\x00")

	for _, a := range s {

		if strings.EqualFold(a, string(str)) {
			return true
		}
		if strings.EqualFold(a+";", string(str)) {

			return true
		}

	}
	return false
}

func main() {

	fmt.Printf("%-20s %10s\n", "Prepare server", "[ OK ]")
	file, _ := os.Open("config.json")
	decoder := json.NewDecoder(file)
	config = configuration{}
	err := decoder.Decode(&config)
	if err != nil {
		fmt.Printf("%-20s %10s\n", "Read configuration", "[ ERROR ]")
		fmt.Println("E-->", err, "<--E")
		os.Exit(1)
	}
	fmt.Printf("%-20s %10s\n", "Read configuration", "[ OK ]")

	conn, err := amqp.Dial("amqp://" + config.RabbitMQ[0] + ":" + config.RabbitMQ[1] + "@" + config.RabbitMQ[2] + ":" + config.RabbitMQ[3] + "/")

	if err != nil {
		fmt.Printf("%-20s %10s\n", "Connect to RabbitMQ", "[ ERROR ]")
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("%-20s %10s\n", "Connect to RabbitMQ", "[ OK ]")

	ch, err := conn.Channel()

	if err != nil {
		fmt.Println("Conn.Channel error")
	}
	defer ch.Close()

	//mock.Next([)

	ln, err := net.Listen("tcp", config.Address[0]+":"+config.Address[1])
	if err != nil {
		fmt.Println("net.Listen error")
	}

	go func() {
		fmt.Printf("%-20s %10s\n", "PgSQL2RMQ starting", "[ OK ]")
		for {
			fmt.Println("ssssssssssssssssssssssssssssssss")
			//rows := &rows{}
			s = pgsrv.New(&rows{})

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
		dataregexp := regexp.MustCompile(`'(\d*\.?\d*)'`)
		datasql := dataregexp.ReplaceAllString(data["SQL"].(string), " $1")

		ses := data["Session"].(map[string]interface{})

		if config.ShowLog {
			fmt.Println("S-->", "Username:", ses["user"], ", Database:", ses["database"], "<--S")
			fmt.Println("Q-->", datasql, "<--Q")
		}

		//splitarr := strings.Split(data["SQL"].(string), "$1")

		//if contains(config.SendFilter, splitarr[0]) {
		//	if config.ShowSendData {
		//fmt.Println("==>", data["SQL"].(string))
		for _, filter := range config.SendFilter {

			valid := regexp.MustCompile(filter)
			if valid.MatchString(datasql) {
				q, err := ch.QueueDeclare(
					ses["database"].(string), // name
					false, // durable
					false, // delete when unused
					false, // exclusive
					false, // no-wait
					nil,   // arguments
				)
				if err != nil {
					fmt.Println("Failed to declare a queue", err)
				}

				body := datasql
				err = ch.Publish(
					"",     // exchange
					q.Name, // routing key
					false,  // mandatory
					false,  // immediate
					amqp.Publishing{
						ContentType: "text/plain",
						Body:        []byte(body),
					})
			}

			//}
		}

	}

}
