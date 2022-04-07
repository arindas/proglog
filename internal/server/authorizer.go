package server

type Authorizer interface {
	Authorize(subject, object, action string) error
}
