# Menéame-diario
Daily digest for meneame.net, a spanish news aggregator.
Host your own server or subscribe to https://meneame-diario.appspot.com/rss.

Endpoints
---------

* **```/```**: basic web browser version of the most recent digest
* **```/rss```**: most recent digest in RSS format

Configuration
-------------

The configuration options are stored in ```config.yaml```:

Parameter | Description | Type | Default value
--- | --- | --- | ---
**meneame_url** | menéame endpoint | string | https://meneame.net
**refresh_rate** | Refresh rate for the digest, in hours | uint | 24
**max_articles** | Maximum articles per digest | uint | 10

Deployment in Google Cloud Platform
-----------------------------------

* Modify the application id setting the parameter ```application``` in ```app.yaml```.
* Execute ```goapp deploy .``` inside the root directory.

Algorithm
---------
TODO: Explanation

Limitations
-----------
TODO: Not archived

License
-------
[MIT](LICENSE) - Feel free to use and edit.
This project is not affiliated, connected or associated with the official Menéame project (https://github.com/gallir/Meneame).
