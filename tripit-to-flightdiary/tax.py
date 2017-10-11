import json
import sys
from datetime import datetime

start, end = "2017-01-01", "2017-12-31"
year_start, year_end = datetime.strptime(start, "%Y-%m-%d"), datetime.strptime(end, "%Y-%m-%d")

airports_uk = "LCY,LGW,LHR,LTN,STN,MAN".split(",")
airports_us = "CLT,DEN,DFW,EWR,IAD,JFK,LAS,LAX,LGA,ORD,PHX,SEA,SFO,SJC".split(",")
airports = airports_us

trips = json.load(sys.stdin)

# for ao in trips['AirObject']:
#     segments = ao['Segment']
#     if type(segments) not in (tuple, list):
#         segments = (segments, )
#     for s in segments:
#         print s['end_airport_code'], s['end_city_name']
#         print s['start_airport_code'], s['start_city_name']
# exit()

events = []

for ao in trips['AirObject']:
    segments = ao['Segment']
    if type(segments) not in (tuple, list):
        segments = (segments, )
    for s in segments:
        if s['end_airport_code'] in airports and not s['start_airport_code'] in airports:
            if not 'date' in s['EndDateTime']:
                print 'WARNING: skipped flight started on', s['StartDateTime']['date']
                continue
            events.append((s['EndDateTime']['date'], "IN", s['end_airport_code']))
        if s['start_airport_code'] in airports and not s['end_airport_code'] in airports:
            events.append((s['StartDateTime']['date'], "OUT", s['start_airport_code']))

events.sort()
last_in = None
nights_in = 0
for event in events:
    print event[0], "-", "entered" if event[1] == "IN" else "left", "via", event[2]
    if event[1] == "OUT":
        if last_in is None:
            print "Warning: DROPPED THIS 'OUT' EVENT"
            print
            continue

        from_, to = datetime.strptime(last_in, "%Y-%m-%d"), datetime.strptime(event[0], "%Y-%m-%d"), 
        if (from_ < year_start and to < year_start) or (from_ > year_end and to > year_end):
            last_in = None
            continue
        from_, to = max(from_, year_start), min(to, year_end)
        print "From", from_.strftime("%Y-%m-%d"), "to", to.strftime("%Y-%m-%d"), "-", (to - from_).days + 1, "days"
        print
        nights_in += (to - from_).days + 1

        last_in = None
    else:
        if last_in is not None:
            print "Warning: DROPPED PREVIOUS 'IN' EVENT"
        last_in = event[0]
if last_in is not None:
    print "Warning: DROPPED PREVIOUS 'IN' EVENT"

print
print "Total days in between", year_start.strftime("%Y-%m-%d"), "and", year_end.strftime("%Y-%m-%d"), "-", nights_in
