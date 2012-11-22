import web
import os
import gettext
from PIL import Image
import urllib2
import hmac, hashlib
from urlparse import parse_qsl
from cStringIO import StringIO

import fb

DEBUG = (os.environ.get('DEBUG') == '1')

curdir = os.path.abspath(os.path.dirname(__file__))
localedir = curdir + '/i18n'
gettext.install('messages', localedir, unicode=True)   
render = web.template.render(curdir + '/templates/', globals={'_': _})

def flip_url(url):
    h = hmac.new(fb.FB_APP_SECRET + 'flipurl', url, hashlib.sha256)
    sig = fb.b64encode(h.digest())
    return "/flip/{}.jpg?sig={}".format(fb.b64encode(url), sig)

urls = (
    '/', 'index',
    '/flip/(.+?)(\.jpg)?', 'flip',
)

class flip:
    def GET(self, url, ext):
        url = fb.b64decode(url)
        h = hmac.new(fb.FB_APP_SECRET + 'flipurl', url, hashlib.sha256)
        sig = fb.b64encode(h.digest())
        if not sig == dict(parse_qsl(web.ctx.query.lstrip('?'))).get('sig'):
            raise web.forbidden()

        fp = urllib2.urlopen(url)
        im = StringIO(fp.read())
        m = Image.open(im)
        m = m.transpose(Image.FLIP_LEFT_RIGHT)

        output = StringIO()
        m.save(output, format='jpeg')
        contents = output.getvalue()
        output.close()
        im.close()

        web.header("Content-Type", "images/jpeg")
        return contents

class index:
    def GET(self):
        raise web.forbidden()

    def POST(self):
        fb_data = fb.parse_signed_request(web.input()['signed_request'])
        if not fb_data:
            raise web.forbidden()

        gettext.translation('messages', localedir, languages=[fb_data['user']['locale'], 'en_US']).install(True)

        if not fb_data.get('user_id'):
            return "<script> top.location.href='" + fb.oauth_login_url() + "'</script>"

        cover = fb.call('fql', args={'q': "SELECT pic_cover FROM user WHERE uid='{}'".format(fb_data['user_id']),
                                     'access_token': fb_data['oauth_token']})['data'][0]['pic_cover']['source']

        flipped_cover = flip_url(cover)

        return render.index(fb_data, cover, flipped_cover, _)

application = web.application(urls, globals())
if DEBUG: application.internalerror = web.debugerror
app = application.wsgifunc()
