# pkguninstall

pkguninstall will use the information stored in the "receipts db" to allow you to remove files and folders created by .pkg installers.

It handles custom locations/volumes and will remove the receipt once done.

CAUTION: check with `-n` first and make sure you know what you are doing. This is not a "intended" feature of most installers.

```
Usage: pkguninstall [-vnh] pkgid

Options:
  -n   Simulate only
  -v   Verbose
  -h   Print this help
```

For the list of installed .pkg use `pkgutil --pkgs`.
