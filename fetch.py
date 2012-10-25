import urllib2
import re
import time
import json
import traceback

from geopy import geocoders  
g = geocoders.Google(domain='maps.google.com')

def parse_listing(listing):
    res = {}
    if '<strong>Event name</strong>' in listing:
        res['name'] = re.search(r'<strong>Event name</strong>: <b>([^<]+)</b>', listing).group(1)
    else:
        res['name'] = re.search(r'<li>Name: (?:<[^>]+?>)*([^<]+)', listing).group(1)
    mobj = re.search(r'Email: <b>(\w+) &quot;at"</b>&nbsp;<strong>\n([\w\.]+)&nbsp;</strong>', listing)
    res['email'] = mobj.group(1) + '@' + mobj.group(2) if mobj else None
    mobj = re.search(r'Fingerprint: <a href="[^"]*">(\w+)<b>(\w+)</b></a>\n \(([\w/]+)\)', listing)
    res['fingerprint'] = mobj.group(1) + mobj.group(2) if mobj else None
    res['key'] = mobj.group(2) if mobj else None
    res['keytype'] = mobj.group(3) if mobj else None
    mobj = re.search(r'URL: <b><a href="([^"]+)">', listing, re.DOTALL)
    res['url'] = mobj.group(1) if mobj else None
    mobj = re.search(r'Key comment: <strong><tt>(.*?)</tt></strong>', listing, re.DOTALL)
    res['key_comment'] = mobj.group(1) if mobj else None
    mobj = re.search(r'Notes: <br /><blockquote><tt>(.*?)</tt></blockquote>', listing, re.DOTALL)
    res['notes'] = mobj.group(1).replace('<br />\r\n', '\n').replace('\r\n', '\n') if mobj else None
    return res

# TODO Key ID field like here http://biglumber.com/x/web?sn=Enrico+Franceschi

index_page = urllib2.urlopen('http://biglumber.com/x/web?va=1').read().decode('iso-8859-1')
#index_page = index_page[index_page.index('<li><a href="http://biglumber.com/x/web?so=Italy">'):]
#index_page = index_page[:index_page.index('</ul></li>')]

locations_list = re.findall(r'http://biglumber\.com/x/web\?sl=(\d+)', index_page)

result = {}

for location in locations_list:
    try:
        city_listings_page = urllib2.urlopen('http://biglumber.com/x/web?sl=' + location).read().decode('iso-8859-1')
        location_name = re.search(r'<h1>Biglumber listings for ([^<]+)</h1>', city_listings_page).group(1)
        place, (lat, lng) = g.geocode(location_name.encode('utf8'), exactly_one=False)[0]
        listings = re.findall(r'^<ul>$(.*?)^</ul>$', city_listings_page, re.MULTILINE | re.DOTALL)
        parsed_listings = map(parse_listing, listings)
        print place, '=>', len(parsed_listings)
        result[location] = (location_name, (lat, lng), len(parsed_listings), parsed_listings)
    except:
        print '[ERROR]', location
        traceback.print_exc()
    time.sleep(3)

with open('biglumber.json', 'w') as f:
    print >> f, json.dumps(result)