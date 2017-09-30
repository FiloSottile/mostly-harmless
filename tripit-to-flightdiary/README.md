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

## Syncing

Run `sync.py`, it will add missing flight (as identified by date and flight number) that are in TripIt to Flightdiary.
Additionally, you can run using the argument "-i <tripit json filename>" (e.g. `sync.py -i tripit.json`) to import the file generated when running `tripit.py > tripit.json`

## Dumping all TripIt information

Run `tripit.py`, it will output a JSON of all your past trips.

You can then run `./jq.sh < tripit.json > tripit.txt` to get a textual list of flights. (You need to install jq.)

## Dumping Flightdiary flights

Run `flightdiary.py`, it will output a textual list of all your flights.

You can compare this list with the one on TripIt (for example before running the sync) like this:

```
./flightdiary.py | sort | diff -u - tripit.txt
```

Any lines starting with `+` are the ones that `sync.py` will add (unless it's just the airports codes that are wrong).
