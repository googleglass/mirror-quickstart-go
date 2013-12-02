// Copyright (C) 2012 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package quickstart

import (
	"net/http"
	"net/url"
	"strings"
	"time"
	"code.google.com/p/goauth2/oauth"
	"github.com/gorilla/sessions"

	"appengine"
	"appengine/datastore"
	"appengine/urlfetch"
)

// Cookie store used to store the user's ID in the current session.
var store = sessions.NewCookieStore([]byte(secret))

type SimpleToken struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time         // If zero the token has no (known) expiry time.
}

// OAuth2.0 configuration variables.
func config(host string) *oauth.Config {
	r := &oauth.Config{
		ClientId:       clientId,
		ClientSecret:   clientSecret,
		Scope:          scopes,
		AuthURL:        "https://accounts.google.com/o/oauth2/auth",
		TokenURL:       "https://accounts.google.com/o/oauth2/token",
		AccessType:     "offline",
		ApprovalPrompt: "force",
	}
	if len(host) > 0 {
		r.RedirectURL = fullURL(host, "/oauth2callback")
	}
	return r
}

// fullURL returns the full URL using the provided host and path.
func fullURL(host, path string) string {
	url := &url.URL{Scheme: "https", Host: host, Path: path}
	if !strings.Contains(host, "appspot.com") {
		url.Scheme = "http"
	}
	return url.String()
}

// storeUserID stores the current user's ID in the session's coookies.
func storeUserID(w http.ResponseWriter, r *http.Request, userId string) error {
	session, err := store.Get(r, sessionName)
	if err != nil {
		return err
	}
	session.Values["userId"] = userId
	return session.Save(r, w)
}

// userID retrieves the current user's ID from the session's cookies.
func userID(r *http.Request) (string, error) {
	session, err := store.Get(r, sessionName)
	if err != nil {
		return "", err
	}
	userId := session.Values["userId"]
	if userId != nil {
		return userId.(string), nil
	}
	return "", nil
}

// storeCredential stores the user's credentials in the datastore.
func storeCredential(c appengine.Context, userID string, token *oauth.Token) error {

    simple := new(SimpleToken)
    simple.AccessToken = token.AccessToken
    simple.RefreshToken = token.RefreshToken
    simple.Expiry = token.Expiry

	// Store the tokens in the datastore.
	key := datastore.NewKey(c, "OAuth2Token", userID, 0, nil)
	_, err := datastore.Put(c, key, simple)

	return err
}

// authTransport loads credential for user from the datastore.
func authTransport(c appengine.Context, userID string) *oauth.Transport {
	key := datastore.NewKey(c, "OAuth2Token", userID, 0, nil)
	tok := new(oauth.Token)
	if err := datastore.Get(c, key, tok); err != nil {
		c.Errorf("Get Token: %v", err)
		return nil
	}
	return &oauth.Transport{
		Config:    config(""),
		Token:     tok,
		Transport: &urlfetch.Transport{Context: c},
	}
}

// deleteCredential deletes credential for user from the datastore.
func deleteCredential(c appengine.Context, userId string) error {
	key := datastore.NewKey(c, "OAuth2Token", userId, 0, nil)
	return datastore.Delete(c, key)
}

// errorAdapter executes the HTTP handler and catch the returned error.
func errorAdapter(f func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := appengine.NewContext(r)
		err := f(w, r)
		if err != nil {
			c.Errorf("Handler returned an error: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
