loadedNew();

if (typeof BADFISH_SELF !== 'undefined') triggeredSelf();
else if (typeof BADFISH_SUPER !== 'undefined') triggeredSuper();
else if (typeof BADFISH_SAN !== 'undefined') triggeredSan();
else allGood();

if (typeof console !== 'undefined') console.log('Loaded new');

