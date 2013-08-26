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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"code.google.com/p/goauth2/oauth"
	"code.google.com/p/google-api-go-client/mirror/v1"

	"appengine"
	"appengine/taskqueue"
)

// Because App Engine owns main and starts the HTTP service,
// we do our setup during initialization.
func init() {
	http.HandleFunc("/notify", errorAdapter(notifyHandler))
	http.HandleFunc("/processnotification", notifyProcessorHandler)
}

// notifyHandler starts a new Task Queue to process the notification ping.
func notifyHandler(w http.ResponseWriter, r *http.Request) error {
	c := appengine.NewContext(r)
	t := &taskqueue.Task{
		Path:   "/processnotification",
		Method: "POST",
		Header: r.Header,
	}
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("Unable to read request body: %s", err)
	}
	t.Payload = payload
	// Insert a new Task in the default Task Queue.
	if _, err = taskqueue.Add(c, t, ""); err != nil {
		return fmt.Errorf("Failed to add new task: %s", err)
	}
	return nil
}

// notifyProcessorHandler processes notification pings from the API in a Task Queue.
func notifyProcessorHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	not := new(mirror.Notification)
	if err := json.NewDecoder(r.Body).Decode(not); err != nil {
		c.Errorf("Unable to decode notification: %v", err)
		return
	}
	userId := not.UserToken
	t := authTransport(c, userId)
	if t == nil {
		c.Errorf("Unknown user ID: %s", userId)
		return
	}
	svc, _ := mirror.New(t.Client())

	var err error
	if not.Collection == "locations" {
		err = handleLocationsNotification(c, svc, not)
	} else if not.Collection == "timeline" {
		err = handleTimelineNotification(c, svc, not, t)
	}
	if err != nil {
		c.Errorf("Error occured while processing notification: %s", err)
	}
}

// handleLocationsNotification processes a location notification.
func handleLocationsNotification(c appengine.Context, svc *mirror.Service, not *mirror.Notification) error {
	l, err := svc.Locations.Get(not.ItemId).Do()
	if err != nil {
		return fmt.Errorf("Unable to retrieve location: %s", err)
	}
	text := fmt.Sprintf("Go Quick Start says you are at %f by %f.", l.Latitude, l.Longitude)
	t := &mirror.TimelineItem{
		Text:         text,
		Location:     l,
		MenuItems:    []*mirror.MenuItem{&mirror.MenuItem{Action: "NAVIGATE"}},
		Notification: &mirror.NotificationConfig{Level: "DEFAULT"},
	}
	_, err = svc.Timeline.Insert(t).Do()
	if err != nil {
		return fmt.Errorf("Unable to insert timeline item: %s", err)
	}
	return nil
}

// handleTimelineNotification processes a timeline notification.
func handleTimelineNotification(c appengine.Context, svc *mirror.Service, not *mirror.Notification, transport *oauth.Transport) error {
	for _, ua := range not.UserActions {
		if ua.Type != "SHARE" {
			c.Infof("I don't know what to do with this notification: %+v", ua)
			continue
		}
		t, err := svc.Timeline.Get(not.ItemId).Do()
		if err != nil {
			return fmt.Errorf("Unable to retrieve timeline item: %s", err)
		}
		// We could have just updated the Text attribute in-place and used the
		// Update method instead, but we wanted to illustrate the Patch method
		// here.
		patch := &mirror.TimelineItem{
			Text: fmt.Sprintf("Go Quick Start got your photo! %s", t.Text),
		}
		_, err = svc.Timeline.Patch(not.ItemId, patch).Do()
		if err != nil {
			return fmt.Errorf("Unable to patch timeline item: %s", err)
		}
	}
	return nil
}
