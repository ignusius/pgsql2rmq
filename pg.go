package main

import (
	"context"
	"database/sql/driver"
	"fmt"
	"io"
	"net"
	"regexp"

	"pgfake/pgsrv"

	pg_query "github.com/lfittl/pg_query_go/nodes"

	_ "github.com/lib/pq"
)

var (
	s    pgsrv.Server
	mock *rows
)

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
	//context := ctx.(context.Context)
	//context.WithValue(ctx, "myKeeey", "wwwwwwwwwwwwwwwwwwww")
	fmt.Println("--->", ctx.Value(pgsrv.SqlCtxKey).(string), "<---")

	var validID = regexp.MustCompile(`SELECT a.attname as `)
	var validID2 = regexp.MustCompile(`SELECT count`)
	var validID3 = regexp.MustCompile(`SELECT a.attname FROM pg_class`)
	var validID4 = regexp.MustCompile("SELECT c.relname as \"TableName\"")

	if validID.MatchString(ctx.Value(pgsrv.SqlCtxKey).(string)) {
		mock.AddCol("Field")
		mock.AddCol("Type")
		mock.AddRows([]string{"MARK", "bigint"})
		mock.AddRows([]string{"TM", "timestamp with time zone"})
		mock.AddRows([]string{"VAL", "double precision"})

	}
	if validID2.MatchString(ctx.Value(pgsrv.SqlCtxKey).(string)) {
		mock.AddCol("count")
		mock.AddRowsInt([]int32{1})

	}

	if validID3.MatchString(ctx.Value(pgsrv.SqlCtxKey).(string)) {
		mock.AddCol("attname")
		mock.AddRows([]string{"MARK"})
		mock.AddRows([]string{"TM"})

	}

	if validID4.MatchString(ctx.Value(pgsrv.SqlCtxKey).(string)) {
		mock.AddCol("TableName")
		mock.AddRows([]string{"DBAVl_nn_P1_U_1_A"})
		mock.AddRows([]string{"DBAVl_nn_P1_U_1_B"})
		mock.AddRows([]string{"DBAVl_nn_P1_U_1_C"})

	}

	return mock, nil
}

func main() {

	//mock.Next([)

	ln, err := net.Listen("tcp", ":5432")
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
		fmt.Println("-->", <-pgsrv.GlobCtx)
	}

}
