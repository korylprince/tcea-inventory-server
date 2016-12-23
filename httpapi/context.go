package httpapi

type contextKey int

//TransactionKey is the context key for the database transaction for a request
const TransactionKey contextKey = 0

//UserKey is the context key for the user for a request
const UserKey contextKey = 1
