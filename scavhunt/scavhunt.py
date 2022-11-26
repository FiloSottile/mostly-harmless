import os, time, math, sys
from datetime import datetime, timedelta
from twilio.rest import Client
import requests

msg = "What's with the nails? Item 84. This is text message %d of 5840. The next message will be delivered at %s, the last on 2024-08-11."
last = datetime.fromisoformat("2024-08-12T00:00:00")
now = datetime.now().replace(microsecond=0)

if now >= last: sys.exit(42)
count = math.floor(5840 - (last - now) / timedelta(hours=3)) + 1
m = msg % (count, (now + timedelta(hours=3)).isoformat())
print(m)

account_sid = os.environ['TWILIO_ACCOUNT_SID']
auth_token = os.environ['TWILIO_AUTH_TOKEN']
client = Client(account_sid, auth_token)

from_ = os.environ['SCAVHUNT_FROM']
to = os.environ['SCAVHUNT_TO']
message = client.messages.create(body = m, from_=from_, to=to)

hc_uuid = os.environ['SCAVHUNT_HC_UUID']
requests.post("https://hc-ping.com/" + hc_uuid, data="sid=" + message.sid, timeout=10)
