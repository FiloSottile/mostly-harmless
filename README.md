Set of scripts to export flights from TripIt and import them into Flightdiary. Quick and dirty.

## TripIt setup

* Register an application at https://www.tripit.com/developer
* Get API Key and Secret
* Save them to a file named `creds.json`

```json
{
    "CLIENT_KEY": "*** API Key ***",
    "CLIENT_SECRET": "*** API Secret ***"
}
```

* Run `get_token.py`
* Click on the `/authorize` URL and follow the instructions
* **You will land at a "Page Not Found", that's normal**
* Copy the URL, which looks like `https://www.tripit.com/oauth/foo?...` back into the terminal
* Put the tokens from the last line in `creds.json`

```json
    "OAUTH_TOKEN": "*** oauth_token ***",
    "OAUTH_TOKEN_SECRET": "*** oauth_token_secret ***"
```

## Flightdiary setup

Just put username and password in `creds.json`

```json
    "FD_USER": "*** USER ***",
    "FD_PASSWORD": "*** PASSWORD ***"
```
