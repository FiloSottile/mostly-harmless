---
layout: post
title: "Brainwallets: from the password to the address"
date: 2013-12-05 17:43
comments: true
categories: bitcoin
---

[Brainwallets][brainwallet] are Bitcoin wallets generated uniquely from a passphrase that the users keeps in his mind so that it is required and sufficient to move the funds.

But what is actually the process that takes a password and spits a Bitcoin wallet address? Let's dissect it.

### 1. From a password to a secret value

So, we have a password, but we need a fixed-size (256-bit) secret value to make our private key. This step can be done in a number of ways as it boils down to hashing the password but is crucial to the strength of the resulting brainwallet.

Let's have a look at how popular Brainwallet generators do it. (As of 20131204)

+----------------------------------------------------------|-------------------|-----------------------------------+
  **Generator**                                            | **Algorithm**     | **Notes**
+----------------------------------------------------------|-------------------|-----------------------------------+
  [brainwallet.org](http://brainwallet.org/)               | SHA256(password)  |
  [bitaddress.org](https://www.bitaddress.org/)            | SHA256(password)  |
  [eharning.us/brainwallet-ltc](http://www.eharning.us/brainwallet-ltc/)       | SHA256(password) | Litecoin wallet
  [brainwallet.ltcbbs.com](http://brainwallet.ltcbbs.com/) | SHA256(password)  | Litecoin wallet
  [keybase.io/warp](https://keybase.io/warp/)              | scrypt(password, salt) XOR<br>PBKDF2(password, salt)
+==========================================================|===================|===================================+

A lot of them just take the unsalted [SHA256][sha] hash of the password. **This is wrong**. Because SHA256 **is fast** and that means that an attacker can pregenerate huge tables of all possible brainwallets to monitor and empty them (Spoiler: they do). This kind of thing -- turning a human supplied password into a public hash -- is **exactly** what [password stretching][key stretching] are for, and not using them here is an oversight as bad as not using them to store website user passwords, if not worse since here the hashes (the addresses) are public by default.

(Hint: use [WarpWallet](https://keybase.io/warp/). It's built by people who know what they are doing, and employs a proper KDF, making attacking your wallet really difficult.)

### 2. From the secret value to a private key

This is step is trivial. Actually, the output of the hashing above taken as a 256-bit unsigned number *is already the private key*, what is commonly called the **secret exponent**.

But we are used to see those pretty private keys beginning with a 5, so let's see how it is encoded. That format is called [**WIF**, Wallet import format][wif], and it is pretty handy as it has checksumming built in and employs a charset without confusing characters ([Base58Check][Base58Check]) -- exactly like a Bitcoin address.

A snippet is worth a thousand words:

```python
# Prepend the 0x80 version/application byte
private_key = b'\x80' + private_key
# Append the first 4 bytes of SHA256(SHA256(private_key)) as a checksum
private_key += sha256(sha256(private_key).digest()).digest()[:4]
# Convert to Base58 encoding
code_string = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
value = int.from_bytes(private_key, byteorder='big')
output = ""
while value:
    value, remainder = divmod(value, 58)
    output = code_string[remainder] + output
```

### 3. From a private key to a public key

As [Wikipedia tells us][ECDSA] a ECDSA private key is just the scalar product of a private key (the secret exponent) and the curve -- [secp256k1][secp256k1] for Bitcoin -- base point. [How to do that][scalar] is complex, but let's just take it for granted, as you'll either use a librarty for this or research further by yourself.

What we get out of that operation is a pair **(x, y)** denoting a point on the curve, our public key.

<!-- NOTE: **y**, known its sign, can be calculated from **x**, and this has spawned -->

### 4. From the public key to a Bitcoin address

We're almost there! Now we just need to turn that ECDSA public key into a standard Bitcoin address.

The process is the same as point 4, executed on the SHA256+RIPEMD160 hash of the packed x and y values. Go go snippet:

```python
# 1 byte 0x04, 32 bytes X, 32 bytes Y
public_key = b'\x04' + x.to_bytes(32, byteorder='big') + y.to_bytes(32, byteorder='big')
# Run SHA256 and RIPEMD-160 chained
address = ripemd160(sha256(public_key).digest())
# From now on it is point 4
# Prepend the 0x00 version/application byte for MainNet
address = b'\x00' + address
# Append the first 4 bytes of SHA256(SHA256(address)) as a checksum
address += sha256(sha256(address).digest()).digest()[:4]
# Convert to Base58 encoding
code_string = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
value = int.from_bytes(address, byteorder='big')
output = ""
while value:
    value, remainder = divmod(value, 58)
    output = code_string[remainder] + output
# This wan not needed for the WIF format, but the encoding wants us to normalize the number
# (remove leading zeroes) and prepend a zero for each leading zero byte in the original
output = output.lstrip(code_string[0])
for ch in address:
    if ch == 0: output = code_string[0] + output
    else: break
```

And it's done!


[sha]: https://en.wikipedia.org/wiki/SHA-2
[brainwallet]: https://en.bitcoin.it/wiki/Brainwallet
[key stretching]: https://en.wikipedia.org/wiki/Key_stretching
[wif]: https://en.bitcoin.it/wiki/WIF
[Base58Check]: https://en.bitcoin.it/wiki/Base58Check
[ECDSA]: https://en.wikipedia.org/wiki/Elliptic_Curve_DSA
[scalar]: https://en.wikipedia.org/wiki/Elliptic_curve_point_multiplication
[secp256k1]: https://en.bitcoin.it/wiki/Secp256k1
