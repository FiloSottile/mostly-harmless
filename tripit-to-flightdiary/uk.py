import json
import sys
from datetime import datetime

year_start, year_end = datetime.strptime("2016-04-06", "%Y-%m-%d"), datetime.strptime("2017-04-05", "%Y-%m-%d")

airports = "STN,SOU,SEN,OXF,NWI,NQY,NCL,MME,MAN,LYX,LTN,LPL,LHR,LGW,LEQ,LCY,LBA,ISC,HUY,GLO,EXT,EMA,DSA,BRS,BOH,BLK,BHX,BFS,BHD,LDY,WRY,WIC,TRE,SYY,SOY,PPW,PIK,OBN,NRL,NDY,LWK,LSI,KOI,INV,ILY,GLA,FIE,EOI,EDI,DND,CSA,COL,CAL,BRR,BEB,ABZ,VLY,CWL,ACI,GCI,IOM,JER".split(",")

trips = json.load(sys.stdin)

events = []

for ao in trips['AirObject']:
    segments = ao['Segment']
    if type(segments) not in (tuple, list):
        segments = (segments, )
    for s in segments:
        if s['end_airport_code'] in airports:
            events.append((s['StartDateTime']['date'], "IN", s['end_airport_code']))
        if s['start_airport_code'] in airports:
            events.append((s['StartDateTime']['date'], "OUT", s['start_airport_code']))

events.sort()
last_out = None
nights_out = 0
for event in events:
    print event[0], "-", "entered" if event[1] == "IN" else "left", "UK via", event[2]
    if event[1] == "IN":
        if last_out is None:
            print "Warning: DROPPED THIS 'IN' EVENT"
            print
            continue
        from_, to = datetime.strptime(last_out, "%Y-%m-%d"), datetime.strptime(event[0], "%Y-%m-%d"), 
        if (from_ < year_start and to < year_start) or (from_ > year_end and to > year_end):
            last_out = None
            continue
        from_, to = max(from_, year_start), min(to, year_end)
        print "Outside the UK from", last_out, "to", event[0], "-", (to - from_).days, "nights"
        print
        nights_out += (to - from_).days
        last_out = None
    else:
        if last_out is not None:
            print "Warning: DROPPED PREVIOUS 'OUT' EVENT"
        last_out = event[0]
if last_out is not None:
    print "Warning: DROPPED PREVIOUS 'OUT' EVENT"

print
print "Total nights out between", year_start, "and", year_end, "-", nights_out
