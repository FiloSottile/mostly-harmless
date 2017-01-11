from requests_oauthlib import OAuth1Session
import json

with open('creds.json') as f:
    creds = json.load(f)

client_key = creds['CLIENT_KEY']
client_secret = creds['CLIENT_SECRET']
request_token_url = 'https://api.tripit.com/oauth/request_token'
oauth = OAuth1Session(client_key, client_secret=client_secret)
fetch_response = oauth.fetch_request_token(request_token_url)
print fetch_response
base_authorization_url = 'https://www.tripit.com/oauth/authorize'
authorization_url = oauth.authorization_url(base_authorization_url)
print authorization_url + '&oauth_callback=http%3A%2F%2Flocalhost%2Ffoo'
oauth_response = oauth.parse_authorization_response(raw_input())
print oauth_response
resource_owner_key = fetch_response.get('oauth_token')
resource_owner_secret = fetch_response.get('oauth_token_secret')
access_token_url = 'https://api.tripit.com/oauth/access_token'
oauth = OAuth1Session(client_key, client_secret=client_secret,
    resource_owner_key=resource_owner_key,
    resource_owner_secret=resource_owner_secret, verifier='foo')
oauth_tokens = oauth.fetch_access_token(access_token_url)
print oauth_tokens
