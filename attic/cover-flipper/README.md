The Facebook Cover Flipper
==========================

This Facebook canvas app lets you horizontally flip your Timeline cover photo (useful for example when the main subject is on the left and hidden by your profile photo) and:
* is built using Python and web.py
* uses gettext for i18n and PIL (the Pillow fork) for image processing
* is deployed on Heroku using guincorn

Please note, the app relies on the following env vars:
```
FACEBOOK_APP_ID=109689179196355
FACEBOOK_SECRET=1234567890abcdef1234567890abcdef
FACEBOOK_NAMESPACE=coverflipper
DEBUG=0
TRACKING_ID=UA-5634110-7
```

Also, the following commits are great boilerplates:
* [`45f45ea3a2db12223d8607df5689bd20e0996b51`](https://github.com/FiloSottile/cover-flipper/tree/45f45ea3a2db12223d8607df5689bd20e0996b51) most basic web.py + gunicorn app on Heroku
* [`90da7d96c3604e60b1e6e9c83710ece1588895c4`](https://github.com/FiloSottile/cover-flipper/tree/90da7d96c3604e60b1e6e9c83710ece1588895c4) all the above, plus FB OAuth authorization
* [`d11e77e17b1a6fc28747778599d410f3e9940e5e`](https://github.com/FiloSottile/cover-flipper/tree/d11e77e17b1a6fc28747778599d410f3e9940e5e) and finally, gettext for i18n

## License

Copyright © 2012 Filippo Valsorda, released under the [MIT License](http://filosottile.mit-license.org/).