package api

//SortType is a SQL sort type
type SortType int

//SortTypes
const (
	SortNone SortType = iota
	SortAscending
	SortDescending
)

//OperationType is a SQL operation type
type OperationType int

//OperationTypes
const (
	OperationEquals OperationType = iota
	OperationNotEquals
	OperationIsNull
	OperationIsNotNull
	OperationLessThan
	OperationGreaterThan
	OperationLessThanOrEqualTo
	OperationGreaterThanOrEqualTo
	OperationContains
	OperatationStartsWith
	OperationEndsWith
	OperationRegexp
)

//BooleanType is a SQL boolean type
type BooleanType int

//BooleanTypes
const (
	BooleanAND BooleanType = iota
	BooleanOR
	BooleanXOR
	BooleanNOT
)

//Parameter is a SQL query parameter
type Parameter struct {
	Field     string        `json:"field"`
	Operation OperationType `json:"operation"`
	Value     interface{}   `json:"value"`
	Sort      SortType      `json:"sort,omitempty"`
}

//ParameterTree is a SQL query parameter tree
type ParameterTree struct {
	Parameters []*Parameter     `json:"parameters,omitempty"`
	Trees      []*ParameterTree `json:"trees,omitempty"`
	Boolean    BooleanType      `json:"boolean,omitempty"`
}

//Search is a SQL query
type Search struct {
	Tree   *ParameterTree `json:"tree"`
	Offset int64          `json:"offset,omitempty"`
	Limit  int            `json:"limit,omitempty"`
}
