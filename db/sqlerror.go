package db

type SQLError interface {
	error
	// commonly implemented pattern
	SQLState() string
}
