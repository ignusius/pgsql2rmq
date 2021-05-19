
# pgsql2rmq

Pgsql2rmq is an implementation of a fake PostgreSQL server for forwarding execution instructions and creating virtual tables for queries by templates. Pgsql2rmq send an execution to RabbitMQ (INSERT, COMMIT, BEGIN...) and generate virtual tables for queries. It project use libraries:
- github.com/panoplyio/pgsrv(MIT).
- github.com/streadway/amqp (BSD 2 clause)
- github.com/lib/pq (Copyright (c) 2011-2013, 'pq' Contributors Portions Copyright (C) 2011 Blake Mizerany)

# Build and run

```
cp -R pgsql2rmq $GOPATH/src/
go build pgsql2rmq.go

./pgsql2rmq

```

# Config
```text
{
    "Address": ["0.0.0.0","5432"],  <-- address and port  fake pgsql2rmq server
    "RabbitMQ": ["guest","guest", "0.0.0.0","5672"], <-- RabbitMQ connection
    "BehavQuerys":[  <-- Behavator Querys
        ["SELECT a.attname as",   <-- If query start as "SELECT a.attname as" 
            ["Field","Type"],  <-- Generate Columns "Field" and "Type"
            [
            ["MARK", "bigint"],  <-- Generate row
            ["TM", "timestamp with time zone"], <-- Generate row
            ["VAL", "double precision"] <-- Generate row
            ]
        ],
        ["SELECT a.attname FROM pg_class",   <-- If query start as ""SELECT a.attname FROM pg_class" 
            ["attname"],  <-- Generate Column "attname"
            [
            ["MARK"], <-- Generate row
            ["TM"] <-- Generate row
            ]
        ],
        ["SELECT c.relname as \"TableName\"", <-- If query start as 'SELECT c.relname as "TableName"'
            ["TableName"], <-- Generate Column "TableName"
            [
            ["DBArch"], <-- Generate row
            ["DBAVl_nn_P1_U_1_A"], <-- Generate row
            ["DBAVl_nn_P1_U_1_B"], <-- Generate row
            ["DBAVl_nn_P1_U_1_C"]  <-- Generate row
            ]
        ]

    ],
    "SendFilter":["^INSERT","^BEGIN","^COMMIT"], <--Filter querys from send to RMQ

    "ShowLog":true,  <--Show log in console
    "ShowSendData":true  <-- Show send data to RMQ

}
```
# Test queries
```
root=> SELECT a.attname as test;
 Field |           Type           
-------+--------------------------
 MARK  | bigint
 TM    | timestamp with time zone
 VAL   | double precision
(3 rows)

root=> SELECT a.attname FROM pg_class;
 attname 
---------
 MARK
 TM
(2 rows)

root=> SELECT c.relname as "TableName";
              TableName               
--------------------------------------
 DBArch
 DBAVl_nn_P1_U_1_A
 DBAVl_nn_P1_U_1_B
 DBAVl_nn_P1_U_1_C
(4 rows)

```

# Example client from pgsql2rmq

See [pgsql2rmq-client](pgsql2rmq-client)


![Image alt](https://upload.wikimedia.org/wikipedia/commons/a/a0/Syrischer_Maler_von_1354_001.jpg)

An illustration from a Syrian edition dated 1354. The rabbit fools the elephant king by showing him the reflection of the moon.

Contact
-------
* Developer: Alexander Komarov <ignusius@gmail.com>
