# Deprecation Notice
This sample project has been deprecated. It is no longer being actively maintained and is probably out of date.

In other words, if you decide to clone this repository, you're on your own.

# Google Mirror API's Quickstart for Go

This project shows you how to implement a simple
piece of Glassware that demos the major functionality of the Google Mirror API.

To see a fully-working demo of the quick start project, go to
[https://glass-python-starter-demo.appspot.com/](https://glass-java-starter-demo.appspot.com).
Otherwise, read on to see how to deploy your own version.

## Prerequisites

[The App Engine SDK for Go](/appengine/downloads#Google_App_Engine_SDK_for_Go) -
The Go quick start project is implemented using App Engine. You need
the Go App Engine SDK to develop and deploy your project.
Run the installer if appropriate for your platform, or extract the zip file
in a convenient place.

## Configuring the project

Configure the Quick Start project to use your API client information:

<ol>
  <li>Enter your client ID and secret in <code>config.go</code>:
<pre class="prettyprint">// Created at http://code.google.com/apis/console, these identify
// our app for the OAuth protocol.
clientId     = "[[YOUR_CLIENT_ID]]"
clientSecret = "[[YOUR_CLIENT_SECRET]]"
</pre>
  </li>
  <li>Generate a session secret string and set it in <code>config.go</code>:
<pre class="prettyprint">secret      = "This should really be a secret." // Make it a random string
</pre>
  </li>
  <li>Edit <code>app.yaml</code> to enter your App Engine application ID:
<pre class="prettyprint">application: your_appengine_application_id
version: 1
runtime: go
api_version: go1
...</pre>
  </li>
</ol>

## Deploying the project

Press the blue <b>Deploy</b> button in the App Engine Launch GUI interface or run this shell
command to deploy your code:

    $ appcfg.py --oauth2 update .
