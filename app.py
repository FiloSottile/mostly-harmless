import web
import os

import fb

urls = (
    '/', 'index'
)

class index:
    def POST(self):
        fb_data = fb.parse_signed_request(web.input()['signed_request'])

        if not fb_data.get('user_id'):
            return "<script> top.location.href='" + fb.oauth_login_url() + "'</script>"

        return fb_data.get('user_id')

app = web.application(urls, globals()).wsgifunc()
