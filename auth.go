// Copyright (C) 2013 Google Inc.
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
	"fmt"
	"net/http"
	"strings"

	"code.google.com/p/goauth2/oauth"
	"code.google.com/p/google-api-go-client/mirror/v1"
	"code.google.com/p/google-api-go-client/oauth2/v2"

	"appengine"
	"appengine/urlfetch"
)

const revokeEndpointFmt = "https://accounts.google.com/o/oauth2/revoke?token=%s"

// Because App Engine owns main and starts the HTTP service,
// we do our setup during initialization.
func init() {
	http.HandleFunc("/auth", authHandler)
	http.HandleFunc("/oauth2callback", errorAdapter(oauth2callbackHandler))
	http.HandleFunc("/signout", errorAdapter(signoutHandler))
}

// auth is the HTTP handler that redirects the user to authenticate
// with OAuth.
func authHandler(w http.ResponseWriter, r *http.Request) {
	url := config(r.Host).AuthCodeURL(r.URL.RawQuery)
	http.Redirect(w, r, url, http.StatusFound)
}

// oauth2callback is the handler to which Google's OAuth service redirects the
// user after they have granted the appropriate permissions.
func oauth2callbackHandler(w http.ResponseWriter, r *http.Request) error {
	c := appengine.NewContext(r)

	// Create an oauth transport with a urlfetch.Transport embedded inside.
	t := &oauth.Transport{
		Config:    config(r.Host),
		Transport: &urlfetch.Transport{Context: c},
	}

	// Exchange the code for access and refresh tokens.
	tok, err := t.Exchange(r.FormValue("code"))
	if err != nil {
		return fmt.Errorf("Exchange(%q): %v", r.FormValue("code"), err)
	}

	o, err := oauth2.New(t.Client())
	if err != nil {
		return fmt.Errorf("Unable to instantiate UserInfo service: %s", err)
	}
	u, err := o.Userinfo.Get().Do()
	if err != nil {
		return fmt.Errorf("Unable to retrieve user ID: %s", err)
	}

	userId := fmt.Sprintf("%s_%s", strings.Split(clientId, ".")[0], u.Id)

	if err = storeUserID(w, r, userId); err != nil {
		return fmt.Errorf("Unable to store user ID: %s", err)
	}

	if err = storeCredential(c, userId, tok); err != nil {
		return fmt.Errorf("Unable to store user ID: %s", err)
	}

	bootstrapUser(r, t.Client(), userId)
	http.Redirect(w, r, "/", http.StatusFound)
	return nil
}

// bootstrapUser sets up sharing contact and notificaiton for a new user.
func bootstrapUser(r *http.Request, client *http.Client, userId string) {
	c := appengine.NewContext(r)
	m, _ := mirror.New(client)

	if strings.HasPrefix(r.Host, "https://") {
		s := &mirror.Subscription{
			Collection:  "timeline",
			UserToken:   userId,
			CallbackUrl: fullURL(r.Host, "/notify"),
		}
		m.Subscriptions.Insert(s).Do()

		c := &mirror.Contact{
			Id:          "Go_Quick_Start",
			DisplayName: "Go Quick Start",
			ImageUrls:   []string{fullURL(r.Host, "/static/images/gopher.png")},
		}
		m.Contacts.Insert(c).Do()
	} else {
		c.Infof("Post auth tasks are not supported on staging.")
	}

	t := &mirror.TimelineItem{
		Text:         "Welcome to the Go Quick Start",
		Notification: &mirror.NotificationConfig{Level: "DEFAULT"},
	}

	m.Timeline.Insert(t).Do()
}

// signout Revokes access for the user and removes the associated credentials from the datastore.
func signoutHandler(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		return nil
	}
	c := appengine.NewContext(r)
	userId, err := userID(r)
	if err != nil {
		return fmt.Errorf("Unable to retrieve user ID: %s", err)
	}
	if userId == "" {
		http.Redirect(w, r, "/auth", http.StatusFound)
		return nil
	}
	t := authTransport(c, userId)
	if t == nil {
		http.Redirect(w, r, "/auth", http.StatusFound)
		return nil
	}
	client := urlfetch.Client(c)
	_, err = client.Get(fmt.Sprintf(revokeEndpointFmt, t.Token.RefreshToken))
	if err != nil {
		return fmt.Errorf("Unable to revoke token: %s", err)
	}
	storeUserID(w, r, "")
	deleteCredential(c, userId)

	http.Redirect(w, r, "/", http.StatusFound)
	return nil
}
