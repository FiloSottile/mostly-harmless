#! /bin/sh

jq -r '.AirObject[]|.Segment|(arrays[], objects)|(.StartDateTime.date +" "+ .start_airport_code +" "+ .end_airport_code +" "+ .marketing_airline_code + .marketing_flight_number)' | sort
