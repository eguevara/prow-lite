# Prow-lite

prow-lite is a clone of [kubernetes/prow](https://github.com/kubernetes/test-infra/tree/master/prow) which is used to manage ci/cd tooling around github webhook events. 

prow-light is a stripped down version of [prow](https://github.com/kubernetes/test-infra/tree/master/prow) for the purpose of understanding how the tool works under the hood.  It only has code to support the ```/lgtm``` plugin and is strictly for self-learning Go code.

I was curious on understanding how prow works.  This repo helped me better understand the mechanics of the code by walking through each piece of code to get a single plugin working.  I was interested in understanding how the plugins are wired (loaded), how config files are loaded (and polled), testing is done and how to build a HTTP server with handlers for the webhooks.

For kicks, I also decided to use gitlab rather github.

Lets start with the actual http server that handles webhook requests from Gitlab.  

## HTTP Webhook Server
Gitlab is configured to push events on comments to an endpoint.  This endpoint is the http server that is started as part of the ```cmd/hook/main.go``` code. Nothing fancy here.  Just a typical http server in Go with one handler registered ```hook```.  To start the http server run 

```
cd cmd/hook
go run main.go
``` 

You should see some logs indicating where the server is listening to and some pid info (in json format brought you by the logrus pkg).  

At this point we have http server up and running listening (by default) on localhost port 3000.  Gorilla mux sets up the router, registering the path ```/hook``` to the http handler defined by ```api.HooksHandler``` only accepting the POST http method.  To validate, open a separate terminal and run:
```
curl \
  -X POST --fail \
  -H 'Content-Type: application/json' \
  -H 'X-Gitlab-Event: Note Hook' \
  "http://0.0.0.0:3000/hook" \
  -d @webhook-comment-request.data
```
Should get a 404 if you try to do a GET and should return an error if you do not pass in the X-Gitlab-Event headers.  

Lets now dig into the ```api.HooksHandler``` which implements ServeHTTP.

## HTTP Handler
The handler is where we process the http POST request submitted by the gitlab server (webhook).  The ```ValidateWebhook``` does some simple error checking on the reqeust ensuring that certain headers are present. It then reads the payload and passing it along to be processed by the ```h.processEvent``` function.  

Here is where it gets interesting...

```processEvent``` based on the ```eventType``` will covert the payload (bytes) into a Go value (&mc).  This esentially takes the json payload from the POST request and converts it into a Go struct (gitlab.MergeCommentEvent).  If all good, it calls the ```handleMergeCommentEvent``` passing in a logrus.Entry and go value with all of the fields from the payload. 

```handleMergeCommentEvent``` does the magic of calling the plugin's handler to perform the /lgtm code.  But... how does this all wire up to call the ```handleMergeComment``` in the ```plugins/lgtm/lgtm.go``` file? 

```MergeCommentEventHandlers``` returns a map[string]MergeCommentEventHandler.  The key being the string plugin name (ie "lgtm") and the value of type MergeCommentEventHandler.  ie.  ```map["lgtm"] = MergeCommentEventHandler```.  Notice that the type of MergeCommentEventHandler is a function ```func(PluginClient, gitlab.MergeCommentEvent) error```. 

You can read this as the ```MergeCommentEventHandler``` type is a function that takes two parameters (PluginClient and gitlab.MergeCommmentEvent) and retuns an error. So this means that the value of the map can be any function that meets that signature (same number of parameters and result types)

The code iterates through the list of [plugins](https://github.com/eguevara/prow-lite/blob/642465e4f1dcc58e96d2be98a9b0955d574bf682/plugins/plugins.go#L111) that are enabled for the repo (defined by the plugins.yaml).  Then checks whether the key (the plugin name :lgtm) exists in the ```mergeCommentEventHandler``` map. 

We will get to how handlers are registered in a bit.. but lets continue with the handler. 

Assuming that it does find the handler by the plugin name "lgtm", the code then calls (as a goroutine) an anonymous function passing in the plugin key ("lgmt") and the handler (type function).  ```handler``` is the registered function for the "lgtm" plugin.  This takes us to the actual "lgtm" [code](https://github.com/eguevara/prow-lite/blob/642465e4f1dcc58e96d2be98a9b0955d574bf682/plugins/lgtm/lgtm.go#L19).  Notice that the ```handleMergeComment``` function has the same signature needed by the MergeCommentEventHandler.  So, this function ```handleMergeComment``` is the value of the map["lgtm"] key.  From here we are in the ```lgtm.go``` code where the code uses the pluginClient to access the gitlabclient field to call the Methods needed to [add labels](https://github.com/eguevara/prow-lite/blob/642465e4f1dcc58e96d2be98a9b0955d574bf682/plugins/lgtm/lgtm.go#L40).  

And there you have it.  We get a POST request handled by our handler in ```cmd/hook/main.go```.  The handler is process by its http Handler in ```api/hook.go```.  The handler, based on the request header type, then finds the plugin handler and calls the function (the function value assigned to the plugin key).  This then calls the ```plugins/lgtm/lgtm.go``` function to process the lgtm logic to add the add the labels and returns.

The piece thats missing is how the plugin handlers are registered. Notice the ```init()``` [function](https://github.com/eguevara/prow-lite/blob/642465e4f1dcc58e96d2be98a9b0955d574bf682/plugins/lgtm/lgtm.go#L16) which calls the ```RegisterMergeCommentEventHandler``` function.  Because its the init() it will be called immediately as the lgtm package is loaded.  So when someone imports ```"github.com/eguevara/prow-lite/plugins/lgtm"``` init() is called [registering](https://github.com/eguevara/prow-lite/blob/642465e4f1dcc58e96d2be98a9b0955d574bf682/plugins/plugins.go#L25) ```handleMergeComment``` to the "lgtm" key.  

Now taking a closer look at ```/cmd/hook/main.go```, you'll notice that it imports the ```"github.com/eguevara/prow-lite/api"``` pkg.  Importing this pkg also loads a [blank import](https://github.com/eguevara/prow-lite/blob/642465e4f1dcc58e96d2be98a9b0955d574bf682/api/plugins.go#L6) ```api/plugins.go```.  This import ``plugins/lgtm``` is responsible for calling the init() and registering the "lgtm" plugin on load.

```
mergeCommentEventHandlers["lgtm"] =  func handleMergeComment(pc plugins.PluginClient, e gitlab.MergeCommentEvent) error
```

## Misc
Pretty awesome how yaml files are read and used to drive dynamic configurations.  The code loads the file and loops through a timer looking for new changes.  This means that you can make changes ie change the log_level without have to restart the web service.  Cool!!  This is used to load the [config.yaml](https://github.com/eguevara/prow-lite/blob/642465e4f1dcc58e96d2be98a9b0955d574bf682/config/config.go#L85) and [plugins.yaml](https://github.com/eguevara/prow-lite/blob/642465e4f1dcc58e96d2be98a9b0955d574bf682/plugins/plugins.go#L86).  

Also cool, how the "lgtm" plugin uses a [githubclient](https://github.com/kubernetes/test-infra/blob/3d8be4be12a6a840838d1ab9a617a57566a6afc7/prow/plugins/lgtm/lgtm.go#L61) interface to allow easy testing.  You can easily create a fake struct that satisfies all of the methods and use that as the [githubclient](https://github.com/kubernetes/test-infra/blob/3d8be4be12a6a840838d1ab9a617a57566a6afc7/prow/plugins/lgtm/lgtm_test.go#L169) passed in to handle.