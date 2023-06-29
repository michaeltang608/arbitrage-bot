package server

type Server interface {
	QuantRun() error
	QuantClose() error
}
