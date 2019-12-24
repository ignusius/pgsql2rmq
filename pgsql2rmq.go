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

//Virual cols and rows
type rows struct {
	cols []string
	rows [][]driver.Value
}

//Clear cols and rows
func (rows *rows) Close() error {
	rows.cols = []string{}
	rows.rows = [][]driver.Value{}
	return nil
}

//Return colums
func (rows *rows) Columns() []string {
	return rows.cols
}

//Implement driver.Rows method Next
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

//Add column in virtual table
func (rows *rows) AddCol(name string) {
	rows.cols = append(rows.cols, name)
}

//Add rows in virtual table
func (rows *rows) AddRows(v []interface{}) {
	row := make([]driver.Value, len(v))
	for i := 0; i < len(v); i++ {
		row[i], _ = driver.DefaultParameterConverter.ConvertValue(v[i])

	}

	rows.rows = append(rows.rows, row)

}

//Return virtual table from SQL-query by tamplates
func (rows *rows) Query(ctx context.Context, node pg_query.Node) (driver.Rows, error) {

	rows.Close()

	if config.ShowLog {

		fmt.Println("Q-->", ctx.Value(pgsrv.SqlCtxKey).(string), "<--Q")
	}

	for x := len(config.BehavQuerys) - 1; x >= 0; x-- {

		valid := regexp.MustCompile((config.BehavQuerys[x][0]).(string))
		if valid.MatchString(ctx.Value(pgsrv.SqlCtxKey).(string)) {
			for _, col := range config.BehavQuerys[x][1].([]interface{}) {
				rows.AddCol(col.(string))
			}
			for _, rowsarr := range config.BehavQuerys[x][2].([]interface{}) {
				rows.AddRows(interfaceArr(rowsarr))
			}

		}
	}

	return rows, nil
}

//Transform slice interface{} to []interface{}
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

	ln, err := net.Listen("tcp", config.Address[0]+":"+config.Address[1])
	if err != nil {
		fmt.Println("net.Listen error")
	}
	//loop from form virtual tables
	go func() {
		fmt.Printf("%-20s %10s\n", "PgSQL2RMQ starting", "[ OK ]")
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
		dataregexp := regexp.MustCompile(`'(\d*\.?\d*)'`)
		datasql := dataregexp.ReplaceAllString(data["SQL"].(string), " $1")

		ses := data["Session"].(map[string]interface{})

		if config.ShowLog {
			fmt.Println("S-->", "Username:", ses["user"], ", Database:", ses["database"], "<--S")
			fmt.Println("Q-->", datasql, "<--Q")
		}

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
