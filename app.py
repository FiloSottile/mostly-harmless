import web
import os

urls = (
    '/', 'index'
)

class index:
    def GET(self):
        return "Hello, world!"

    def POST(self):
        return "Hello, POST!"

app = web.application(urls, globals()).wsgifunc()
