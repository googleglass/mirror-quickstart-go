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

/*
Package quickstart provides examples to quickly get started on the Google
Mirror API with Go on Google App Engine.

The main entry points are:
  * main.go: Displays the main page and handles requests from the main UI; this
             where most of the Mirror API logic is implemented.
  * auth.go: Handles authentication and log-out though OAuth 2.0
  * notify.go: Handles push notifications from the Mirror API.
  * attachment.go: Proxies requests from the main page to retrieve media
                   attachments for the current user.
*/
package quickstart
