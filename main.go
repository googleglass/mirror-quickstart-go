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
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"code.google.com/p/goauth2/oauth"
	"code.google.com/p/google-api-go-client/googleapi"
	"code.google.com/p/google-api-go-client/mirror/v1"

	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	"appengine/urlfetch"
)

type uiTemplateData struct {
	Message                    string
	TimelineItems              []*mirror.TimelineItem
	Contact                    *mirror.Contact
	TimelineSubscriptionExists bool
	LocationSubscriptionExists bool
}

// Main template.

var rootTmpl = template.Must(template.New("index.html").
	Funcs(template.FuncMap{"HasPrefix": strings.HasPrefix}).
	ParseFiles("index.html"))

// Map of operations to functions.
var operations = map[string]func(*http.Request, *mirror.Service) string{
	"insertSubscription":   insertSubscription,
	"deleteSubscription":   deleteSubscription,
	"insertItem":           insertItem,
	"insertItemWithAction": insertItemWithAction,
	"insertItemAllUsers":   insertItemAllUsers,
	"insertContact":        insertContact,
	"deleteContact":        deleteContact,
	"deleteTimelineItem":   deleteTimelineItem,
}

// Because App Engine owns main and starts the HTTP service,
// we do our setup during initialization.
func init() {
	http.HandleFunc("/", errorAdapter(rootHandler))
}

// root is the main handler.
func rootHandler(w http.ResponseWriter, r *http.Request) error {
	if r.URL.Path != "/" { // Only supports request on "/", ignore the rest.
		http.Error(w, "", http.StatusNotFound)
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
	if err = t.Refresh(); err != nil { // Check for valid credentials.
		http.Redirect(w, r, "/auth", http.StatusFound)
		return nil
	}
	svc, err := mirror.New(t.Client())
	if err != nil {
		return fmt.Errorf("Unable to create Mirror service: %s", err)
	}

	if r.Method == "POST" {
		op := r.FormValue("operation")
		msg := fmt.Sprintf("I don't know how to %s", op)
		if o, ok := operations[op]; ok {
			msg = o(r, svc)
		}
		m := &memcache.Item{
			Key:        userId,
			Value:      []byte(msg),
			Expiration: 5 * time.Second,
		}
		if err := memcache.Set(c, m); err != nil {
			c.Errorf("Unable to store message: %v", err)
		}

		http.Redirect(w, r, "/", http.StatusFound)
		return nil
	}

	timelineItems, err := svc.Timeline.List().MaxResults(3).Do()
	if err != nil {
		return err
	}
	contact, err := svc.Contacts.Get("Go_Quick_Start").Do()
	if err != nil { // 404 should not be considered an error.
		if error, ok := err.(*googleapi.Error); !ok || error.Code != http.StatusNotFound {
			return err
		}
	}
	subscriptions, err := svc.Subscriptions.List().Do()
	if err != nil {
		return err
	}

	message := ""
	if m, err := memcache.Get(c, userId); err == nil {
		message = string(m.Value)
		memcache.Delete(c, userId)
	}

	tData := uiTemplateData{
		Message:       message,
		TimelineItems: timelineItems.Items,
		Contact:       contact,
	}
	for _, s := range subscriptions.Items {
		if s.Collection == "timeline" {
			tData.TimelineSubscriptionExists = true
		} else if s.Collection == "locations" {
			tData.LocationSubscriptionExists = true
		}
	}

	return rootTmpl.Execute(w, tData)
}

// insertSubscription subscribes the app to notifications for the current user.
func insertSubscription(r *http.Request, svc *mirror.Service) string {
	collection := r.FormValue("collection")
	if collection == "" {
		collection = "timeline"
	}
	userToken, err := userID(r)
	if err != nil {
		return fmt.Sprintf("Unable to retrieve user ID: %s", err)
	}
	body := mirror.Subscription{
		Collection:  collection,
		UserToken:   userToken,
		CallbackUrl: fullURL(r.Host, "/notify"),
	}

	if _, err = svc.Subscriptions.Insert(&body).Do(); err != nil {
		return fmt.Sprintf("Unable to subscribe: %s", err)
	}
	return "Application is now subscribed to updates."
}

// deleteSubscription unsubscribes the app from notifications for the current
// user.
func deleteSubscription(r *http.Request, svc *mirror.Service) string {
	collection := r.FormValue("subscriptionId")

	if err := svc.Subscriptions.Delete(collection).Do(); err != nil {
		return fmt.Sprintf("Unable to unsubscribe: %s", err)
	}
	return "Application has been unsubscribed."
}

// insertItem inserts a Timeline Item in the user's Timeline.
func insertItem(r *http.Request, svc *mirror.Service) string {
	c := appengine.NewContext(r)
	c.Infof("Inserting Timeline Item")

	body := mirror.TimelineItem{
		Notification: &mirror.NotificationConfig{Level: "AUDIO_ONLY"},
	}
	if r.FormValue("html") == "on" {
		body.Html = r.FormValue("message")
	} else {
		body.Text = r.FormValue("message")
	}

	var media io.Reader = nil
	mediaLink := r.FormValue("imageUrl")
	if mediaLink != "" {
		if strings.HasPrefix(mediaLink, "/") {
			mediaLink = fullURL(r.Host, mediaLink)
		}
		c.Infof("Downloading media from: %s", mediaLink)
		client := urlfetch.Client(c)
		if resp, err := client.Get(mediaLink); err != nil {
			c.Errorf("Unable to retrieve media: %s", err)
		} else {
			defer resp.Body.Close()
			media = resp.Body
		}
	}

	if _, err := svc.Timeline.Insert(&body).Media(media).Do(); err != nil {
		return fmt.Sprintf("Unable to insert timeline item: %s", err)
	}
	return "A timeline item has been inserted."
}

// insertItemWithAction inserts a Timeline Item that the user can reply to.
func insertItemWithAction(r *http.Request, svc *mirror.Service) string {
	c := appengine.NewContext(r)
	c.Infof("Inserting Timeline Item")

	body := mirror.TimelineItem{
		Creator:      &mirror.Contact{DisplayName: "Go Quick Start"},
		Text:         "Tell me what you had for lunch :)",
		Notification: &mirror.NotificationConfig{Level: "AUDIO_ONLY"},
		MenuItems:    []*mirror.MenuItem{&mirror.MenuItem{Action: "REPLY"}},
	}

	if _, err := svc.Timeline.Insert(&body).Do(); err != nil {
		return fmt.Sprintf("Unable to insert timeline item: %s", err)
	}
	return "A timeline item with action has been inserted."
}

// insertItemAllUsers inserts a Timeline Item to all authorized users.
func insertItemAllUsers(r *http.Request, svc *mirror.Service) string {
	c := appengine.NewContext(r)
	c.Infof("Inserting timeline item to all users")

	q := datastore.NewQuery("OAuth2Token")
	count, err := q.Count(c)
	if err != nil {
		return fmt.Sprintf("Unable to fetch users: %s", err)
	}
	if count > 5 {
		return fmt.Sprintf("Total user count is %d. Aborting broadcast to save your quota", count)
	}
	body := mirror.TimelineItem{
		Text:         "Hello Everyone!",
		Notification: &mirror.NotificationConfig{Level: "AUDIO_ONLY"},
	}
	i := q.Run(c)
	tok := new(oauth.Token)
	creds := &oauth.Transport{
		Config:    config(""),
		Token:     tok,
		Transport: &urlfetch.Transport{Context: c},
	}
	svc, _ = mirror.New(creds.Client())
	failed := 0

	for _, err = i.Next(tok); err == nil; _, err = i.Next(tok) {
		if _, err := svc.Timeline.Insert(&body).Do(); err != nil {
			c.Errorf("Failed to insert timeline item: %s", err)
			failed += 1
		}
	}
	return fmt.Sprintf("Sent cards to %d (%d failed).", count, failed)
}

// insertContact inserts a contact.
func insertContact(r *http.Request, svc *mirror.Service) string {
	c := appengine.NewContext(r)
	c.Infof("Inserting contact")
	name := r.FormValue("name")
	imageUrl := r.FormValue("imageUrl")
	if name == "" || imageUrl == "" {
		return "Must specify imageUrl and name to insert contact"
	}
	if strings.HasPrefix(imageUrl, "/") {
		imageUrl = fullURL(r.Host, imageUrl)
	}

	body := mirror.Contact{
		DisplayName: name,
		Id:          strings.Replace(name, " ", "_", -1),
		ImageUrls:   []string{imageUrl},
	}

	if _, err := svc.Contacts.Insert(&body).Do(); err != nil {
		return fmt.Sprintf("Unable to insert contact: %s", err)
	}
	return fmt.Sprintf("Inserted contact: %s", name)
}

// deleteContact deletes an existing contact.
func deleteContact(r *http.Request, svc *mirror.Service) string {
	id := strings.Replace(r.FormValue("id"), " ", "_", -1)

	if err := svc.Contacts.Delete(id).Do(); err != nil {
		return fmt.Sprintf("Unable to delete contact: %s", err)
	}
	return "Contact has been deleted."
}

// deleteTimelineItem deletes a timeline item.
func deleteTimelineItem(r *http.Request, svc *mirror.Service) string {
	itemId := r.FormValue("itemId")
	err := svc.Timeline.Delete(itemId).Do()
	if err != nil {
		return fmt.Sprintf("An error occurred: %v\n", err)
	}
	return "A timeline item has been deleted."
}
