import web
import os

urls = (
    '/', 'index'
)

class index:
    def GET(self):
        return "Hello, world!"

app = web.application(urls, globals()).wsgifunc()
