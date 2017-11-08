import web

urls = ('/(.*)/', 'redirect', "/.*", "hello")
app = web.application(urls, globals())

class redirect:
    def GET(self, path):
        web.seeother('/' + path)

class hello:
    def GET(self):
        return 'Hello, world!\n'

if __name__ == "__main__":
    # web.config.debug = False
    app.run()
