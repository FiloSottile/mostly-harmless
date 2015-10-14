import requests
import collections
import bs4
import json

with open('creds.json') as f:
    creds = json.load(f)

s = requests.Session()
s.post("http://flightdiary.net/sign-in", data={
    "username": creds["FD_USER"],
    "password": creds["FD_PASSWORD"],
    "b730gn3": "!",
})

def get_flights():
    r = s.get("http://flightdiary.net/%s/flights" % creds["FD_USER"])

    Flight = collections.namedtuple('Flight', ['date', 'from_', 'to', 'flight'])
    flights = []

    soup = bs4.BeautifulSoup(r.text, "html5lib")
    for row in soup.find_all(lambda tag: tag.has_attr('data-row-number')):
        flights.append(Flight(
            date=row.find(class_="flight-date").text.strip(),
            from_=row.find(class_="flight-from").text.strip(),
            to=row.find(class_="flight-to").text.strip(),
            flight=row.find(class_="flight-flight").text.strip(),
        ))
        last = int(row['data-row-number'])

    while True:
        r = s.get('http://flightdiary.net/public-scripts/flight-list/%s/%d/' % (creds["FD_USER"], last)).json()
        if len(r) == 0: break
        for last, row in sorted(r.items()):
            flights.append(Flight(
                date=row[0],
                from_=bs4.BeautifulSoup(row[2], "html5lib").text,
                to=bs4.BeautifulSoup(row[3], "html5lib").text,
                flight=row[1],
            ))
            last = int(last)

    return reversed(flights)

def get_airport(code):
    l = s.get('http://flightdiary.net/add-flight/search/airport/?term=' + code).json()
    return l[0]

def get_airline(code):
    l = s.get('http://flightdiary.net/add-flight/search/airline/?term=' + code).json()
    return l[0]

def add_flight(data):
    data["ac85b1561c"] = "!"
    r = s.post('http://flightdiary.net/add-flight', data=data)
    if "Couldn't add flight" in r.text: raise Exception

def main():
    for f in get_flights():
        print f.date, f.from_, f.to, f.flight

if __name__ == '__main__':
    main()
