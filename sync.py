import tripit
import flightdiary

flights = set()
for f in flightdiary.get_flights():
    flights.add((f.date, f.flight))

trips = tripit.get_trips()
for ao in trips['AirObject']:
    segments = ao['Segment']
    if type(segments) not in (tuple, list):
        segments = (segments, )
    for s in segments:
        if (s['StartDateTime']['date'], s['marketing_airline_code']
            + s['marketing_flight_number']) in flights:
            continue

        air = flightdiary.get_airline(s['marketing_airline_code'])
        if not '(%s/' % s['marketing_airline_code'] in air["label"]: raise Exception
        from_ = flightdiary.get_airport(s['start_airport_code'])
        if not from_['id'].startswith(s['start_airport_code']): raise Exception
        to = flightdiary.get_airport(s['end_airport_code'])
        if not to['id'].startswith(s['end_airport_code']): raise Exception

        data = {
            "departure-date": s['StartDateTime']['date'],
            "flight-number": s['marketing_airline_code'] + s['marketing_flight_number'],
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

        flightdiary.add_flight(data)

        print 'Added flight %s %s %s %s' % (
            s['StartDateTime']['date'],
            s['start_airport_code'],
            s['end_airport_code'],
            s['marketing_airline_code'] + s['marketing_flight_number'],
        )
