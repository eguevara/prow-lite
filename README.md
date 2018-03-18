# Prow-lite

prow-lite is a clone of [kubernetes/prow](https://github.com/kubernetes/test-infra/tree/master/prow) which is used to manage ci/cd tooling around github webhook events. 

prow-light is a stripped down version of [prow](https://github.com/kubernetes/test-infra/tree/master/prow) for the purpose of understanding how the tool works under the hood.

The goal is to better understand Go code by digging into how the plugins are wired (loaded), how config files are loaded (and polled) and how to build a HTTP server with handlers for the webhooks.

For kicks, I also decided to use gitlab rather github.

Lets start with the actual http server that handles webhook requests from Gitlab.  

## HTTP Webhook Server
Gitlab is configured to push events on comments to an endpoint.  This endpoint is the http server that is started as part of the ```cmd/hook/main.go``` code. Nothing fancy here.  Just a typical http server in Go with one handler registered ```hook```.  




