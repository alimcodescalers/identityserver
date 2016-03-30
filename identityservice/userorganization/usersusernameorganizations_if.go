package userorganization

//This file is auto-generated by go-raml
//Do not edit this file by hand since it will be overwritten during the next generation

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/justinas/alice"
)

// UsersusernameorganizationsInterface is interface for /users/{username}/organizations root endpoint
type UsersusernameorganizationsInterface interface {
	// Get is the handler for GET /users/{username}/organizations
	// Get the list organizations a user is owner or member of
	Get(http.ResponseWriter, *http.Request)
	// globalidrolesrolePost is the handler for POST /users/{username}/organizations/{globalid}/roles/{role}
	// Accept membership in organization
	globalidrolesrolePost(http.ResponseWriter, *http.Request)
	// globalidrolesroleDelete is the handler for DELETE /users/{username}/organizations/{globalid}/roles/{role}
	// Reject membership invitation in an organization.
	globalidrolesroleDelete(http.ResponseWriter, *http.Request)
}

// UsersusernameorganizationsInterfaceRoutes is routing for /users/{username}/organizations root endpoint
func UsersusernameorganizationsInterfaceRoutes(r *mux.Router, i UsersusernameorganizationsInterface) {
	r.Handle("/users/{username}/organizations", alice.New(newOauth2oauth_2_0Middleware([]string{"user:admin"}).Handler).Then(http.HandlerFunc(i.Get))).Methods("GET")
	r.Handle("/users/{username}/organizations/{globalid}/roles/{role}", alice.New(newOauth2oauth_2_0Middleware([]string{"user:admin"}).Handler).Then(http.HandlerFunc(i.globalidrolesrolePost))).Methods("POST")
	r.Handle("/users/{username}/organizations/{globalid}/roles/{role}", alice.New(newOauth2oauth_2_0Middleware([]string{"user:admin"}).Handler).Then(http.HandlerFunc(i.globalidrolesroleDelete))).Methods("DELETE")
}