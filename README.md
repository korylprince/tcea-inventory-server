#Info

This is the back end server for [tcea-inventory-client](https://github.com/korylprince/tcea-inventory-client), an inventory system the Tech Team for the annual [TCEA](tcea.org) convention uses.

#Install

```
go get github.com/korylprince/tcea-inventory-server
```

Create a MySQL database with `model.sql`.

#Configuration

INVENTORY_SESSIONDURATION="60" #in minutes
INVENTORY_SQLDRIVER="mysql"
INVENTORY_SQLDSN="username:password@tcp(server:3306)/database?parseTime=true"
INVENTORY_LISTENADDR=":8080"
INVENTORY_PREFIX="/inventory" #URL prefix
