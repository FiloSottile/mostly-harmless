import requests
import collections
import bs4
import json

with open('creds.json') as f:
    creds = json.load(f)

s = requests.Session()
s.headers = { 'User-Agent': 'MyFlightradar import 1.0' }
assert s.post("https://www.flightradar24.com/user/login", data={
    "email": creds["FR_EMAIL"],
    "password": creds["FR_PASSWORD"],
}).json()["success"]
s.get("https://my.flightradar24.com/sign-in")

def get_flights():
    r = s.get("https://my.flightradar24.com/%s/flights" % creds["FR_USER"])

    Flight = collections.namedtuple('Flight', ['date', 'from_', 'to', 'flight'])
    flights = []

    last = -1
    soup = bs4.BeautifulSoup(r.text, "html5lib")
    for row in soup.find_all(lambda tag: tag.has_attr('data-row-number')):
        flights.append(Flight(
            date=row.find(class_="inner-date").text.strip(),
            from_=row.find(class_="flight-from").text.strip(),
            to=row.find(class_="flight-to").text.strip(),
            flight=row.find(class_="flight-flight").text.strip(),
        ))
        last = int(row['data-row-number'])

    if last == -1:
        return ''
    
    while True:
        # TODO: report vulnerability: no authentication on this endpoint
        r = s.get('https://my.flightradar24.com/public-scripts/flight-list/%s/%d/' % (creds["FR_USER"], last)).json()
        if len(r) == 0: break
        for last, row in sorted(r.items()):
            flights.append(Flight(
                date=bs4.BeautifulSoup(row[0], "html5lib").find(class_="inner-date").text,
                from_=bs4.BeautifulSoup(row[2], "html5lib").text,
                to=bs4.BeautifulSoup(row[3], "html5lib").text,
                flight=row[1],
            ))
            last = int(last)

    return reversed(flights)

def get_airport(code):
    l = s.get('https://my.flightradar24.com/add-flight/search/airport/?term=' + code.strip()).json()
    return l[0]

def get_airline(code):
    l = s.get('https://my.flightradar24.com/add-flight/search/airline/?term=' + code.strip()).json()
    return l[0]

def add_flight(data):
    r = s.post('https://my.flightradar24.com/add-flight', data=data)
    r.raise_for_status()
    if "Couldn't add flight" in r.text: raise Exception

def main():
    for f in get_flights():
        print(f.date, f.from_, f.to, f.flight)

if __name__ == '__main__':
    main()
