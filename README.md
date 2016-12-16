# Menéame Digest [Archived]
Feed digest server for [meneame.net](https://meneame.net), a Spanish news aggregator - host your own server or subscribe to [meneame-diario.appspot.com/rss](https://meneame-diario.appspot.com/rss) for daily top ten menéame news stories

Endpoints
---------

* **```/```**: basic web browser version of the most recent digest
* **```/rss```**: most recent digest in RSS format

Configuration
-------------

The configuration options are stored in ```mnmdigest/config.yaml```:

Parameter        | Description                          | Type   | Default value
---              | ---                                  | ---    | ---
**server_url**   | server endpoint                      | string | https://meneame-diario.appspot.com
**meneame_url**  | menéame endpoint                     | string | https://meneame.net
**refresh_rate** | Refresh rate for the digest, in days | uint   | 1
**max_articles** | Maximum articles per digest          | uint   | 10

Deployment in Google Cloud Platform
-----------------------------------

* Install the [Google App Engine for Go](https://cloud.google.com/appengine/downloads)
* Modify the application id setting the parameter ```application``` in ```app.yaml``` to your own application id
* Execute ```goapp deploy``` inside the root directory of the repository

Local Deployment
----------------

* Install [Cloud SDK](https://cloud.google.com/sdk/#Quick_Start)
* Modify the application id setting the parameter ```application``` in ```app.yaml``` to your own application id
* Execute ```goapp serve .``` inside the root directory of the repository

Overview
--------

* **Problem**: if subscribed directly to menéame, there is no option to set the newsfeed to only update every X days and only show the top Y stories over the these last X days, reducing the overall noise and constant updates
* **Proposed solution**: Menéame Digest retrieves the top X stories over the last Y days and displays them, setting your own pace
* **Limitations**: pasts digests are not stored, only the most recent one is - use a newsfeed's service (like [Feedly](https://feedly.com)) to subscribe to this digest and store past feeds

#### Algorithm

The server generates the digest on demand.
Once a request to any of the endpoints is made, the server checks if the last refresh made to the list of top stories was within the time limit set in the configuration parameter ```refresh_rate```.
If so, it just serves that cached page.
If not, it retrieves the new list of top stories from menéame, displaying (and storing) them.

In order to ensure that the new list of top stories is unique, a history of the past top lists has to be saved too.
Since we don't want this stored list to grow indefinitely, we need to infer, given that a story was retrieved at the update ```Y1```, how many updates we need so that the probability of the story showing again at the update ```Y2 = Y1+#Updates``` is close to zero. This is achieved using the story property ```updates_to_flush```.

A story has then the following properties:
* **id**: permalink in menéame (used as identifier)
* **url**: permalink of the story itself
* **title**: title
* **updates_to_flush**: decreasing counter to know when to flush this story from storage

So we store:
* **last_digest**: date of the last digest
* **page_html**: last generated HTML page
* **page_rss**: last generated RSS page
* **past_stories**: past stories, as described above

Finally, the algorithm pseudocode is:
```
New endpoint request
  If no lock and the time difference between last_digest and now is greater than refresh_date days
    Lock
      Retrieve a *long enough* news list from menéame, sorted by karma
      Filter out stories that are in the past_stories list
      Decrease updates_to_flush for every story in past_stories by one
      Remove the stories that have updates_to_flush equal to zero from past_stories
      Trim the filtered list so it contains at maximum max_articles stories
      Store the trimmed list of new stories in past_stories
      Generate page_html and page_rss using the trimmed list of new stories
    Unlock
  Serve page_html or page_rss, depending on the endpoint
```

License
-------
[MIT](LICENSE) - Feel free to use and edit.
This project is not affiliated, connected or associated to the official menéame project ([github.com/gallir/Meneame](https://github.com/gallir/Meneame)).
