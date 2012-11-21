import web
import os
import gettext

import fb

DEBUG = (os.environ.get('DEBUG') == '1')

curdir = os.path.abspath(os.path.dirname(__file__))
localedir = curdir + '/i18n'
gettext.install('messages', localedir, unicode=True)   
render = web.template.render(curdir + '/templates/', globals={'_': _})


urls = (
    '/', 'index'
)

class index:
    def GET(self):
        return web.forbidden()

    def POST(self):
        fb_data = fb.parse_signed_request(web.input()['signed_request'])
        if not fb_data:
            return web.forbidden()

        gettext.translation('messages', localedir, languages=[fb_data['user']['locale'], 'en_US']).install(True)

        if not fb_data.get('user_id'):
            return "<script> top.location.href='" + fb.oauth_login_url() + "'</script>"

        return render.index(fb_data['user_id'], _)

application = web.application(urls, globals())
if DEBUG: application.internalerror = web.debugerror
app = application.wsgifunc()
