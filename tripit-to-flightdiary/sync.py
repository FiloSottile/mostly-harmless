import tripit
import flightradar
import json,sys,getopt

def main(argv):
    inputfile = ''
    try:
        opts, args = getopt.getopt(argv,"hi:",["ifile="])
    except getopt.GetoptError:
        pass # No input file detected, that's okay!
    for opt, arg in opts:
        if opt == '-h':
            print('sync.py [-i <tripit json inputfile>]')
            sys.exit()
        elif opt in ("-i", "--ifile"):
            inputfile = arg
            print('Input file is', inputfile)
    
    if inputfile == '':
        print('No input file detected, querying TripIt for trips...')
        trips = tripit.get_trips()
    else:
        with open(inputfile) as f:
            trips = json.load(f)
    
    flights = set()
    for f in flightradar.get_flights():
        flights.add((f.date, f.flight))

    for ao in trips['AirObject']:
        segments = ao['Segment']
        if type(segments) not in (tuple, list):
            segments = (segments, )
        for s in segments:
            if 'operating_airline_code' in s and \
                'operating_flight_number' in s and (s['StartDateTime']['date'],
                s['operating_airline_code'] + s['operating_flight_number']) in flights:
                continue
    
            airline_code = s.get('marketing_airline_code', s.get('operating_airline_code'))
            flight_number = s.get('marketing_flight_number', s.get('operating_flight_number'))
            if airline_code == None or flight_number == None:
                print("Invalid segment:", s)
                continue
            if (s['StartDateTime']['date'], airline_code + flight_number) in flights:
                continue

            air = flightradar.get_airline(airline_code)
            if not '(%s/' % airline_code in air["label"]: raise Exception
            from_ = flightradar.get_airport(s['start_airport_code'])
            if not from_['iata'].startswith(s['start_airport_code']): raise Exception
            to = flightradar.get_airport(s['end_airport_code'])
            if not to['iata'].startswith(s['end_airport_code']): raise Exception

            if not 'time' in s['StartDateTime'] or not 'time' in s['EndDateTime']:
                continue
    
            data = {
                "departure-date": s['StartDateTime']['date'],
                "flight-number": airline_code + flight_number,
                "departure-airport": from_["label"],
                "departure-airport-value": from_["id"],
                "departure-time-hour": s['StartDateTime']['time'].split(':')[0],
                "departure-time-minute": s['StartDateTime']['time'].split(':')[1],
                "arrival-airport": to["label"],
                "arrival-airport-value": to["id"],
                "arrival-time-hour": s['EndDateTime']['time'].split(':')[0],
                "arrival-time-minute": s['EndDateTime']['time'].split(':')[1],
                "airline": air["label"],
                "airline-value": air["id"],
    
                "aircraft": "",
                "aircraft-value": "NULL",
                "aircraft-registration": "",
                "seat-number": "",
                "flight-comment": "",
                "post-facebook-type": "",
                "facebook-post-content": "",
                "duration-hour": "",
                "duration-minute": "",
            }
    
            flightradar.add_flight(data)
    
            print('Added flight %s %s %s %s' % (
                s['StartDateTime']['date'],
                s['start_airport_code'],
                s['end_airport_code'],
                airline_code + flight_number,
            ))

if __name__ == '__main__':
    main(sys.argv[1:])
