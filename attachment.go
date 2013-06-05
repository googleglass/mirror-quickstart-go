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
	"io"
	"net/http"

	"code.google.com/p/google-api-go-client/mirror/v1"

	"appengine"
)

// Init HTTP handlers.
func init() {
	http.HandleFunc("/attachmentproxy", errorAdapter(attachmentProxyHandler))
}

// attachmentProxy returns the attachment for the current user using the IDs provided by the "timelineItem" and "attachment" form values.
func attachmentProxyHandler(w http.ResponseWriter, r *http.Request) error {
	itemId := r.FormValue("timelineItem")
	attachmentId := r.FormValue("attachment")
	c := appengine.NewContext(r)
	userId, err := userID(r)
	if err != nil {
		return err
	}
	t := authTransport(c, userId)
	svc, err := mirror.New(t.Client())
	if err != nil {
		return err
	}
	if itemId == "" || attachmentId == "" {
		http.Error(w, "", http.StatusBadRequest)
		return nil
	}
	a, err := svc.Timeline.Attachments.Get(itemId, attachmentId).Do()
	if err != nil {
		return err
	}
	req, err := http.NewRequest("GET", a.ContentUrl, nil)
	if err != nil {
		return err
	}

	resp, err := t.RoundTrip(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	io.Copy(w, resp.Body)
	return nil
}
