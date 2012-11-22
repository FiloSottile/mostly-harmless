import urllib2
import web
import json
import os
import hmac, hashlib
from urllib import urlencode
from base64 import urlsafe_b64decode, urlsafe_b64encode

FB_APP_ID = os.environ.get('FACEBOOK_APP_ID')
FB_APP_SECRET = os.environ.get('FACEBOOK_SECRET')
FB_NAMESPACE = os.environ.get('FACEBOOK_NAMESPACE')
FBAPI_SCOPE = []

def get_home():
    return (web.ctx.env.get('wsgi.url_scheme') + "://"
            + web.ctx.env.get('HTTP_HOST') + '/')

def oauth_login_url():
    fb_login_uri = ("https://www.facebook.com/dialog/oauth"
                    "?client_id=%s&redirect_uri=%s" %
                    (FB_APP_ID, urllib2.quote('https://apps.facebook.com/' + FB_NAMESPACE + '/')))

    if FBAPI_SCOPE:
        fb_login_uri += "&scope=%s" % ",".join(FBAPI_SCOPE)
    return fb_login_uri

def b64decode(s):
    s = str(s)
    s = s.ljust((len(s)/4)*4+4, '=') if len(s) % 4 else s
    return urlsafe_b64decode(s)

def b64encode(d):
    return urlsafe_b64encode(d).rstrip('=')

def parse_signed_request(signed_request):
    encoded_sig, payload = signed_request.split('.', 1)
    data = json.loads(b64decode(payload))

    if not data['algorithm'].upper() == 'HMAC-SHA256':
        # TODO log
        return False

    h = hmac.new(FB_APP_SECRET, digestmod=hashlib.sha256)
    h.update(payload)
    expected_sig = b64encode(h.digest())
    if expected_sig != encoded_sig:
        # TODO log
        return False

    return data

def call(c, args=None):
    url = "https://graph.facebook.com/{0}".format(c)
    if args: params = '?' + urlencode(args)
    else: params = ''
    r = urllib2.urlopen(url + params)
    return json.loads(r.read())