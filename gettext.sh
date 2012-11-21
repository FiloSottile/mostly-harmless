#! /bin/bash

xgettext -L Python -d messages -o i18n/messages.po -j *.py templates/*.html 

for lang in i18n/*/; do

xgettext -L Python -d messages -o ${lang}LC_MESSAGES/messages.po -j *.py templates/*.html 
msgfmt -o ${lang}LC_MESSAGES/messages.mo ${lang}LC_MESSAGES/messages.po

done