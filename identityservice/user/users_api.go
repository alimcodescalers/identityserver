package user

import (
	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/itsyouonline/identityserver/identityservice/contract"
	"github.com/itsyouonline/identityserver/identityservice/invitations"
)

type UsersAPI struct {
}

// It is handler for POST /users
func (api UsersAPI) Post(w http.ResponseWriter, r *http.Request) {

	var u User

	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	userMgr := NewManager(r)
	if err := userMgr.Save(&u); err != nil {
		log.Error("ERROR while saving user:\n", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(&u)
}

// It is handler for GET /users/{username}
func (api UsersAPI) usernameGet(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]

	userMgr := NewManager(r)

	user, err := userMgr.GetByName(username)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// Update an existing user. Updating ``username`` is not allowed. The labelled lists
// can not be updated this way, the normal properties can (like github and facebook account).
// It is handler for PUT /users/{username}
func (api UsersAPI) usernamePut(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]

	var u User

	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	userMgr := NewManager(r)

	oldUser, uerr := userMgr.GetByName(username)
	if uerr != nil {
		log.Debug(uerr)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if u.Username != username || u.getID() != oldUser.getID() {
		http.Error(w, "Changing username or id is Forbidden!", http.StatusForbidden)
		return
	}

	// Update only selected fields!
	// Other fields should be update explicitly via their own handlers.
	oldUser.Facebook = u.Facebook
	oldUser.Github = u.Github
	oldUser.PublicKeys = u.PublicKeys

	if err := userMgr.Save(oldUser); err != nil {
		log.Error("ERROR while saving user:\n", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&u)
}

func isValidLabel(label string) (valid bool) {
	valid = true
	labelLength := len(label)
	valid = valid && labelLength > 2 && labelLength < 51

	if !valid {
		log.Debug("Invalid label: ", label)
	}
	return valid
}

// RegisterNewEmailAddress is the handler for POST /users/{username}/emailaddresses
// Register a new email address
func (api UsersAPI) RegisterNewEmailAddress(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	body := struct {
		Label        string
		Emailaddress string
	}{}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if !isValidLabel(body.Label) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	userMgr := NewManager(r)
	u, err := userMgr.GetByName(username)
	if err != nil {
		log.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if _, ok := u.Email[body.Label]; ok {
		http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
		return
	}

	if err = userMgr.SaveEmail(username, body.Label, body.Emailaddress); err != nil {
		log.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(body)
}

// UpdateEmailAddress is the handler for PUT /users/{username}/emailaddresses/{label}
// Updates the label and/or value of an email address
func (api UsersAPI) UpdateEmailAddress(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	oldlabel := mux.Vars(r)["label"]

	body := struct {
		Label        string
		Emailaddress string
	}{}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if !isValidLabel(body.Label) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	userMgr := NewManager(r)

	if oldlabel != body.Label {
		u, err := userMgr.GetByName(username)
		if err != nil {
			log.Error(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if _, ok := u.Email[body.Label]; ok {
			http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
			return
		}
	}

	if err := userMgr.SaveEmail(username, body.Label, body.Emailaddress); err != nil {
		log.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if oldlabel != body.Label {
		if err := userMgr.RemoveEmail(username, oldlabel); err != nil {
			log.Error(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(body)
}

// DeleteEmailAddress is the handler for DELETE /users/{username}/emailaddresses/{label}
// Removes an email address
func (api UsersAPI) DeleteEmailAddress(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	label := mux.Vars(r)["label"]

	userMgr := NewManager(r)

	u, err := userMgr.GetByName(username)
	if err != nil {
		log.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if _, ok := u.Email[label]; !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if len(u.Email) == 1 {
		http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
		return
	}

	if err = userMgr.RemoveEmail(username, label); err != nil {
		log.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)

}

// GetUserInformation is the handler for GET /users/{username}/info
func (api UsersAPI) GetUserInformation(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	userMgr := NewManager(r)

	user, err := userMgr.GetByName(username)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	addresses := map[string]Address{}
	emails := map[string]string{}
	phones := map[string]Phonenumber{}

	// TODO: apply authorization limits and scope mapping.
	for label, address := range user.Address {
		addresses[label] = address
	}

	for label, email := range user.Email {
		emails[label] = email
	}

	for label, phone := range user.Phone {
		phones[label] = phone
	}

	respBody := &userview{
		Address:  addresses,
		Email:    emails,
		Phone:    phones,
		Username: user.Username,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(respBody)
}

// usernamevalidateGet is the handler for GET /users/{username}/validate
func (api UsersAPI) usernamevalidateGet(w http.ResponseWriter, r *http.Request) {

	// token := req.FormValue("token")

	// uncomment below line to add header
	// w.Header.Set("key","value")
}

// RegisterNewPhonenumber is the handler for POST /users/{username}/phonenumbers
// Register a new phonenumber
func (api UsersAPI) RegisterNewPhonenumber(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	userMgr := NewManager(r)

	u, err := userMgr.GetByName(username)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	body := struct {
		Label       string
		Phonenumber Phonenumber
	}{}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if !isValidLabel(body.Label) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if !IsValidPhonenumber(body.Phonenumber) {
		log.Debug("Invalid phonenumber: ", body.Phonenumber)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	//Check if this label is already used
	if _, ok := u.Phone[body.Label]; ok {
		http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
		return
	}

	if err := userMgr.SavePhone(username, body.Label, body.Phonenumber); err != nil {
		log.Error("ERROR while saving a phonenumber - ", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// respond with created phone number.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(body)
}

// usernamephonenumbersGet is the handler for GET /users/{username}/phonenumbers
func (api UsersAPI) usernamephonenumbersGet(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	userMgr := NewManager(r)

	user, err := userMgr.GetByName(username)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user.Phone)
}

// usernamephonenumberslabelGet is the handler for GET /users/{username}/phonenumbers/{label}
func (api UsersAPI) usernamephonenumberslabelGet(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	label := mux.Vars(r)["label"]
	userMgr := NewManager(r)

	user, err := userMgr.GetByName(username)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if _, ok := user.Phone[label]; ok != true {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	respBody := map[string]Phonenumber{
		label: user.Phone[label],
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(respBody)
}

// UpdatePhonenumber is the handler for PUT /users/{username}/phonenumbers/{label}
// Update the label and/or value of an existing phonenumber.
func (api UsersAPI) UpdatePhonenumber(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	oldlabel := mux.Vars(r)["label"]

	body := struct {
		Label       string
		Phonenumber Phonenumber
	}{}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if !isValidLabel(body.Label) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if !IsValidPhonenumber(body.Phonenumber) {
		http.Error(w, "Invalid phone number!", http.StatusBadRequest)
		return
	}

	userMgr := NewManager(r)

	u, err := userMgr.GetByName(username)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if _, ok := u.Phone[oldlabel]; ok != true {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if oldlabel != body.Label {
		if _, ok := u.Phone[body.Label]; ok {
			http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
			return
		}
	}

	if err = userMgr.SavePhone(username, body.Label, body.Phonenumber); err != nil {
		log.Error("ERROR while saving phonenumber - ", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if oldlabel != body.Label {
		if err := userMgr.RemovePhone(username, oldlabel); err != nil {
			log.Error(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(body)

}

// DeletePhonenumber is the handler for DELETE /users/{username}/phonenumbers/{label}
// Removes a phonenumber
func (api UsersAPI) DeletePhonenumber(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	label := mux.Vars(r)["label"]
	userMgr := NewManager(r)

	user, err := userMgr.GetByName(username)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if _, ok := user.Phone[label]; ok != true {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if err := userMgr.RemovePhone(username, label); err != nil {
		log.Error("ERROR while saving user:\n", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Create new bank account
// It is handler for POST /users/{username}/banks
func (api UsersAPI) usernamebanksPost(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	userMgr := NewManager(r)

	user, err := userMgr.GetByName(username)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	var bank map[string]BankAccount

	if err := json.NewDecoder(r.Body).Decode(&bank); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	// Only allow creation of One bank account
	if len(bank) > 1 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	for label, val := range bank {
		if _, ok := user.Bank[label]; ok {
			// Do not allow creating existing label!
			http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
			return
		}

		if err := userMgr.SaveBank(user, label, val); err != nil {
			log.Error("ERROR while saving user:\n", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

	}

	// respond with created phone number.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(bank)
}

// It is handler for GET /users/{username}/banks
func (api UsersAPI) usernamebanksGet(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	userMgr := NewManager(r)

	user, err := userMgr.GetByName(username)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user.Bank)
}

// It is handler for GET /users/{username}/banks/{label}
func (api UsersAPI) usernamebankslabelGet(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	label := mux.Vars(r)["label"]
	userMgr := NewManager(r)

	user, err := userMgr.GetByName(username)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if _, ok := user.Bank[label]; ok != true {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	respBody := map[string]BankAccount{
		label: user.Bank[label],
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(respBody)
}

// Update or create an existing bankaccount.
// It is handler for PUT /users/{username}/banks/{label}
func (api UsersAPI) usernamebankslabelPut(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	label := mux.Vars(r)["label"]
	userMgr := NewManager(r)

	user, err := userMgr.GetByName(username)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if _, ok := user.Bank[label]; ok != true {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	var bank map[string]BankAccount

	if err := json.NewDecoder(r.Body).Decode(&bank); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if _, ok := bank[label]; ok != true {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := userMgr.SaveBank(user, label, bank[label]); err != nil {
		log.Error("ERROR while saving user:\n", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bank)
}

// Delete a BankAccount
// It is handler for DELETE /users/{username}/banks/{label}
func (api UsersAPI) usernamebankslabelDelete(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	label := mux.Vars(r)["label"]
	userMgr := NewManager(r)

	user, err := userMgr.GetByName(username)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if _, ok := user.Bank[label]; ok != true {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if err := userMgr.RemoveBank(user, label); err != nil {
		log.Error("ERROR while saving user:\n", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RegisterNewAddress is the handler for POST /users/{username}/addresses
// Register a new address
func (api UsersAPI) RegisterNewAddress(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	userMgr := NewManager(r)

	body := struct {
		Label   string
		Address Address
	}{}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Debug("Error while decoding the body: ", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if !isValidLabel(body.Label) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	u, err := userMgr.GetByName(username)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	//Check if this label is already used
	if _, ok := u.Address[body.Label]; ok {
		http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
		return
	}

	if err := userMgr.SaveAddress(username, body.Label, body.Address); err != nil {
		log.Error("ERROR while saving address:\n", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// respond with created phone number.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(body)
}

// It is handler for GET /users/{username}/addresses
func (api UsersAPI) usernameaddressesGet(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	userMgr := NewManager(r)

	user, err := userMgr.GetByName(username)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user.Address)
}

// It is handler for GET /users/{username}/addresses/{label}
func (api UsersAPI) usernameaddresseslabelGet(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	label := mux.Vars(r)["label"]
	userMgr := NewManager(r)

	user, err := userMgr.GetByName(username)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if _, ok := user.Address[label]; ok != true {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	respBody := map[string]Address{
		label: user.Address[label],
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(respBody)
}

// UpdateAddress is the handler for PUT /users/{username}/addresses/{label}
// Update the label and/or value of an existing address.
func (api UsersAPI) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	oldlabel := mux.Vars(r)["label"]

	body := struct {
		Label   string
		Address Address
	}{}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if !isValidLabel(body.Label) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	userMgr := NewManager(r)

	u, err := userMgr.GetByName(username)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if _, ok := u.Address[oldlabel]; ok != true {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if oldlabel != body.Label {
		if _, ok := u.Address[body.Label]; ok {
			http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
			return
		}
	}

	if err = userMgr.SaveAddress(username, body.Label, body.Address); err != nil {
		log.Error("ERROR while saving address - ", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if oldlabel != body.Label {
		if err := userMgr.RemoveAddress(username, oldlabel); err != nil {
			log.Error(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(body)
}

// DeleteAddress is the handler for DELETE /users/{username}/addresses/{label}
// Removes an address
func (api UsersAPI) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	label := mux.Vars(r)["label"]
	userMgr := NewManager(r)

	u, err := userMgr.GetByName(username)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if _, ok := u.Address[label]; ok != true {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if err := userMgr.RemoveAddress(username, label); err != nil {
		log.Error("ERROR while saving address:\n", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Get the contracts where the user is 1 of the parties. Order descending by date.
// It is handler for GET /users/{username}/contracts
func (api UsersAPI) usernamecontractsGet(w http.ResponseWriter, r *http.Request) {
}

// Get a specific authorization
// It is handler for GET /users/{username}/scopes/{grantedTo}
func (api UsersAPI) usernamescopesgrantedToGet(w http.ResponseWriter, r *http.Request) {}

// Update a Scope
// It is handler for PUT /users/{username}/scopes/{grantedTo}
func (api UsersAPI) usernamescopesgrantedToPut(w http.ResponseWriter, r *http.Request) {}

// Remove a Scope, the granted organization will no longer have access the user's information.
// It is handler for DELETE /users/{username}/scopes/{grantedTo}
func (api UsersAPI) usernamescopesgrantedToDelete(w http.ResponseWriter, r *http.Request) {}

// Get the list of notifications, these are pending invitations or approvals
// It is handler for GET /users/{username}/notifications
func (api UsersAPI) usernamenotificationsGet(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]

	type NotificationList struct {
		Approvals        []invitations.JoinOrganizationInvitation `json:"approvals"`
		ContractRequests []contract.ContractSigningRequest        `json:"contractRequests"`
		Invitations      []invitations.JoinOrganizationInvitation `json:"invitations"`
	}
	var notifications NotificationList

	invititationMgr := invitations.NewInvitationManager(r)

	userOrgRequests, err := invititationMgr.GetByUser(username)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	notifications.Invitations = userOrgRequests

	// TODO: Get Approvals and Contract requests
	notifications.Approvals = []invitations.JoinOrganizationInvitation{}
	notifications.ContractRequests = []contract.ContractSigningRequest{}

	w.Header().Set("Content-type", "application/json")
	json.NewEncoder(w).Encode(&notifications)

}

// usernameorganizationsGet is the handler for GET /users/{username}/organizations
// Get the list organizations a user is owner of member of
func (api UsersAPI) usernameorganizationsGet(w http.ResponseWriter, r *http.Request) {}

// usernamescopesGet is the handler for GET /users/{username}/scopes
// Get the list of authorization scopes
func (api UsersAPI) usernamescopesGet(w http.ResponseWriter, r *http.Request) {}
