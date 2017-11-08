import urllib2
import re
import time
import json
import traceback

from geopy import geocoders  
g = geocoders.GoogleV3()

from bs4 import BeautifulSoup

def parse_listing(listing):
    res = {
            'name': None,
            'email': None,
            'fingerprint': None,
            'key': None,
            'keytype': None,
            'url': None,
            'key_comment': None,
            'notes': None
          }
    for li in listing.find_all("li", recursive=False):
        if li.get_text().startswith('Event name:'):
            res['name'] = li.a.string
        elif li.get_text().startswith('Name:'):
            res['name'] = li.a.string
        elif li.get_text().startswith("Email:"):
            who = li.b.string.rsplit(" ", 1)[0]
            at = li.strong.string.strip()
            res['email'] = str(len(at.split('.')[0])) + ',' + str(len(at)) + ',' + at.replace('.', '', 1) + who
        elif li.get_text().startswith("Fingerprint:"):
            res['fingerprint'] = li.a.get_text()
            if li.a.b:
                res['key'] = li.a.b.string
                res['keytype'] = re.search(r"\(([^\)]+)\)", li.get_text()).group(1)
        elif li.get_text().startswith("Key ID:"):
            res['fingerprint'] += li.get_text().split(':')[-1].strip()
            res['key'] = li.b.string
        elif li.get_text().startswith("URL:"):
            res['url'] = li.a.get("href")
        elif li.get_text().startswith("Key comment:"):
            res['key_comment'] = li.tt.string
        elif li.get_text().startswith("Notes:"):
            res['notes'] = li.tt.get_text().replace('\r\n', '{br}').replace('\n', ' ').replace('{br}', '\n')
    return res

# TODO Key ID field like here http://biglumber.com/x/web?sn=Enrico+Franceschi

index_page = urllib2.urlopen('http://biglumber.com/x/web?va=1').read().decode('iso-8859-1')
index_page = index_page[index_page.index('<li>'):]

locations_list = re.findall(r'http://biglumber\.com/x/web\?sl=(\d+)', index_page)

result = {}

for location in locations_list:
    try:
        city_listings_page = BeautifulSoup(urllib2.urlopen('http://biglumber.com/x/web?sl=' + location))
        location_name = re.search(r'Biglumber listings for (.+)', city_listings_page.find("h1").string).group(1)
        place, (lat, lng) = g.geocode(location_name.encode('utf8'), exactly_one=False)[0]
        listings = city_listings_page.html.body.find_all("ul", recursive=False)
        parsed_listings = []
        for l in listings:
            try:
                parsed_listings.append(parse_listing(l))
            except:
                print '[SKIP]', location, l
        print place, '=>', len(parsed_listings)
        result[location] = (location_name, (lat, lng), len(parsed_listings), parsed_listings)
    except:
        print '[ERROR]', location
        # TODO log
        traceback.print_exc()
    time.sleep(3)

with open('biglumber.json', 'w') as f:
    print >> f, json.dumps(result)
