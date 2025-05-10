"use strict";
var age = (() => {
  var __defProp = Object.defineProperty;
  var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
  var __getOwnPropNames = Object.getOwnPropertyNames;
  var __hasOwnProp = Object.prototype.hasOwnProperty;
  var __export = (target, all) => {
    for (var name in all)
      __defProp(target, name, { get: all[name], enumerable: true });
  };
  var __copyProps = (to, from, except, desc) => {
    if (from && typeof from === "object" || typeof from === "function") {
      for (let key of __getOwnPropNames(from))
        if (!__hasOwnProp.call(to, key) && key !== except)
          __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
    }
    return to;
  };
  var __toCommonJS = (mod2) => __copyProps(__defProp({}, "__esModule", { value: true }), mod2);

  // dist/index.js
  var dist_exports = {};
  __export(dist_exports, {
    Decrypter: () => Decrypter,
    Encrypter: () => Encrypter,
    Stanza: () => Stanza,
    armor: () => armor_exports,
    generateIdentity: () => generateIdentity,
    identityToRecipient: () => identityToRecipient,
    webauthn: () => webauthn_exports
  });

  // node_modules/@noble/hashes/esm/_assert.js
  function anumber(n) {
    if (!Number.isSafeInteger(n) || n < 0)
      throw new Error("positive integer expected, got " + n);
  }
  function isBytes(a) {
    return a instanceof Uint8Array || ArrayBuffer.isView(a) && a.constructor.name === "Uint8Array";
  }
  function abytes(b, ...lengths) {
    if (!isBytes(b))
      throw new Error("Uint8Array expected");
    if (lengths.length > 0 && !lengths.includes(b.length))
      throw new Error("Uint8Array expected of length " + lengths + ", got length=" + b.length);
  }
  function ahash(h) {
    if (typeof h !== "function" || typeof h.create !== "function")
      throw new Error("Hash should be wrapped by utils.wrapConstructor");
    anumber(h.outputLen);
    anumber(h.blockLen);
  }
  function aexists(instance, checkFinished = true) {
    if (instance.destroyed)
      throw new Error("Hash instance has been destroyed");
    if (checkFinished && instance.finished)
      throw new Error("Hash#digest() has already been called");
  }
  function aoutput(out, instance) {
    abytes(out);
    const min = instance.outputLen;
    if (out.length < min) {
      throw new Error("digestInto() expects output buffer of length at least " + min);
    }
  }

  // node_modules/@noble/hashes/esm/crypto.js
  var crypto2 = typeof globalThis === "object" && "crypto" in globalThis ? globalThis.crypto : void 0;

  // node_modules/@noble/hashes/esm/utils.js
  function u32(arr) {
    return new Uint32Array(arr.buffer, arr.byteOffset, Math.floor(arr.byteLength / 4));
  }
  function createView(arr) {
    return new DataView(arr.buffer, arr.byteOffset, arr.byteLength);
  }
  function rotr(word, shift) {
    return word << 32 - shift | word >>> shift;
  }
  function rotl(word, shift) {
    return word << shift | word >>> 32 - shift >>> 0;
  }
  var isLE = /* @__PURE__ */ (() => new Uint8Array(new Uint32Array([287454020]).buffer)[0] === 68)();
  function byteSwap(word) {
    return word << 24 & 4278190080 | word << 8 & 16711680 | word >>> 8 & 65280 | word >>> 24 & 255;
  }
  function byteSwap32(arr) {
    for (let i = 0; i < arr.length; i++) {
      arr[i] = byteSwap(arr[i]);
    }
  }
  function utf8ToBytes(str) {
    if (typeof str !== "string")
      throw new Error("utf8ToBytes expected string, got " + typeof str);
    return new Uint8Array(new TextEncoder().encode(str));
  }
  function toBytes(data) {
    if (typeof data === "string")
      data = utf8ToBytes(data);
    abytes(data);
    return data;
  }
  var Hash = class {
    // Safe version that clones internal state
    clone() {
      return this._cloneInto();
    }
  };
  function checkOpts(defaults, opts) {
    if (opts !== void 0 && {}.toString.call(opts) !== "[object Object]")
      throw new Error("Options should be object or undefined");
    const merged = Object.assign(defaults, opts);
    return merged;
  }
  function wrapConstructor(hashCons) {
    const hashC = (msg) => hashCons().update(toBytes(msg)).digest();
    const tmp = hashCons();
    hashC.outputLen = tmp.outputLen;
    hashC.blockLen = tmp.blockLen;
    hashC.create = () => hashCons();
    return hashC;
  }
  function randomBytes(bytesLength = 32) {
    if (crypto2 && typeof crypto2.getRandomValues === "function") {
      return crypto2.getRandomValues(new Uint8Array(bytesLength));
    }
    if (crypto2 && typeof crypto2.randomBytes === "function") {
      return crypto2.randomBytes(bytesLength);
    }
    throw new Error("crypto.getRandomValues must be defined");
  }

  // node_modules/@noble/hashes/esm/hmac.js
  var HMAC = class extends Hash {
    constructor(hash, _key) {
      super();
      this.finished = false;
      this.destroyed = false;
      ahash(hash);
      const key = toBytes(_key);
      this.iHash = hash.create();
      if (typeof this.iHash.update !== "function")
        throw new Error("Expected instance of class which extends utils.Hash");
      this.blockLen = this.iHash.blockLen;
      this.outputLen = this.iHash.outputLen;
      const blockLen = this.blockLen;
      const pad = new Uint8Array(blockLen);
      pad.set(key.length > blockLen ? hash.create().update(key).digest() : key);
      for (let i = 0; i < pad.length; i++)
        pad[i] ^= 54;
      this.iHash.update(pad);
      this.oHash = hash.create();
      for (let i = 0; i < pad.length; i++)
        pad[i] ^= 54 ^ 92;
      this.oHash.update(pad);
      pad.fill(0);
    }
    update(buf) {
      aexists(this);
      this.iHash.update(buf);
      return this;
    }
    digestInto(out) {
      aexists(this);
      abytes(out, this.outputLen);
      this.finished = true;
      this.iHash.digestInto(out);
      this.oHash.update(out);
      this.oHash.digestInto(out);
      this.destroy();
    }
    digest() {
      const out = new Uint8Array(this.oHash.outputLen);
      this.digestInto(out);
      return out;
    }
    _cloneInto(to) {
      to || (to = Object.create(Object.getPrototypeOf(this), {}));
      const { oHash, iHash, finished, destroyed, blockLen, outputLen } = this;
      to = to;
      to.finished = finished;
      to.destroyed = destroyed;
      to.blockLen = blockLen;
      to.outputLen = outputLen;
      to.oHash = oHash._cloneInto(to.oHash);
      to.iHash = iHash._cloneInto(to.iHash);
      return to;
    }
    destroy() {
      this.destroyed = true;
      this.oHash.destroy();
      this.iHash.destroy();
    }
  };
  var hmac = (hash, key, message) => new HMAC(hash, key).update(message).digest();
  hmac.create = (hash, key) => new HMAC(hash, key);

  // node_modules/@noble/hashes/esm/hkdf.js
  function extract(hash, ikm, salt) {
    ahash(hash);
    if (salt === void 0)
      salt = new Uint8Array(hash.outputLen);
    return hmac(hash, toBytes(salt), toBytes(ikm));
  }
  var HKDF_COUNTER = /* @__PURE__ */ new Uint8Array([0]);
  var EMPTY_BUFFER = /* @__PURE__ */ new Uint8Array();
  function expand(hash, prk, info, length = 32) {
    ahash(hash);
    anumber(length);
    if (length > 255 * hash.outputLen)
      throw new Error("Length should be <= 255*HashLen");
    const blocks = Math.ceil(length / hash.outputLen);
    if (info === void 0)
      info = EMPTY_BUFFER;
    const okm = new Uint8Array(blocks * hash.outputLen);
    const HMAC2 = hmac.create(hash, prk);
    const HMACTmp = HMAC2._cloneInto();
    const T = new Uint8Array(HMAC2.outputLen);
    for (let counter = 0; counter < blocks; counter++) {
      HKDF_COUNTER[0] = counter + 1;
      HMACTmp.update(counter === 0 ? EMPTY_BUFFER : T).update(info).update(HKDF_COUNTER).digestInto(T);
      okm.set(T, hash.outputLen * counter);
      HMAC2._cloneInto(HMACTmp);
    }
    HMAC2.destroy();
    HMACTmp.destroy();
    T.fill(0);
    HKDF_COUNTER.fill(0);
    return okm.slice(0, length);
  }
  var hkdf = (hash, ikm, salt, info, length) => expand(hash, extract(hash, ikm, salt), info, length);

  // node_modules/@noble/hashes/esm/_md.js
  function setBigUint64(view, byteOffset, value, isLE3) {
    if (typeof view.setBigUint64 === "function")
      return view.setBigUint64(byteOffset, value, isLE3);
    const _32n = BigInt(32);
    const _u32_max = BigInt(4294967295);
    const wh = Number(value >> _32n & _u32_max);
    const wl = Number(value & _u32_max);
    const h = isLE3 ? 4 : 0;
    const l = isLE3 ? 0 : 4;
    view.setUint32(byteOffset + h, wh, isLE3);
    view.setUint32(byteOffset + l, wl, isLE3);
  }
  function Chi(a, b, c) {
    return a & b ^ ~a & c;
  }
  function Maj(a, b, c) {
    return a & b ^ a & c ^ b & c;
  }
  var HashMD = class extends Hash {
    constructor(blockLen, outputLen, padOffset, isLE3) {
      super();
      this.blockLen = blockLen;
      this.outputLen = outputLen;
      this.padOffset = padOffset;
      this.isLE = isLE3;
      this.finished = false;
      this.length = 0;
      this.pos = 0;
      this.destroyed = false;
      this.buffer = new Uint8Array(blockLen);
      this.view = createView(this.buffer);
    }
    update(data) {
      aexists(this);
      const { view, buffer, blockLen } = this;
      data = toBytes(data);
      const len = data.length;
      for (let pos = 0; pos < len; ) {
        const take = Math.min(blockLen - this.pos, len - pos);
        if (take === blockLen) {
          const dataView = createView(data);
          for (; blockLen <= len - pos; pos += blockLen)
            this.process(dataView, pos);
          continue;
        }
        buffer.set(data.subarray(pos, pos + take), this.pos);
        this.pos += take;
        pos += take;
        if (this.pos === blockLen) {
          this.process(view, 0);
          this.pos = 0;
        }
      }
      this.length += data.length;
      this.roundClean();
      return this;
    }
    digestInto(out) {
      aexists(this);
      aoutput(out, this);
      this.finished = true;
      const { buffer, view, blockLen, isLE: isLE3 } = this;
      let { pos } = this;
      buffer[pos++] = 128;
      this.buffer.subarray(pos).fill(0);
      if (this.padOffset > blockLen - pos) {
        this.process(view, 0);
        pos = 0;
      }
      for (let i = pos; i < blockLen; i++)
        buffer[i] = 0;
      setBigUint64(view, blockLen - 8, BigInt(this.length * 8), isLE3);
      this.process(view, 0);
      const oview = createView(out);
      const len = this.outputLen;
      if (len % 4)
        throw new Error("_sha2: outputLen should be aligned to 32bit");
      const outLen = len / 4;
      const state = this.get();
      if (outLen > state.length)
        throw new Error("_sha2: outputLen bigger than state");
      for (let i = 0; i < outLen; i++)
        oview.setUint32(4 * i, state[i], isLE3);
    }
    digest() {
      const { buffer, outputLen } = this;
      this.digestInto(buffer);
      const res = buffer.slice(0, outputLen);
      this.destroy();
      return res;
    }
    _cloneInto(to) {
      to || (to = new this.constructor());
      to.set(...this.get());
      const { blockLen, buffer, length, finished, destroyed, pos } = this;
      to.length = length;
      to.pos = pos;
      to.finished = finished;
      to.destroyed = destroyed;
      if (length % blockLen)
        to.buffer.set(buffer);
      return to;
    }
  };

  // node_modules/@noble/hashes/esm/sha256.js
  var SHA256_K = /* @__PURE__ */ new Uint32Array([
    1116352408,
    1899447441,
    3049323471,
    3921009573,
    961987163,
    1508970993,
    2453635748,
    2870763221,
    3624381080,
    310598401,
    607225278,
    1426881987,
    1925078388,
    2162078206,
    2614888103,
    3248222580,
    3835390401,
    4022224774,
    264347078,
    604807628,
    770255983,
    1249150122,
    1555081692,
    1996064986,
    2554220882,
    2821834349,
    2952996808,
    3210313671,
    3336571891,
    3584528711,
    113926993,
    338241895,
    666307205,
    773529912,
    1294757372,
    1396182291,
    1695183700,
    1986661051,
    2177026350,
    2456956037,
    2730485921,
    2820302411,
    3259730800,
    3345764771,
    3516065817,
    3600352804,
    4094571909,
    275423344,
    430227734,
    506948616,
    659060556,
    883997877,
    958139571,
    1322822218,
    1537002063,
    1747873779,
    1955562222,
    2024104815,
    2227730452,
    2361852424,
    2428436474,
    2756734187,
    3204031479,
    3329325298
  ]);
  var SHA256_IV = /* @__PURE__ */ new Uint32Array([
    1779033703,
    3144134277,
    1013904242,
    2773480762,
    1359893119,
    2600822924,
    528734635,
    1541459225
  ]);
  var SHA256_W = /* @__PURE__ */ new Uint32Array(64);
  var SHA256 = class extends HashMD {
    constructor() {
      super(64, 32, 8, false);
      this.A = SHA256_IV[0] | 0;
      this.B = SHA256_IV[1] | 0;
      this.C = SHA256_IV[2] | 0;
      this.D = SHA256_IV[3] | 0;
      this.E = SHA256_IV[4] | 0;
      this.F = SHA256_IV[5] | 0;
      this.G = SHA256_IV[6] | 0;
      this.H = SHA256_IV[7] | 0;
    }
    get() {
      const { A, B, C, D, E, F, G, H } = this;
      return [A, B, C, D, E, F, G, H];
    }
    // prettier-ignore
    set(A, B, C, D, E, F, G, H) {
      this.A = A | 0;
      this.B = B | 0;
      this.C = C | 0;
      this.D = D | 0;
      this.E = E | 0;
      this.F = F | 0;
      this.G = G | 0;
      this.H = H | 0;
    }
    process(view, offset) {
      for (let i = 0; i < 16; i++, offset += 4)
        SHA256_W[i] = view.getUint32(offset, false);
      for (let i = 16; i < 64; i++) {
        const W15 = SHA256_W[i - 15];
        const W2 = SHA256_W[i - 2];
        const s0 = rotr(W15, 7) ^ rotr(W15, 18) ^ W15 >>> 3;
        const s1 = rotr(W2, 17) ^ rotr(W2, 19) ^ W2 >>> 10;
        SHA256_W[i] = s1 + SHA256_W[i - 7] + s0 + SHA256_W[i - 16] | 0;
      }
      let { A, B, C, D, E, F, G, H } = this;
      for (let i = 0; i < 64; i++) {
        const sigma1 = rotr(E, 6) ^ rotr(E, 11) ^ rotr(E, 25);
        const T1 = H + sigma1 + Chi(E, F, G) + SHA256_K[i] + SHA256_W[i] | 0;
        const sigma0 = rotr(A, 2) ^ rotr(A, 13) ^ rotr(A, 22);
        const T2 = sigma0 + Maj(A, B, C) | 0;
        H = G;
        G = F;
        F = E;
        E = D + T1 | 0;
        D = C;
        C = B;
        B = A;
        A = T1 + T2 | 0;
      }
      A = A + this.A | 0;
      B = B + this.B | 0;
      C = C + this.C | 0;
      D = D + this.D | 0;
      E = E + this.E | 0;
      F = F + this.F | 0;
      G = G + this.G | 0;
      H = H + this.H | 0;
      this.set(A, B, C, D, E, F, G, H);
    }
    roundClean() {
      SHA256_W.fill(0);
    }
    destroy() {
      this.set(0, 0, 0, 0, 0, 0, 0, 0);
      this.buffer.fill(0);
    }
  };
  var sha256 = /* @__PURE__ */ wrapConstructor(() => new SHA256());

  // node_modules/@scure/base/lib/esm/index.js
  function isBytes2(a) {
    return a instanceof Uint8Array || ArrayBuffer.isView(a) && a.constructor.name === "Uint8Array";
  }
  function isArrayOf(isString, arr) {
    if (!Array.isArray(arr))
      return false;
    if (arr.length === 0)
      return true;
    if (isString) {
      return arr.every((item) => typeof item === "string");
    } else {
      return arr.every((item) => Number.isSafeInteger(item));
    }
  }
  function afn(input) {
    if (typeof input !== "function")
      throw new Error("function expected");
    return true;
  }
  function astr(label2, input) {
    if (typeof input !== "string")
      throw new Error(`${label2}: string expected`);
    return true;
  }
  function anumber2(n) {
    if (!Number.isSafeInteger(n))
      throw new Error(`invalid integer: ${n}`);
  }
  function aArr(input) {
    if (!Array.isArray(input))
      throw new Error("array expected");
  }
  function astrArr(label2, input) {
    if (!isArrayOf(true, input))
      throw new Error(`${label2}: array of strings expected`);
  }
  function anumArr(label2, input) {
    if (!isArrayOf(false, input))
      throw new Error(`${label2}: array of numbers expected`);
  }
  // @__NO_SIDE_EFFECTS__
  function chain(...args) {
    const id = (a) => a;
    const wrap = (a, b) => (c) => a(b(c));
    const encode2 = args.map((x) => x.encode).reduceRight(wrap, id);
    const decode2 = args.map((x) => x.decode).reduce(wrap, id);
    return { encode: encode2, decode: decode2 };
  }
  // @__NO_SIDE_EFFECTS__
  function alphabet(letters) {
    const lettersA = typeof letters === "string" ? letters.split("") : letters;
    const len = lettersA.length;
    astrArr("alphabet", lettersA);
    const indexes = new Map(lettersA.map((l, i) => [l, i]));
    return {
      encode: (digits) => {
        aArr(digits);
        return digits.map((i) => {
          if (!Number.isSafeInteger(i) || i < 0 || i >= len)
            throw new Error(`alphabet.encode: digit index outside alphabet "${i}". Allowed: ${letters}`);
          return lettersA[i];
        });
      },
      decode: (input) => {
        aArr(input);
        return input.map((letter) => {
          astr("alphabet.decode", letter);
          const i = indexes.get(letter);
          if (i === void 0)
            throw new Error(`Unknown letter: "${letter}". Allowed: ${letters}`);
          return i;
        });
      }
    };
  }
  // @__NO_SIDE_EFFECTS__
  function join(separator = "") {
    astr("join", separator);
    return {
      encode: (from) => {
        astrArr("join.decode", from);
        return from.join(separator);
      },
      decode: (to) => {
        astr("join.decode", to);
        return to.split(separator);
      }
    };
  }
  // @__NO_SIDE_EFFECTS__
  function padding(bits, chr = "=") {
    anumber2(bits);
    astr("padding", chr);
    return {
      encode(data) {
        astrArr("padding.encode", data);
        while (data.length * bits % 8)
          data.push(chr);
        return data;
      },
      decode(input) {
        astrArr("padding.decode", input);
        let end = input.length;
        if (end * bits % 8)
          throw new Error("padding: invalid, string should have whole number of bytes");
        for (; end > 0 && input[end - 1] === chr; end--) {
          const last = end - 1;
          const byte = last * bits;
          if (byte % 8 === 0)
            throw new Error("padding: invalid, string has too much padding");
        }
        return input.slice(0, end);
      }
    };
  }
  var gcd = (a, b) => b === 0 ? a : gcd(b, a % b);
  var radix2carry = /* @__NO_SIDE_EFFECTS__ */ (from, to) => from + (to - gcd(from, to));
  var powers = /* @__PURE__ */ (() => {
    let res = [];
    for (let i = 0; i < 40; i++)
      res.push(2 ** i);
    return res;
  })();
  function convertRadix2(data, from, to, padding2) {
    aArr(data);
    if (from <= 0 || from > 32)
      throw new Error(`convertRadix2: wrong from=${from}`);
    if (to <= 0 || to > 32)
      throw new Error(`convertRadix2: wrong to=${to}`);
    if (/* @__PURE__ */ radix2carry(from, to) > 32) {
      throw new Error(`convertRadix2: carry overflow from=${from} to=${to} carryBits=${/* @__PURE__ */ radix2carry(from, to)}`);
    }
    let carry = 0;
    let pos = 0;
    const max = powers[from];
    const mask = powers[to] - 1;
    const res = [];
    for (const n of data) {
      anumber2(n);
      if (n >= max)
        throw new Error(`convertRadix2: invalid data word=${n} from=${from}`);
      carry = carry << from | n;
      if (pos + from > 32)
        throw new Error(`convertRadix2: carry overflow pos=${pos} from=${from}`);
      pos += from;
      for (; pos >= to; pos -= to)
        res.push((carry >> pos - to & mask) >>> 0);
      const pow3 = powers[pos];
      if (pow3 === void 0)
        throw new Error("invalid carry");
      carry &= pow3 - 1;
    }
    carry = carry << to - pos & mask;
    if (!padding2 && pos >= from)
      throw new Error("Excess padding");
    if (!padding2 && carry > 0)
      throw new Error(`Non-zero padding: ${carry}`);
    if (padding2 && pos > 0)
      res.push(carry >>> 0);
    return res;
  }
  // @__NO_SIDE_EFFECTS__
  function radix2(bits, revPadding = false) {
    anumber2(bits);
    if (bits <= 0 || bits > 32)
      throw new Error("radix2: bits should be in (0..32]");
    if (/* @__PURE__ */ radix2carry(8, bits) > 32 || /* @__PURE__ */ radix2carry(bits, 8) > 32)
      throw new Error("radix2: carry overflow");
    return {
      encode: (bytes) => {
        if (!isBytes2(bytes))
          throw new Error("radix2.encode input should be Uint8Array");
        return convertRadix2(Array.from(bytes), 8, bits, !revPadding);
      },
      decode: (digits) => {
        anumArr("radix2.decode", digits);
        return Uint8Array.from(convertRadix2(digits, bits, 8, revPadding));
      }
    };
  }
  function unsafeWrapper(fn) {
    afn(fn);
    return function(...args) {
      try {
        return fn.apply(null, args);
      } catch (e) {
      }
    };
  }
  var base64 = /* @__PURE__ */ chain(/* @__PURE__ */ radix2(6), /* @__PURE__ */ alphabet("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"), /* @__PURE__ */ padding(6), /* @__PURE__ */ join(""));
  var base64nopad = /* @__PURE__ */ chain(/* @__PURE__ */ radix2(6), /* @__PURE__ */ alphabet("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"), /* @__PURE__ */ join(""));
  var BECH_ALPHABET = /* @__PURE__ */ chain(/* @__PURE__ */ alphabet("qpzry9x8gf2tvdw0s3jn54khce6mua7l"), /* @__PURE__ */ join(""));
  var POLYMOD_GENERATORS = [996825010, 642813549, 513874426, 1027748829, 705979059];
  function bech32Polymod(pre) {
    const b = pre >> 25;
    let chk = (pre & 33554431) << 5;
    for (let i = 0; i < POLYMOD_GENERATORS.length; i++) {
      if ((b >> i & 1) === 1)
        chk ^= POLYMOD_GENERATORS[i];
    }
    return chk;
  }
  function bechChecksum(prefix2, words, encodingConst = 1) {
    const len = prefix2.length;
    let chk = 1;
    for (let i = 0; i < len; i++) {
      const c = prefix2.charCodeAt(i);
      if (c < 33 || c > 126)
        throw new Error(`Invalid prefix (${prefix2})`);
      chk = bech32Polymod(chk) ^ c >> 5;
    }
    chk = bech32Polymod(chk);
    for (let i = 0; i < len; i++)
      chk = bech32Polymod(chk) ^ prefix2.charCodeAt(i) & 31;
    for (let v of words)
      chk = bech32Polymod(chk) ^ v;
    for (let i = 0; i < 6; i++)
      chk = bech32Polymod(chk);
    chk ^= encodingConst;
    return BECH_ALPHABET.encode(convertRadix2([chk % powers[30]], 30, 5, false));
  }
  // @__NO_SIDE_EFFECTS__
  function genBech32(encoding) {
    const ENCODING_CONST = encoding === "bech32" ? 1 : 734539939;
    const _words = /* @__PURE__ */ radix2(5);
    const fromWords = _words.decode;
    const toWords = _words.encode;
    const fromWordsUnsafe = unsafeWrapper(fromWords);
    function encode2(prefix2, words, limit = 90) {
      astr("bech32.encode prefix", prefix2);
      if (isBytes2(words))
        words = Array.from(words);
      anumArr("bech32.encode", words);
      const plen = prefix2.length;
      if (plen === 0)
        throw new TypeError(`Invalid prefix length ${plen}`);
      const actualLength = plen + 7 + words.length;
      if (limit !== false && actualLength > limit)
        throw new TypeError(`Length ${actualLength} exceeds limit ${limit}`);
      const lowered = prefix2.toLowerCase();
      const sum = bechChecksum(lowered, words, ENCODING_CONST);
      return `${lowered}1${BECH_ALPHABET.encode(words)}${sum}`;
    }
    function decode2(str, limit = 90) {
      astr("bech32.decode input", str);
      const slen = str.length;
      if (slen < 8 || limit !== false && slen > limit)
        throw new TypeError(`invalid string length: ${slen} (${str}). Expected (8..${limit})`);
      const lowered = str.toLowerCase();
      if (str !== lowered && str !== str.toUpperCase())
        throw new Error(`String must be lowercase or uppercase`);
      const sepIndex = lowered.lastIndexOf("1");
      if (sepIndex === 0 || sepIndex === -1)
        throw new Error(`Letter "1" must be present between prefix and data only`);
      const prefix2 = lowered.slice(0, sepIndex);
      const data = lowered.slice(sepIndex + 1);
      if (data.length < 6)
        throw new Error("Data must be at least 6 characters long");
      const words = BECH_ALPHABET.decode(data).slice(0, -6);
      const sum = bechChecksum(prefix2, words, ENCODING_CONST);
      if (!data.endsWith(sum))
        throw new Error(`Invalid checksum in ${str}: expected "${sum}"`);
      return { prefix: prefix2, words };
    }
    const decodeUnsafe = unsafeWrapper(decode2);
    function decodeToBytes(str) {
      const { prefix: prefix2, words } = decode2(str, false);
      return { prefix: prefix2, words, bytes: fromWords(words) };
    }
    function encodeFromBytes(prefix2, bytes) {
      return encode2(prefix2, toWords(bytes));
    }
    return {
      encode: encode2,
      decode: decode2,
      encodeFromBytes,
      decodeToBytes,
      decodeUnsafe,
      fromWords,
      fromWordsUnsafe,
      toWords
    };
  }
  var bech32 = /* @__PURE__ */ genBech32("bech32");

  // node_modules/@noble/hashes/esm/pbkdf2.js
  function pbkdf2Init(hash, _password, _salt, _opts) {
    ahash(hash);
    const opts = checkOpts({ dkLen: 32, asyncTick: 10 }, _opts);
    const { c, dkLen, asyncTick } = opts;
    anumber(c);
    anumber(dkLen);
    anumber(asyncTick);
    if (c < 1)
      throw new Error("PBKDF2: iterations (c) should be >= 1");
    const password = toBytes(_password);
    const salt = toBytes(_salt);
    const DK = new Uint8Array(dkLen);
    const PRF = hmac.create(hash, password);
    const PRFSalt = PRF._cloneInto().update(salt);
    return { c, dkLen, asyncTick, DK, PRF, PRFSalt };
  }
  function pbkdf2Output(PRF, PRFSalt, DK, prfW, u) {
    PRF.destroy();
    PRFSalt.destroy();
    if (prfW)
      prfW.destroy();
    u.fill(0);
    return DK;
  }
  function pbkdf2(hash, password, salt, opts) {
    const { c, dkLen, DK, PRF, PRFSalt } = pbkdf2Init(hash, password, salt, opts);
    let prfW;
    const arr = new Uint8Array(4);
    const view = createView(arr);
    const u = new Uint8Array(PRF.outputLen);
    for (let ti = 1, pos = 0; pos < dkLen; ti++, pos += PRF.outputLen) {
      const Ti = DK.subarray(pos, pos + PRF.outputLen);
      view.setInt32(0, ti, false);
      (prfW = PRFSalt._cloneInto(prfW)).update(arr).digestInto(u);
      Ti.set(u.subarray(0, Ti.length));
      for (let ui = 1; ui < c; ui++) {
        PRF._cloneInto(prfW).update(u).digestInto(u);
        for (let i = 0; i < Ti.length; i++)
          Ti[i] ^= u[i];
      }
    }
    return pbkdf2Output(PRF, PRFSalt, DK, prfW, u);
  }

  // node_modules/@noble/hashes/esm/scrypt.js
  function XorAndSalsa(prev, pi, input, ii, out, oi) {
    let y00 = prev[pi++] ^ input[ii++], y01 = prev[pi++] ^ input[ii++];
    let y02 = prev[pi++] ^ input[ii++], y03 = prev[pi++] ^ input[ii++];
    let y04 = prev[pi++] ^ input[ii++], y05 = prev[pi++] ^ input[ii++];
    let y06 = prev[pi++] ^ input[ii++], y07 = prev[pi++] ^ input[ii++];
    let y08 = prev[pi++] ^ input[ii++], y09 = prev[pi++] ^ input[ii++];
    let y10 = prev[pi++] ^ input[ii++], y11 = prev[pi++] ^ input[ii++];
    let y12 = prev[pi++] ^ input[ii++], y13 = prev[pi++] ^ input[ii++];
    let y14 = prev[pi++] ^ input[ii++], y15 = prev[pi++] ^ input[ii++];
    let x00 = y00, x01 = y01, x02 = y02, x03 = y03, x04 = y04, x05 = y05, x06 = y06, x07 = y07, x08 = y08, x09 = y09, x10 = y10, x11 = y11, x12 = y12, x13 = y13, x14 = y14, x15 = y15;
    for (let i = 0; i < 8; i += 2) {
      x04 ^= rotl(x00 + x12 | 0, 7);
      x08 ^= rotl(x04 + x00 | 0, 9);
      x12 ^= rotl(x08 + x04 | 0, 13);
      x00 ^= rotl(x12 + x08 | 0, 18);
      x09 ^= rotl(x05 + x01 | 0, 7);
      x13 ^= rotl(x09 + x05 | 0, 9);
      x01 ^= rotl(x13 + x09 | 0, 13);
      x05 ^= rotl(x01 + x13 | 0, 18);
      x14 ^= rotl(x10 + x06 | 0, 7);
      x02 ^= rotl(x14 + x10 | 0, 9);
      x06 ^= rotl(x02 + x14 | 0, 13);
      x10 ^= rotl(x06 + x02 | 0, 18);
      x03 ^= rotl(x15 + x11 | 0, 7);
      x07 ^= rotl(x03 + x15 | 0, 9);
      x11 ^= rotl(x07 + x03 | 0, 13);
      x15 ^= rotl(x11 + x07 | 0, 18);
      x01 ^= rotl(x00 + x03 | 0, 7);
      x02 ^= rotl(x01 + x00 | 0, 9);
      x03 ^= rotl(x02 + x01 | 0, 13);
      x00 ^= rotl(x03 + x02 | 0, 18);
      x06 ^= rotl(x05 + x04 | 0, 7);
      x07 ^= rotl(x06 + x05 | 0, 9);
      x04 ^= rotl(x07 + x06 | 0, 13);
      x05 ^= rotl(x04 + x07 | 0, 18);
      x11 ^= rotl(x10 + x09 | 0, 7);
      x08 ^= rotl(x11 + x10 | 0, 9);
      x09 ^= rotl(x08 + x11 | 0, 13);
      x10 ^= rotl(x09 + x08 | 0, 18);
      x12 ^= rotl(x15 + x14 | 0, 7);
      x13 ^= rotl(x12 + x15 | 0, 9);
      x14 ^= rotl(x13 + x12 | 0, 13);
      x15 ^= rotl(x14 + x13 | 0, 18);
    }
    out[oi++] = y00 + x00 | 0;
    out[oi++] = y01 + x01 | 0;
    out[oi++] = y02 + x02 | 0;
    out[oi++] = y03 + x03 | 0;
    out[oi++] = y04 + x04 | 0;
    out[oi++] = y05 + x05 | 0;
    out[oi++] = y06 + x06 | 0;
    out[oi++] = y07 + x07 | 0;
    out[oi++] = y08 + x08 | 0;
    out[oi++] = y09 + x09 | 0;
    out[oi++] = y10 + x10 | 0;
    out[oi++] = y11 + x11 | 0;
    out[oi++] = y12 + x12 | 0;
    out[oi++] = y13 + x13 | 0;
    out[oi++] = y14 + x14 | 0;
    out[oi++] = y15 + x15 | 0;
  }
  function BlockMix(input, ii, out, oi, r) {
    let head = oi + 0;
    let tail = oi + 16 * r;
    for (let i = 0; i < 16; i++)
      out[tail + i] = input[ii + (2 * r - 1) * 16 + i];
    for (let i = 0; i < r; i++, head += 16, ii += 16) {
      XorAndSalsa(out, tail, input, ii, out, head);
      if (i > 0)
        tail += 16;
      XorAndSalsa(out, head, input, ii += 16, out, tail);
    }
  }
  function scryptInit(password, salt, _opts) {
    const opts = checkOpts({
      dkLen: 32,
      asyncTick: 10,
      maxmem: 1024 ** 3 + 1024
    }, _opts);
    const { N, r, p, dkLen, asyncTick, maxmem, onProgress } = opts;
    anumber(N);
    anumber(r);
    anumber(p);
    anumber(dkLen);
    anumber(asyncTick);
    anumber(maxmem);
    if (onProgress !== void 0 && typeof onProgress !== "function")
      throw new Error("progressCb should be function");
    const blockSize = 128 * r;
    const blockSize32 = blockSize / 4;
    if (N <= 1 || (N & N - 1) !== 0 || N > 2 ** 32) {
      throw new Error("Scrypt: N must be larger than 1, a power of 2, and less than 2^32");
    }
    if (p < 0 || p > (2 ** 32 - 1) * 32 / blockSize) {
      throw new Error("Scrypt: p must be a positive integer less than or equal to ((2^32 - 1) * 32) / (128 * r)");
    }
    if (dkLen < 0 || dkLen > (2 ** 32 - 1) * 32) {
      throw new Error("Scrypt: dkLen should be positive integer less than or equal to (2^32 - 1) * 32");
    }
    const memUsed = blockSize * (N + p);
    if (memUsed > maxmem) {
      throw new Error("Scrypt: memused is bigger than maxMem. Expected 128 * r * (N + p) > maxmem of " + maxmem);
    }
    const B = pbkdf2(sha256, password, salt, { c: 1, dkLen: blockSize * p });
    const B32 = u32(B);
    const V = u32(new Uint8Array(blockSize * N));
    const tmp = u32(new Uint8Array(blockSize));
    let blockMixCb = () => {
    };
    if (onProgress) {
      const totalBlockMix = 2 * N * p;
      const callbackPer = Math.max(Math.floor(totalBlockMix / 1e4), 1);
      let blockMixCnt = 0;
      blockMixCb = () => {
        blockMixCnt++;
        if (onProgress && (!(blockMixCnt % callbackPer) || blockMixCnt === totalBlockMix))
          onProgress(blockMixCnt / totalBlockMix);
      };
    }
    return { N, r, p, dkLen, blockSize32, V, B32, B, tmp, blockMixCb, asyncTick };
  }
  function scryptOutput(password, dkLen, B, V, tmp) {
    const res = pbkdf2(sha256, password, B, { c: 1, dkLen });
    B.fill(0);
    V.fill(0);
    tmp.fill(0);
    return res;
  }
  function scrypt(password, salt, opts) {
    const { N, r, p, dkLen, blockSize32, V, B32, B, tmp, blockMixCb } = scryptInit(password, salt, opts);
    if (!isLE)
      byteSwap32(B32);
    for (let pi = 0; pi < p; pi++) {
      const Pi = blockSize32 * pi;
      for (let i = 0; i < blockSize32; i++)
        V[i] = B32[Pi + i];
      for (let i = 0, pos = 0; i < N - 1; i++) {
        BlockMix(V, pos, V, pos += blockSize32, r);
        blockMixCb();
      }
      BlockMix(V, (N - 1) * blockSize32, B32, Pi, r);
      blockMixCb();
      for (let i = 0; i < N; i++) {
        const j = B32[Pi + blockSize32 - 16] % N;
        for (let k = 0; k < blockSize32; k++)
          tmp[k] = B32[Pi + k] ^ V[j * blockSize32 + k];
        BlockMix(tmp, 0, B32, Pi, r);
        blockMixCb();
      }
    }
    if (!isLE)
      byteSwap32(B32);
    return scryptOutput(password, dkLen, B, V, tmp);
  }

  // node_modules/@noble/ciphers/esm/_assert.js
  function anumber3(n) {
    if (!Number.isSafeInteger(n) || n < 0)
      throw new Error("positive integer expected, got " + n);
  }
  function isBytes3(a) {
    return a instanceof Uint8Array || ArrayBuffer.isView(a) && a.constructor.name === "Uint8Array";
  }
  function abytes2(b, ...lengths) {
    if (!isBytes3(b))
      throw new Error("Uint8Array expected");
    if (lengths.length > 0 && !lengths.includes(b.length))
      throw new Error("Uint8Array expected of length " + lengths + ", got length=" + b.length);
  }
  function aexists2(instance, checkFinished = true) {
    if (instance.destroyed)
      throw new Error("Hash instance has been destroyed");
    if (checkFinished && instance.finished)
      throw new Error("Hash#digest() has already been called");
  }
  function aoutput2(out, instance) {
    abytes2(out);
    const min = instance.outputLen;
    if (out.length < min) {
      throw new Error("digestInto() expects output buffer of length at least " + min);
    }
  }
  function abool(b) {
    if (typeof b !== "boolean")
      throw new Error(`boolean expected, not ${b}`);
  }

  // node_modules/@noble/ciphers/esm/utils.js
  var u322 = (arr) => new Uint32Array(arr.buffer, arr.byteOffset, Math.floor(arr.byteLength / 4));
  var createView2 = (arr) => new DataView(arr.buffer, arr.byteOffset, arr.byteLength);
  var isLE2 = new Uint8Array(new Uint32Array([287454020]).buffer)[0] === 68;
  if (!isLE2)
    throw new Error("Non little-endian hardware is not supported");
  function utf8ToBytes2(str) {
    if (typeof str !== "string")
      throw new Error("string expected");
    return new Uint8Array(new TextEncoder().encode(str));
  }
  function toBytes2(data) {
    if (typeof data === "string")
      data = utf8ToBytes2(data);
    else if (isBytes3(data))
      data = copyBytes(data);
    else
      throw new Error("Uint8Array expected, got " + typeof data);
    return data;
  }
  function checkOpts2(defaults, opts) {
    if (opts == null || typeof opts !== "object")
      throw new Error("options must be defined");
    const merged = Object.assign(defaults, opts);
    return merged;
  }
  function equalBytes(a, b) {
    if (a.length !== b.length)
      return false;
    let diff = 0;
    for (let i = 0; i < a.length; i++)
      diff |= a[i] ^ b[i];
    return diff === 0;
  }
  var wrapCipher = /* @__NO_SIDE_EFFECTS__ */ (params, constructor) => {
    function wrappedCipher(key, ...args) {
      abytes2(key);
      if (params.nonceLength !== void 0) {
        const nonce = args[0];
        if (!nonce)
          throw new Error("nonce / iv required");
        if (params.varSizeNonce)
          abytes2(nonce);
        else
          abytes2(nonce, params.nonceLength);
      }
      const tagl = params.tagLength;
      if (tagl && args[1] !== void 0) {
        abytes2(args[1]);
      }
      const cipher = constructor(key, ...args);
      const checkOutput = (fnLength, output) => {
        if (output !== void 0) {
          if (fnLength !== 2)
            throw new Error("cipher output not supported");
          abytes2(output);
        }
      };
      let called = false;
      const wrCipher = {
        encrypt(data, output) {
          if (called)
            throw new Error("cannot encrypt() twice with same key + nonce");
          called = true;
          abytes2(data);
          checkOutput(cipher.encrypt.length, output);
          return cipher.encrypt(data, output);
        },
        decrypt(data, output) {
          abytes2(data);
          if (tagl && data.length < tagl)
            throw new Error("invalid ciphertext length: smaller than tagLength=" + tagl);
          checkOutput(cipher.decrypt.length, output);
          return cipher.decrypt(data, output);
        }
      };
      return wrCipher;
    }
    Object.assign(wrappedCipher, params);
    return wrappedCipher;
  };
  function getOutput(expectedLength, out, onlyAligned = true) {
    if (out === void 0)
      return new Uint8Array(expectedLength);
    if (out.length !== expectedLength)
      throw new Error("invalid output length, expected " + expectedLength + ", got: " + out.length);
    if (onlyAligned && !isAligned32(out))
      throw new Error("invalid output, must be aligned");
    return out;
  }
  function setBigUint642(view, byteOffset, value, isLE3) {
    if (typeof view.setBigUint64 === "function")
      return view.setBigUint64(byteOffset, value, isLE3);
    const _32n = BigInt(32);
    const _u32_max = BigInt(4294967295);
    const wh = Number(value >> _32n & _u32_max);
    const wl = Number(value & _u32_max);
    const h = isLE3 ? 4 : 0;
    const l = isLE3 ? 0 : 4;
    view.setUint32(byteOffset + h, wh, isLE3);
    view.setUint32(byteOffset + l, wl, isLE3);
  }
  function isAligned32(bytes) {
    return bytes.byteOffset % 4 === 0;
  }
  function copyBytes(bytes) {
    return Uint8Array.from(bytes);
  }
  function clean(...arrays) {
    for (let i = 0; i < arrays.length; i++) {
      arrays[i].fill(0);
    }
  }

  // node_modules/@noble/ciphers/esm/_arx.js
  var _utf8ToBytes = (str) => Uint8Array.from(str.split("").map((c) => c.charCodeAt(0)));
  var sigma16 = _utf8ToBytes("expand 16-byte k");
  var sigma32 = _utf8ToBytes("expand 32-byte k");
  var sigma16_32 = u322(sigma16);
  var sigma32_32 = u322(sigma32);
  function rotl2(a, b) {
    return a << b | a >>> 32 - b;
  }
  function isAligned322(b) {
    return b.byteOffset % 4 === 0;
  }
  var BLOCK_LEN = 64;
  var BLOCK_LEN32 = 16;
  var MAX_COUNTER = 2 ** 32 - 1;
  var U32_EMPTY = new Uint32Array();
  function runCipher(core, sigma, key, nonce, data, output, counter, rounds) {
    const len = data.length;
    const block = new Uint8Array(BLOCK_LEN);
    const b32 = u322(block);
    const isAligned = isAligned322(data) && isAligned322(output);
    const d32 = isAligned ? u322(data) : U32_EMPTY;
    const o32 = isAligned ? u322(output) : U32_EMPTY;
    for (let pos = 0; pos < len; counter++) {
      core(sigma, key, nonce, b32, counter, rounds);
      if (counter >= MAX_COUNTER)
        throw new Error("arx: counter overflow");
      const take = Math.min(BLOCK_LEN, len - pos);
      if (isAligned && take === BLOCK_LEN) {
        const pos32 = pos / 4;
        if (pos % 4 !== 0)
          throw new Error("arx: invalid block position");
        for (let j = 0, posj; j < BLOCK_LEN32; j++) {
          posj = pos32 + j;
          o32[posj] = d32[posj] ^ b32[j];
        }
        pos += BLOCK_LEN;
        continue;
      }
      for (let j = 0, posj; j < take; j++) {
        posj = pos + j;
        output[posj] = data[posj] ^ block[j];
      }
      pos += take;
    }
  }
  function createCipher(core, opts) {
    const { allowShortKeys, extendNonceFn, counterLength, counterRight, rounds } = checkOpts2({ allowShortKeys: false, counterLength: 8, counterRight: false, rounds: 20 }, opts);
    if (typeof core !== "function")
      throw new Error("core must be a function");
    anumber3(counterLength);
    anumber3(rounds);
    abool(counterRight);
    abool(allowShortKeys);
    return (key, nonce, data, output, counter = 0) => {
      abytes2(key);
      abytes2(nonce);
      abytes2(data);
      const len = data.length;
      if (output === void 0)
        output = new Uint8Array(len);
      abytes2(output);
      anumber3(counter);
      if (counter < 0 || counter >= MAX_COUNTER)
        throw new Error("arx: counter overflow");
      if (output.length < len)
        throw new Error(`arx: output (${output.length}) is shorter than data (${len})`);
      const toClean = [];
      let l = key.length;
      let k;
      let sigma;
      if (l === 32) {
        toClean.push(k = copyBytes(key));
        sigma = sigma32_32;
      } else if (l === 16 && allowShortKeys) {
        k = new Uint8Array(32);
        k.set(key);
        k.set(key, 16);
        sigma = sigma16_32;
        toClean.push(k);
      } else {
        throw new Error(`arx: invalid 32-byte key, got length=${l}`);
      }
      if (!isAligned322(nonce))
        toClean.push(nonce = copyBytes(nonce));
      const k32 = u322(k);
      if (extendNonceFn) {
        if (nonce.length !== 24)
          throw new Error(`arx: extended nonce must be 24 bytes`);
        extendNonceFn(sigma, k32, u322(nonce.subarray(0, 16)), k32);
        nonce = nonce.subarray(16);
      }
      const nonceNcLen = 16 - counterLength;
      if (nonceNcLen !== nonce.length)
        throw new Error(`arx: nonce must be ${nonceNcLen} or 16 bytes`);
      if (nonceNcLen !== 12) {
        const nc = new Uint8Array(12);
        nc.set(nonce, counterRight ? 0 : 12 - nonce.length);
        nonce = nc;
        toClean.push(nonce);
      }
      const n32 = u322(nonce);
      runCipher(core, sigma, k32, n32, data, output, counter, rounds);
      clean(...toClean);
      return output;
    };
  }

  // node_modules/@noble/ciphers/esm/_poly1305.js
  var u8to16 = (a, i) => a[i++] & 255 | (a[i++] & 255) << 8;
  var Poly1305 = class {
    constructor(key) {
      this.blockLen = 16;
      this.outputLen = 16;
      this.buffer = new Uint8Array(16);
      this.r = new Uint16Array(10);
      this.h = new Uint16Array(10);
      this.pad = new Uint16Array(8);
      this.pos = 0;
      this.finished = false;
      key = toBytes2(key);
      abytes2(key, 32);
      const t0 = u8to16(key, 0);
      const t1 = u8to16(key, 2);
      const t2 = u8to16(key, 4);
      const t3 = u8to16(key, 6);
      const t4 = u8to16(key, 8);
      const t5 = u8to16(key, 10);
      const t6 = u8to16(key, 12);
      const t7 = u8to16(key, 14);
      this.r[0] = t0 & 8191;
      this.r[1] = (t0 >>> 13 | t1 << 3) & 8191;
      this.r[2] = (t1 >>> 10 | t2 << 6) & 7939;
      this.r[3] = (t2 >>> 7 | t3 << 9) & 8191;
      this.r[4] = (t3 >>> 4 | t4 << 12) & 255;
      this.r[5] = t4 >>> 1 & 8190;
      this.r[6] = (t4 >>> 14 | t5 << 2) & 8191;
      this.r[7] = (t5 >>> 11 | t6 << 5) & 8065;
      this.r[8] = (t6 >>> 8 | t7 << 8) & 8191;
      this.r[9] = t7 >>> 5 & 127;
      for (let i = 0; i < 8; i++)
        this.pad[i] = u8to16(key, 16 + 2 * i);
    }
    process(data, offset, isLast = false) {
      const hibit = isLast ? 0 : 1 << 11;
      const { h, r } = this;
      const r0 = r[0];
      const r1 = r[1];
      const r2 = r[2];
      const r3 = r[3];
      const r4 = r[4];
      const r5 = r[5];
      const r6 = r[6];
      const r7 = r[7];
      const r8 = r[8];
      const r9 = r[9];
      const t0 = u8to16(data, offset + 0);
      const t1 = u8to16(data, offset + 2);
      const t2 = u8to16(data, offset + 4);
      const t3 = u8to16(data, offset + 6);
      const t4 = u8to16(data, offset + 8);
      const t5 = u8to16(data, offset + 10);
      const t6 = u8to16(data, offset + 12);
      const t7 = u8to16(data, offset + 14);
      let h0 = h[0] + (t0 & 8191);
      let h1 = h[1] + ((t0 >>> 13 | t1 << 3) & 8191);
      let h2 = h[2] + ((t1 >>> 10 | t2 << 6) & 8191);
      let h3 = h[3] + ((t2 >>> 7 | t3 << 9) & 8191);
      let h4 = h[4] + ((t3 >>> 4 | t4 << 12) & 8191);
      let h5 = h[5] + (t4 >>> 1 & 8191);
      let h6 = h[6] + ((t4 >>> 14 | t5 << 2) & 8191);
      let h7 = h[7] + ((t5 >>> 11 | t6 << 5) & 8191);
      let h8 = h[8] + ((t6 >>> 8 | t7 << 8) & 8191);
      let h9 = h[9] + (t7 >>> 5 | hibit);
      let c = 0;
      let d0 = c + h0 * r0 + h1 * (5 * r9) + h2 * (5 * r8) + h3 * (5 * r7) + h4 * (5 * r6);
      c = d0 >>> 13;
      d0 &= 8191;
      d0 += h5 * (5 * r5) + h6 * (5 * r4) + h7 * (5 * r3) + h8 * (5 * r2) + h9 * (5 * r1);
      c += d0 >>> 13;
      d0 &= 8191;
      let d1 = c + h0 * r1 + h1 * r0 + h2 * (5 * r9) + h3 * (5 * r8) + h4 * (5 * r7);
      c = d1 >>> 13;
      d1 &= 8191;
      d1 += h5 * (5 * r6) + h6 * (5 * r5) + h7 * (5 * r4) + h8 * (5 * r3) + h9 * (5 * r2);
      c += d1 >>> 13;
      d1 &= 8191;
      let d2 = c + h0 * r2 + h1 * r1 + h2 * r0 + h3 * (5 * r9) + h4 * (5 * r8);
      c = d2 >>> 13;
      d2 &= 8191;
      d2 += h5 * (5 * r7) + h6 * (5 * r6) + h7 * (5 * r5) + h8 * (5 * r4) + h9 * (5 * r3);
      c += d2 >>> 13;
      d2 &= 8191;
      let d3 = c + h0 * r3 + h1 * r2 + h2 * r1 + h3 * r0 + h4 * (5 * r9);
      c = d3 >>> 13;
      d3 &= 8191;
      d3 += h5 * (5 * r8) + h6 * (5 * r7) + h7 * (5 * r6) + h8 * (5 * r5) + h9 * (5 * r4);
      c += d3 >>> 13;
      d3 &= 8191;
      let d4 = c + h0 * r4 + h1 * r3 + h2 * r2 + h3 * r1 + h4 * r0;
      c = d4 >>> 13;
      d4 &= 8191;
      d4 += h5 * (5 * r9) + h6 * (5 * r8) + h7 * (5 * r7) + h8 * (5 * r6) + h9 * (5 * r5);
      c += d4 >>> 13;
      d4 &= 8191;
      let d5 = c + h0 * r5 + h1 * r4 + h2 * r3 + h3 * r2 + h4 * r1;
      c = d5 >>> 13;
      d5 &= 8191;
      d5 += h5 * r0 + h6 * (5 * r9) + h7 * (5 * r8) + h8 * (5 * r7) + h9 * (5 * r6);
      c += d5 >>> 13;
      d5 &= 8191;
      let d6 = c + h0 * r6 + h1 * r5 + h2 * r4 + h3 * r3 + h4 * r2;
      c = d6 >>> 13;
      d6 &= 8191;
      d6 += h5 * r1 + h6 * r0 + h7 * (5 * r9) + h8 * (5 * r8) + h9 * (5 * r7);
      c += d6 >>> 13;
      d6 &= 8191;
      let d7 = c + h0 * r7 + h1 * r6 + h2 * r5 + h3 * r4 + h4 * r3;
      c = d7 >>> 13;
      d7 &= 8191;
      d7 += h5 * r2 + h6 * r1 + h7 * r0 + h8 * (5 * r9) + h9 * (5 * r8);
      c += d7 >>> 13;
      d7 &= 8191;
      let d8 = c + h0 * r8 + h1 * r7 + h2 * r6 + h3 * r5 + h4 * r4;
      c = d8 >>> 13;
      d8 &= 8191;
      d8 += h5 * r3 + h6 * r2 + h7 * r1 + h8 * r0 + h9 * (5 * r9);
      c += d8 >>> 13;
      d8 &= 8191;
      let d9 = c + h0 * r9 + h1 * r8 + h2 * r7 + h3 * r6 + h4 * r5;
      c = d9 >>> 13;
      d9 &= 8191;
      d9 += h5 * r4 + h6 * r3 + h7 * r2 + h8 * r1 + h9 * r0;
      c += d9 >>> 13;
      d9 &= 8191;
      c = (c << 2) + c | 0;
      c = c + d0 | 0;
      d0 = c & 8191;
      c = c >>> 13;
      d1 += c;
      h[0] = d0;
      h[1] = d1;
      h[2] = d2;
      h[3] = d3;
      h[4] = d4;
      h[5] = d5;
      h[6] = d6;
      h[7] = d7;
      h[8] = d8;
      h[9] = d9;
    }
    finalize() {
      const { h, pad } = this;
      const g = new Uint16Array(10);
      let c = h[1] >>> 13;
      h[1] &= 8191;
      for (let i = 2; i < 10; i++) {
        h[i] += c;
        c = h[i] >>> 13;
        h[i] &= 8191;
      }
      h[0] += c * 5;
      c = h[0] >>> 13;
      h[0] &= 8191;
      h[1] += c;
      c = h[1] >>> 13;
      h[1] &= 8191;
      h[2] += c;
      g[0] = h[0] + 5;
      c = g[0] >>> 13;
      g[0] &= 8191;
      for (let i = 1; i < 10; i++) {
        g[i] = h[i] + c;
        c = g[i] >>> 13;
        g[i] &= 8191;
      }
      g[9] -= 1 << 13;
      let mask = (c ^ 1) - 1;
      for (let i = 0; i < 10; i++)
        g[i] &= mask;
      mask = ~mask;
      for (let i = 0; i < 10; i++)
        h[i] = h[i] & mask | g[i];
      h[0] = (h[0] | h[1] << 13) & 65535;
      h[1] = (h[1] >>> 3 | h[2] << 10) & 65535;
      h[2] = (h[2] >>> 6 | h[3] << 7) & 65535;
      h[3] = (h[3] >>> 9 | h[4] << 4) & 65535;
      h[4] = (h[4] >>> 12 | h[5] << 1 | h[6] << 14) & 65535;
      h[5] = (h[6] >>> 2 | h[7] << 11) & 65535;
      h[6] = (h[7] >>> 5 | h[8] << 8) & 65535;
      h[7] = (h[8] >>> 8 | h[9] << 5) & 65535;
      let f = h[0] + pad[0];
      h[0] = f & 65535;
      for (let i = 1; i < 8; i++) {
        f = (h[i] + pad[i] | 0) + (f >>> 16) | 0;
        h[i] = f & 65535;
      }
      clean(g);
    }
    update(data) {
      aexists2(this);
      const { buffer, blockLen } = this;
      data = toBytes2(data);
      const len = data.length;
      for (let pos = 0; pos < len; ) {
        const take = Math.min(blockLen - this.pos, len - pos);
        if (take === blockLen) {
          for (; blockLen <= len - pos; pos += blockLen)
            this.process(data, pos);
          continue;
        }
        buffer.set(data.subarray(pos, pos + take), this.pos);
        this.pos += take;
        pos += take;
        if (this.pos === blockLen) {
          this.process(buffer, 0, false);
          this.pos = 0;
        }
      }
      return this;
    }
    destroy() {
      clean(this.h, this.r, this.buffer, this.pad);
    }
    digestInto(out) {
      aexists2(this);
      aoutput2(out, this);
      this.finished = true;
      const { buffer, h } = this;
      let { pos } = this;
      if (pos) {
        buffer[pos++] = 1;
        for (; pos < 16; pos++)
          buffer[pos] = 0;
        this.process(buffer, 0, true);
      }
      this.finalize();
      let opos = 0;
      for (let i = 0; i < 8; i++) {
        out[opos++] = h[i] >>> 0;
        out[opos++] = h[i] >>> 8;
      }
      return out;
    }
    digest() {
      const { buffer, outputLen } = this;
      this.digestInto(buffer);
      const res = buffer.slice(0, outputLen);
      this.destroy();
      return res;
    }
  };
  function wrapConstructorWithKey(hashCons) {
    const hashC = (msg, key) => hashCons(key).update(toBytes2(msg)).digest();
    const tmp = hashCons(new Uint8Array(32));
    hashC.outputLen = tmp.outputLen;
    hashC.blockLen = tmp.blockLen;
    hashC.create = (key) => hashCons(key);
    return hashC;
  }
  var poly1305 = wrapConstructorWithKey((key) => new Poly1305(key));

  // node_modules/@noble/ciphers/esm/chacha.js
  function chachaCore(s, k, n, out, cnt, rounds = 20) {
    let y00 = s[0], y01 = s[1], y02 = s[2], y03 = s[3], y04 = k[0], y05 = k[1], y06 = k[2], y07 = k[3], y08 = k[4], y09 = k[5], y10 = k[6], y11 = k[7], y12 = cnt, y13 = n[0], y14 = n[1], y15 = n[2];
    let x00 = y00, x01 = y01, x02 = y02, x03 = y03, x04 = y04, x05 = y05, x06 = y06, x07 = y07, x08 = y08, x09 = y09, x10 = y10, x11 = y11, x12 = y12, x13 = y13, x14 = y14, x15 = y15;
    for (let r = 0; r < rounds; r += 2) {
      x00 = x00 + x04 | 0;
      x12 = rotl2(x12 ^ x00, 16);
      x08 = x08 + x12 | 0;
      x04 = rotl2(x04 ^ x08, 12);
      x00 = x00 + x04 | 0;
      x12 = rotl2(x12 ^ x00, 8);
      x08 = x08 + x12 | 0;
      x04 = rotl2(x04 ^ x08, 7);
      x01 = x01 + x05 | 0;
      x13 = rotl2(x13 ^ x01, 16);
      x09 = x09 + x13 | 0;
      x05 = rotl2(x05 ^ x09, 12);
      x01 = x01 + x05 | 0;
      x13 = rotl2(x13 ^ x01, 8);
      x09 = x09 + x13 | 0;
      x05 = rotl2(x05 ^ x09, 7);
      x02 = x02 + x06 | 0;
      x14 = rotl2(x14 ^ x02, 16);
      x10 = x10 + x14 | 0;
      x06 = rotl2(x06 ^ x10, 12);
      x02 = x02 + x06 | 0;
      x14 = rotl2(x14 ^ x02, 8);
      x10 = x10 + x14 | 0;
      x06 = rotl2(x06 ^ x10, 7);
      x03 = x03 + x07 | 0;
      x15 = rotl2(x15 ^ x03, 16);
      x11 = x11 + x15 | 0;
      x07 = rotl2(x07 ^ x11, 12);
      x03 = x03 + x07 | 0;
      x15 = rotl2(x15 ^ x03, 8);
      x11 = x11 + x15 | 0;
      x07 = rotl2(x07 ^ x11, 7);
      x00 = x00 + x05 | 0;
      x15 = rotl2(x15 ^ x00, 16);
      x10 = x10 + x15 | 0;
      x05 = rotl2(x05 ^ x10, 12);
      x00 = x00 + x05 | 0;
      x15 = rotl2(x15 ^ x00, 8);
      x10 = x10 + x15 | 0;
      x05 = rotl2(x05 ^ x10, 7);
      x01 = x01 + x06 | 0;
      x12 = rotl2(x12 ^ x01, 16);
      x11 = x11 + x12 | 0;
      x06 = rotl2(x06 ^ x11, 12);
      x01 = x01 + x06 | 0;
      x12 = rotl2(x12 ^ x01, 8);
      x11 = x11 + x12 | 0;
      x06 = rotl2(x06 ^ x11, 7);
      x02 = x02 + x07 | 0;
      x13 = rotl2(x13 ^ x02, 16);
      x08 = x08 + x13 | 0;
      x07 = rotl2(x07 ^ x08, 12);
      x02 = x02 + x07 | 0;
      x13 = rotl2(x13 ^ x02, 8);
      x08 = x08 + x13 | 0;
      x07 = rotl2(x07 ^ x08, 7);
      x03 = x03 + x04 | 0;
      x14 = rotl2(x14 ^ x03, 16);
      x09 = x09 + x14 | 0;
      x04 = rotl2(x04 ^ x09, 12);
      x03 = x03 + x04 | 0;
      x14 = rotl2(x14 ^ x03, 8);
      x09 = x09 + x14 | 0;
      x04 = rotl2(x04 ^ x09, 7);
    }
    let oi = 0;
    out[oi++] = y00 + x00 | 0;
    out[oi++] = y01 + x01 | 0;
    out[oi++] = y02 + x02 | 0;
    out[oi++] = y03 + x03 | 0;
    out[oi++] = y04 + x04 | 0;
    out[oi++] = y05 + x05 | 0;
    out[oi++] = y06 + x06 | 0;
    out[oi++] = y07 + x07 | 0;
    out[oi++] = y08 + x08 | 0;
    out[oi++] = y09 + x09 | 0;
    out[oi++] = y10 + x10 | 0;
    out[oi++] = y11 + x11 | 0;
    out[oi++] = y12 + x12 | 0;
    out[oi++] = y13 + x13 | 0;
    out[oi++] = y14 + x14 | 0;
    out[oi++] = y15 + x15 | 0;
  }
  function hchacha(s, k, i, o32) {
    let x00 = s[0], x01 = s[1], x02 = s[2], x03 = s[3], x04 = k[0], x05 = k[1], x06 = k[2], x07 = k[3], x08 = k[4], x09 = k[5], x10 = k[6], x11 = k[7], x12 = i[0], x13 = i[1], x14 = i[2], x15 = i[3];
    for (let r = 0; r < 20; r += 2) {
      x00 = x00 + x04 | 0;
      x12 = rotl2(x12 ^ x00, 16);
      x08 = x08 + x12 | 0;
      x04 = rotl2(x04 ^ x08, 12);
      x00 = x00 + x04 | 0;
      x12 = rotl2(x12 ^ x00, 8);
      x08 = x08 + x12 | 0;
      x04 = rotl2(x04 ^ x08, 7);
      x01 = x01 + x05 | 0;
      x13 = rotl2(x13 ^ x01, 16);
      x09 = x09 + x13 | 0;
      x05 = rotl2(x05 ^ x09, 12);
      x01 = x01 + x05 | 0;
      x13 = rotl2(x13 ^ x01, 8);
      x09 = x09 + x13 | 0;
      x05 = rotl2(x05 ^ x09, 7);
      x02 = x02 + x06 | 0;
      x14 = rotl2(x14 ^ x02, 16);
      x10 = x10 + x14 | 0;
      x06 = rotl2(x06 ^ x10, 12);
      x02 = x02 + x06 | 0;
      x14 = rotl2(x14 ^ x02, 8);
      x10 = x10 + x14 | 0;
      x06 = rotl2(x06 ^ x10, 7);
      x03 = x03 + x07 | 0;
      x15 = rotl2(x15 ^ x03, 16);
      x11 = x11 + x15 | 0;
      x07 = rotl2(x07 ^ x11, 12);
      x03 = x03 + x07 | 0;
      x15 = rotl2(x15 ^ x03, 8);
      x11 = x11 + x15 | 0;
      x07 = rotl2(x07 ^ x11, 7);
      x00 = x00 + x05 | 0;
      x15 = rotl2(x15 ^ x00, 16);
      x10 = x10 + x15 | 0;
      x05 = rotl2(x05 ^ x10, 12);
      x00 = x00 + x05 | 0;
      x15 = rotl2(x15 ^ x00, 8);
      x10 = x10 + x15 | 0;
      x05 = rotl2(x05 ^ x10, 7);
      x01 = x01 + x06 | 0;
      x12 = rotl2(x12 ^ x01, 16);
      x11 = x11 + x12 | 0;
      x06 = rotl2(x06 ^ x11, 12);
      x01 = x01 + x06 | 0;
      x12 = rotl2(x12 ^ x01, 8);
      x11 = x11 + x12 | 0;
      x06 = rotl2(x06 ^ x11, 7);
      x02 = x02 + x07 | 0;
      x13 = rotl2(x13 ^ x02, 16);
      x08 = x08 + x13 | 0;
      x07 = rotl2(x07 ^ x08, 12);
      x02 = x02 + x07 | 0;
      x13 = rotl2(x13 ^ x02, 8);
      x08 = x08 + x13 | 0;
      x07 = rotl2(x07 ^ x08, 7);
      x03 = x03 + x04 | 0;
      x14 = rotl2(x14 ^ x03, 16);
      x09 = x09 + x14 | 0;
      x04 = rotl2(x04 ^ x09, 12);
      x03 = x03 + x04 | 0;
      x14 = rotl2(x14 ^ x03, 8);
      x09 = x09 + x14 | 0;
      x04 = rotl2(x04 ^ x09, 7);
    }
    let oi = 0;
    o32[oi++] = x00;
    o32[oi++] = x01;
    o32[oi++] = x02;
    o32[oi++] = x03;
    o32[oi++] = x12;
    o32[oi++] = x13;
    o32[oi++] = x14;
    o32[oi++] = x15;
  }
  var chacha20 = /* @__PURE__ */ createCipher(chachaCore, {
    counterRight: false,
    counterLength: 4,
    allowShortKeys: false
  });
  var xchacha20 = /* @__PURE__ */ createCipher(chachaCore, {
    counterRight: false,
    counterLength: 8,
    extendNonceFn: hchacha,
    allowShortKeys: false
  });
  var ZEROS16 = /* @__PURE__ */ new Uint8Array(16);
  var updatePadded = (h, msg) => {
    h.update(msg);
    const left = msg.length % 16;
    if (left)
      h.update(ZEROS16.subarray(left));
  };
  var ZEROS32 = /* @__PURE__ */ new Uint8Array(32);
  function computeTag(fn, key, nonce, data, AAD) {
    const authKey = fn(key, nonce, ZEROS32);
    const h = poly1305.create(authKey);
    if (AAD)
      updatePadded(h, AAD);
    updatePadded(h, data);
    const num = new Uint8Array(16);
    const view = createView2(num);
    setBigUint642(view, 0, BigInt(AAD ? AAD.length : 0), true);
    setBigUint642(view, 8, BigInt(data.length), true);
    h.update(num);
    const res = h.digest();
    clean(authKey, num);
    return res;
  }
  var _poly1305_aead = (xorStream) => (key, nonce, AAD) => {
    const tagLength = 16;
    return {
      encrypt(plaintext, output) {
        const plength = plaintext.length;
        output = getOutput(plength + tagLength, output, false);
        output.set(plaintext);
        const oPlain = output.subarray(0, -tagLength);
        xorStream(key, nonce, oPlain, oPlain, 1);
        const tag = computeTag(xorStream, key, nonce, oPlain, AAD);
        output.set(tag, plength);
        clean(tag);
        return output;
      },
      decrypt(ciphertext, output) {
        output = getOutput(ciphertext.length - tagLength, output, false);
        const data = ciphertext.subarray(0, -tagLength);
        const passedTag = ciphertext.subarray(-tagLength);
        const tag = computeTag(xorStream, key, nonce, data, AAD);
        if (!equalBytes(passedTag, tag))
          throw new Error("invalid tag");
        output.set(ciphertext.subarray(0, -tagLength));
        xorStream(key, nonce, output, output, 1);
        clean(tag);
        return output;
      }
    };
  };
  var chacha20poly1305 = /* @__PURE__ */ wrapCipher({ blockSize: 64, nonceLength: 12, tagLength: 16 }, _poly1305_aead(chacha20));
  var xchacha20poly1305 = /* @__PURE__ */ wrapCipher({ blockSize: 64, nonceLength: 24, tagLength: 16 }, _poly1305_aead(xchacha20));

  // node_modules/@noble/curves/esm/abstract/utils.js
  var _0n = /* @__PURE__ */ BigInt(0);
  function isBytes4(a) {
    return a instanceof Uint8Array || ArrayBuffer.isView(a) && a.constructor.name === "Uint8Array";
  }
  function abytes3(item) {
    if (!isBytes4(item))
      throw new Error("Uint8Array expected");
  }
  var hexes = /* @__PURE__ */ Array.from({ length: 256 }, (_, i) => i.toString(16).padStart(2, "0"));
  function bytesToHex(bytes) {
    abytes3(bytes);
    let hex = "";
    for (let i = 0; i < bytes.length; i++) {
      hex += hexes[bytes[i]];
    }
    return hex;
  }
  function hexToNumber(hex) {
    if (typeof hex !== "string")
      throw new Error("hex string expected, got " + typeof hex);
    return hex === "" ? _0n : BigInt("0x" + hex);
  }
  var asciis = { _0: 48, _9: 57, A: 65, F: 70, a: 97, f: 102 };
  function asciiToBase16(ch) {
    if (ch >= asciis._0 && ch <= asciis._9)
      return ch - asciis._0;
    if (ch >= asciis.A && ch <= asciis.F)
      return ch - (asciis.A - 10);
    if (ch >= asciis.a && ch <= asciis.f)
      return ch - (asciis.a - 10);
    return;
  }
  function hexToBytes(hex) {
    if (typeof hex !== "string")
      throw new Error("hex string expected, got " + typeof hex);
    const hl = hex.length;
    const al = hl / 2;
    if (hl % 2)
      throw new Error("hex string expected, got unpadded hex of length " + hl);
    const array = new Uint8Array(al);
    for (let ai = 0, hi = 0; ai < al; ai++, hi += 2) {
      const n1 = asciiToBase16(hex.charCodeAt(hi));
      const n2 = asciiToBase16(hex.charCodeAt(hi + 1));
      if (n1 === void 0 || n2 === void 0) {
        const char = hex[hi] + hex[hi + 1];
        throw new Error('hex string expected, got non-hex character "' + char + '" at index ' + hi);
      }
      array[ai] = n1 * 16 + n2;
    }
    return array;
  }
  function bytesToNumberLE(bytes) {
    abytes3(bytes);
    return hexToNumber(bytesToHex(Uint8Array.from(bytes).reverse()));
  }
  function numberToBytesBE(n, len) {
    return hexToBytes(n.toString(16).padStart(len * 2, "0"));
  }
  function numberToBytesLE(n, len) {
    return numberToBytesBE(n, len).reverse();
  }
  function ensureBytes(title, hex, expectedLength) {
    let res;
    if (typeof hex === "string") {
      try {
        res = hexToBytes(hex);
      } catch (e) {
        throw new Error(title + " must be hex string or Uint8Array, cause: " + e);
      }
    } else if (isBytes4(hex)) {
      res = Uint8Array.from(hex);
    } else {
      throw new Error(title + " must be hex string or Uint8Array");
    }
    const len = res.length;
    if (typeof expectedLength === "number" && len !== expectedLength)
      throw new Error(title + " of length " + expectedLength + " expected, got " + len);
    return res;
  }
  var isPosBig = (n) => typeof n === "bigint" && _0n <= n;
  function inRange(n, min, max) {
    return isPosBig(n) && isPosBig(min) && isPosBig(max) && min <= n && n < max;
  }
  function aInRange(title, n, min, max) {
    if (!inRange(n, min, max))
      throw new Error("expected valid " + title + ": " + min + " <= n < " + max + ", got " + n);
  }
  var validatorFns = {
    bigint: (val) => typeof val === "bigint",
    function: (val) => typeof val === "function",
    boolean: (val) => typeof val === "boolean",
    string: (val) => typeof val === "string",
    stringOrUint8Array: (val) => typeof val === "string" || isBytes4(val),
    isSafeInteger: (val) => Number.isSafeInteger(val),
    array: (val) => Array.isArray(val),
    field: (val, object) => object.Fp.isValid(val),
    hash: (val) => typeof val === "function" && Number.isSafeInteger(val.outputLen)
  };
  function validateObject(object, validators, optValidators = {}) {
    const checkField = (fieldName, type, isOptional) => {
      const checkVal = validatorFns[type];
      if (typeof checkVal !== "function")
        throw new Error("invalid validator function");
      const val = object[fieldName];
      if (isOptional && val === void 0)
        return;
      if (!checkVal(val, object)) {
        throw new Error("param " + String(fieldName) + " is invalid. Expected " + type + ", got " + val);
      }
    };
    for (const [fieldName, type] of Object.entries(validators))
      checkField(fieldName, type, false);
    for (const [fieldName, type] of Object.entries(optValidators))
      checkField(fieldName, type, true);
    return object;
  }

  // node_modules/@noble/curves/esm/abstract/modular.js
  var _0n2 = BigInt(0);
  var _1n = BigInt(1);
  function mod(a, b) {
    const result = a % b;
    return result >= _0n2 ? result : b + result;
  }
  function pow(num, power, modulo) {
    if (power < _0n2)
      throw new Error("invalid exponent, negatives unsupported");
    if (modulo <= _0n2)
      throw new Error("invalid modulus");
    if (modulo === _1n)
      return _0n2;
    let res = _1n;
    while (power > _0n2) {
      if (power & _1n)
        res = res * num % modulo;
      num = num * num % modulo;
      power >>= _1n;
    }
    return res;
  }
  function pow2(x, power, modulo) {
    let res = x;
    while (power-- > _0n2) {
      res *= res;
      res %= modulo;
    }
    return res;
  }

  // node_modules/@noble/curves/esm/abstract/montgomery.js
  var _0n3 = BigInt(0);
  var _1n2 = BigInt(1);
  function validateOpts(curve) {
    validateObject(curve, {
      a: "bigint"
    }, {
      montgomeryBits: "isSafeInteger",
      nByteLength: "isSafeInteger",
      adjustScalarBytes: "function",
      domain: "function",
      powPminus2: "function",
      Gu: "bigint"
    });
    return Object.freeze({ ...curve });
  }
  function montgomery(curveDef) {
    const CURVE = validateOpts(curveDef);
    const { P } = CURVE;
    const modP = (n) => mod(n, P);
    const montgomeryBits = CURVE.montgomeryBits;
    const montgomeryBytes = Math.ceil(montgomeryBits / 8);
    const fieldLen = CURVE.nByteLength;
    const adjustScalarBytes2 = CURVE.adjustScalarBytes || ((bytes) => bytes);
    const powPminus2 = CURVE.powPminus2 || ((x) => pow(x, P - BigInt(2), P));
    function cswap(swap, x_2, x_3) {
      const dummy = modP(swap * (x_2 - x_3));
      x_2 = modP(x_2 - dummy);
      x_3 = modP(x_3 + dummy);
      return [x_2, x_3];
    }
    const a24 = (CURVE.a - BigInt(2)) / BigInt(4);
    function montgomeryLadder(u, scalar) {
      aInRange("u", u, _0n3, P);
      aInRange("scalar", scalar, _0n3, P);
      const k = scalar;
      const x_1 = u;
      let x_2 = _1n2;
      let z_2 = _0n3;
      let x_3 = u;
      let z_3 = _1n2;
      let swap = _0n3;
      let sw;
      for (let t = BigInt(montgomeryBits - 1); t >= _0n3; t--) {
        const k_t = k >> t & _1n2;
        swap ^= k_t;
        sw = cswap(swap, x_2, x_3);
        x_2 = sw[0];
        x_3 = sw[1];
        sw = cswap(swap, z_2, z_3);
        z_2 = sw[0];
        z_3 = sw[1];
        swap = k_t;
        const A = x_2 + z_2;
        const AA = modP(A * A);
        const B = x_2 - z_2;
        const BB = modP(B * B);
        const E = AA - BB;
        const C = x_3 + z_3;
        const D = x_3 - z_3;
        const DA = modP(D * A);
        const CB = modP(C * B);
        const dacb = DA + CB;
        const da_cb = DA - CB;
        x_3 = modP(dacb * dacb);
        z_3 = modP(x_1 * modP(da_cb * da_cb));
        x_2 = modP(AA * BB);
        z_2 = modP(E * (AA + modP(a24 * E)));
      }
      sw = cswap(swap, x_2, x_3);
      x_2 = sw[0];
      x_3 = sw[1];
      sw = cswap(swap, z_2, z_3);
      z_2 = sw[0];
      z_3 = sw[1];
      const z2 = powPminus2(z_2);
      return modP(x_2 * z2);
    }
    function encodeUCoordinate(u) {
      return numberToBytesLE(modP(u), montgomeryBytes);
    }
    function decodeUCoordinate(uEnc) {
      const u = ensureBytes("u coordinate", uEnc, montgomeryBytes);
      if (fieldLen === 32)
        u[31] &= 127;
      return bytesToNumberLE(u);
    }
    function decodeScalar(n) {
      const bytes = ensureBytes("scalar", n);
      const len = bytes.length;
      if (len !== montgomeryBytes && len !== fieldLen) {
        let valid = "" + montgomeryBytes + " or " + fieldLen;
        throw new Error("invalid scalar, expected " + valid + " bytes, got " + len);
      }
      return bytesToNumberLE(adjustScalarBytes2(bytes));
    }
    function scalarMult2(scalar, u) {
      const pointU = decodeUCoordinate(u);
      const _scalar = decodeScalar(scalar);
      const pu = montgomeryLadder(pointU, _scalar);
      if (pu === _0n3)
        throw new Error("invalid private or public key received");
      return encodeUCoordinate(pu);
    }
    const GuBytes = encodeUCoordinate(CURVE.Gu);
    function scalarMultBase2(scalar) {
      return scalarMult2(scalar, GuBytes);
    }
    return {
      scalarMult: scalarMult2,
      scalarMultBase: scalarMultBase2,
      getSharedSecret: (privateKey, publicKey) => scalarMult2(privateKey, publicKey),
      getPublicKey: (privateKey) => scalarMultBase2(privateKey),
      utils: { randomPrivateKey: () => CURVE.randomBytes(CURVE.nByteLength) },
      GuBytes
    };
  }

  // node_modules/@noble/curves/esm/ed25519.js
  var ED25519_P = BigInt("57896044618658097711785492504343953926634992332820282019728792003956564819949");
  var _0n4 = BigInt(0);
  var _1n3 = BigInt(1);
  var _2n = BigInt(2);
  var _3n = BigInt(3);
  var _5n = BigInt(5);
  var _8n = BigInt(8);
  function ed25519_pow_2_252_3(x) {
    const _10n = BigInt(10), _20n = BigInt(20), _40n = BigInt(40), _80n = BigInt(80);
    const P = ED25519_P;
    const x2 = x * x % P;
    const b2 = x2 * x % P;
    const b4 = pow2(b2, _2n, P) * b2 % P;
    const b5 = pow2(b4, _1n3, P) * x % P;
    const b10 = pow2(b5, _5n, P) * b5 % P;
    const b20 = pow2(b10, _10n, P) * b10 % P;
    const b40 = pow2(b20, _20n, P) * b20 % P;
    const b80 = pow2(b40, _40n, P) * b40 % P;
    const b160 = pow2(b80, _80n, P) * b80 % P;
    const b240 = pow2(b160, _80n, P) * b80 % P;
    const b250 = pow2(b240, _10n, P) * b10 % P;
    const pow_p_5_8 = pow2(b250, _2n, P) * x % P;
    return { pow_p_5_8, b2 };
  }
  function adjustScalarBytes(bytes) {
    bytes[0] &= 248;
    bytes[31] &= 127;
    bytes[31] |= 64;
    return bytes;
  }
  var x25519 = /* @__PURE__ */ (() => montgomery({
    P: ED25519_P,
    a: BigInt(486662),
    montgomeryBits: 255,
    // n is 253 bits
    nByteLength: 32,
    Gu: BigInt(9),
    powPminus2: (x) => {
      const P = ED25519_P;
      const { pow_p_5_8, b2 } = ed25519_pow_2_252_3(x);
      return mod(pow2(pow_p_5_8, _3n, P) * b2, P);
    },
    adjustScalarBytes,
    randomBytes
  }))();

  // dist/x25519.js
  var exportable = false;
  var webCryptoOff = false;
  var isX25519Supported = /* @__PURE__ */ (() => {
    let supported;
    return async () => {
      if (supported === void 0) {
        try {
          await crypto.subtle.importKey("raw", x25519.GuBytes, { name: "X25519" }, exportable, []);
          supported = true;
        } catch {
          supported = false;
        }
      }
      return supported;
    };
  })();
  async function scalarMult(scalar, u) {
    if (!await isX25519Supported() || webCryptoOff) {
      if (isCryptoKey(scalar)) {
        throw new Error("CryptoKey provided but X25519 WebCrypto is not supported");
      }
      return x25519.scalarMult(scalar, u);
    }
    let key;
    if (isCryptoKey(scalar)) {
      key = scalar;
    } else {
      key = await importX25519Key(scalar);
    }
    const peer = await crypto.subtle.importKey("raw", u, { name: "X25519" }, exportable, []);
    return new Uint8Array(await crypto.subtle.deriveBits({ name: "X25519", public: peer }, key, 256));
  }
  async function scalarMultBase(scalar) {
    if (!await isX25519Supported() || webCryptoOff) {
      if (isCryptoKey(scalar)) {
        throw new Error("CryptoKey provided but X25519 WebCrypto is not supported");
      }
      return x25519.scalarMultBase(scalar);
    }
    return scalarMult(scalar, x25519.GuBytes);
  }
  var pkcs8Prefix = /* @__PURE__ */ new Uint8Array([
    48,
    46,
    2,
    1,
    0,
    48,
    5,
    6,
    3,
    43,
    101,
    110,
    4,
    34,
    4,
    32
  ]);
  async function importX25519Key(key) {
    if (key.length !== 32) {
      throw new Error("X25519 private key must be 32 bytes");
    }
    const pkcs8 = new Uint8Array([...pkcs8Prefix, ...key]);
    return crypto.subtle.importKey("pkcs8", pkcs8, { name: "X25519" }, exportable, ["deriveBits"]);
  }
  function isCryptoKey(key) {
    return typeof CryptoKey !== "undefined" && key instanceof CryptoKey;
  }

  // dist/format.js
  var Stanza = class {
    /**
     * All space-separated arguments on the first line of the stanza.
     * Each argument is a string that does not contain spaces.
     * The first argument is often a recipient type, which should look like
     * `example.com/...` to avoid collisions.
     */
    args;
    /**
     * The raw body of the stanza. This is automatically base64-encoded and
     * split into lines of 48 characters each.
     */
    body;
    constructor(args, body) {
      this.args = args;
      this.body = body;
    }
  };
  var ByteReader = class {
    arr;
    constructor(arr) {
      this.arr = arr;
    }
    toString(bytes) {
      bytes.forEach((b) => {
        if (b < 32 || b > 136) {
          throw Error("invalid non-ASCII byte in header");
        }
      });
      return new TextDecoder().decode(bytes);
    }
    readString(n) {
      const out = this.arr.subarray(0, n);
      this.arr = this.arr.subarray(n);
      return this.toString(out);
    }
    readLine() {
      const i = this.arr.indexOf("\n".charCodeAt(0));
      if (i >= 0) {
        const out = this.arr.subarray(0, i);
        this.arr = this.arr.subarray(i + 1);
        return this.toString(out);
      }
      return null;
    }
    rest() {
      return this.arr;
    }
  };
  function parseNextStanza(header) {
    const hdr = new ByteReader(header);
    if (hdr.readString(3) !== "-> ") {
      throw Error("invalid stanza");
    }
    const argsLine = hdr.readLine();
    if (argsLine === null) {
      throw Error("invalid stanza");
    }
    const args = argsLine.split(" ");
    if (args.length < 1) {
      throw Error("invalid stanza");
    }
    for (const arg of args) {
      if (arg.length === 0) {
        throw Error("invalid stanza");
      }
    }
    const bodyLines = [];
    for (; ; ) {
      const nextLine = hdr.readLine();
      if (nextLine === null) {
        throw Error("invalid stanza");
      }
      const line = base64nopad.decode(nextLine);
      if (line.length > 48) {
        throw Error("invalid stanza");
      }
      bodyLines.push(line);
      if (line.length < 48) {
        break;
      }
    }
    const body = flattenArray(bodyLines);
    return [new Stanza(args, body), hdr.rest()];
  }
  function flattenArray(arr) {
    const len = arr.reduce((sum, line) => sum + line.length, 0);
    const out = new Uint8Array(len);
    let n = 0;
    for (const a of arr) {
      out.set(a, n);
      n += a.length;
    }
    return out;
  }
  function parseHeader(header) {
    const hdr = new ByteReader(header);
    const versionLine = hdr.readLine();
    if (versionLine !== "age-encryption.org/v1") {
      throw Error("invalid version " + (versionLine ?? "line"));
    }
    let rest = hdr.rest();
    const stanzas = [];
    for (; ; ) {
      let s;
      [s, rest] = parseNextStanza(rest);
      stanzas.push(s);
      const hdr2 = new ByteReader(rest);
      if (hdr2.readString(4) === "--- ") {
        const headerNoMAC = header.subarray(0, header.length - hdr2.rest().length - 1);
        const macLine = hdr2.readLine();
        if (macLine === null) {
          throw Error("invalid header");
        }
        const mac = base64nopad.decode(macLine);
        return {
          stanzas,
          headerNoMAC,
          MAC: mac,
          rest: hdr2.rest()
        };
      }
    }
  }
  function encodeHeaderNoMAC(recipients) {
    const lines = [];
    lines.push("age-encryption.org/v1\n");
    for (const s of recipients) {
      lines.push("-> " + s.args.join(" ") + "\n");
      for (let i = 0; i < s.body.length; i += 48) {
        let end = i + 48;
        if (end > s.body.length)
          end = s.body.length;
        lines.push(base64nopad.encode(s.body.subarray(i, end)) + "\n");
      }
      if (s.body.length % 48 === 0)
        lines.push("\n");
    }
    lines.push("---");
    return new TextEncoder().encode(lines.join(""));
  }
  function encodeHeader(recipients, MAC) {
    return flattenArray([
      encodeHeaderNoMAC(recipients),
      new TextEncoder().encode(" " + base64nopad.encode(MAC) + "\n")
    ]);
  }

  // dist/recipients.js
  function generateIdentity() {
    const scalar = randomBytes(32);
    const identity = bech32.encodeFromBytes("AGE-SECRET-KEY-", scalar).toUpperCase();
    return Promise.resolve(identity);
  }
  async function identityToRecipient(identity) {
    let scalar;
    if (isCryptoKey2(identity)) {
      scalar = identity;
    } else {
      const res = bech32.decodeToBytes(identity);
      if (!identity.startsWith("AGE-SECRET-KEY-1") || res.prefix.toUpperCase() !== "AGE-SECRET-KEY-" || res.bytes.length !== 32) {
        throw Error("invalid identity");
      }
      scalar = res.bytes;
    }
    const recipient = await scalarMultBase(scalar);
    return bech32.encodeFromBytes("age", recipient);
  }
  var X25519Recipient = class {
    recipient;
    constructor(s) {
      const res = bech32.decodeToBytes(s);
      if (!s.startsWith("age1") || res.prefix.toLowerCase() !== "age" || res.bytes.length !== 32) {
        throw Error("invalid recipient");
      }
      this.recipient = res.bytes;
    }
    async wrapFileKey(fileKey) {
      const ephemeral = randomBytes(32);
      const share = await scalarMultBase(ephemeral);
      const secret = await scalarMult(ephemeral, this.recipient);
      const salt = new Uint8Array(share.length + this.recipient.length);
      salt.set(share);
      salt.set(this.recipient, share.length);
      const key = hkdf(sha256, secret, salt, "age-encryption.org/v1/X25519", 32);
      return [new Stanza(["X25519", base64nopad.encode(share)], encryptFileKey(fileKey, key))];
    }
  };
  var X25519Identity = class {
    identity;
    recipient;
    constructor(s) {
      if (isCryptoKey2(s)) {
        this.identity = s;
        this.recipient = scalarMultBase(s);
        return;
      }
      const res = bech32.decodeToBytes(s);
      if (!s.startsWith("AGE-SECRET-KEY-1") || res.prefix.toUpperCase() !== "AGE-SECRET-KEY-" || res.bytes.length !== 32) {
        throw Error("invalid identity");
      }
      this.identity = res.bytes;
      this.recipient = scalarMultBase(res.bytes);
    }
    async unwrapFileKey(stanzas) {
      for (const s of stanzas) {
        if (s.args.length < 1 || s.args[0] !== "X25519") {
          continue;
        }
        if (s.args.length !== 2) {
          throw Error("invalid X25519 stanza");
        }
        const share = base64nopad.decode(s.args[1]);
        if (share.length !== 32) {
          throw Error("invalid X25519 stanza");
        }
        const secret = await scalarMult(this.identity, share);
        const recipient = await this.recipient;
        const salt = new Uint8Array(share.length + recipient.length);
        salt.set(share);
        salt.set(recipient, share.length);
        const key = hkdf(sha256, secret, salt, "age-encryption.org/v1/X25519", 32);
        const fileKey = decryptFileKey(s.body, key);
        if (fileKey !== null)
          return fileKey;
      }
      return null;
    }
  };
  var ScryptRecipient = class {
    passphrase;
    logN;
    constructor(passphrase, logN) {
      this.passphrase = passphrase;
      this.logN = logN;
    }
    wrapFileKey(fileKey) {
      const salt = randomBytes(16);
      const label2 = "age-encryption.org/v1/scrypt";
      const labelAndSalt = new Uint8Array(label2.length + 16);
      labelAndSalt.set(new TextEncoder().encode(label2));
      labelAndSalt.set(salt, label2.length);
      const key = scrypt(this.passphrase, labelAndSalt, { N: 2 ** this.logN, r: 8, p: 1, dkLen: 32 });
      return [new Stanza(["scrypt", base64nopad.encode(salt), this.logN.toString()], encryptFileKey(fileKey, key))];
    }
  };
  var ScryptIdentity = class {
    passphrase;
    constructor(passphrase) {
      this.passphrase = passphrase;
    }
    unwrapFileKey(stanzas) {
      for (const s of stanzas) {
        if (s.args.length < 1 || s.args[0] !== "scrypt") {
          continue;
        }
        if (stanzas.length !== 1) {
          throw Error("scrypt recipient is not the only one in the header");
        }
        if (s.args.length !== 3) {
          throw Error("invalid scrypt stanza");
        }
        if (!/^[1-9][0-9]*$/.test(s.args[2])) {
          throw Error("invalid scrypt stanza");
        }
        const salt = base64nopad.decode(s.args[1]);
        if (salt.length !== 16) {
          throw Error("invalid scrypt stanza");
        }
        const logN = Number(s.args[2]);
        if (logN > 20) {
          throw Error("scrypt work factor is too high");
        }
        const label2 = "age-encryption.org/v1/scrypt";
        const labelAndSalt = new Uint8Array(label2.length + 16);
        labelAndSalt.set(new TextEncoder().encode(label2));
        labelAndSalt.set(salt, label2.length);
        const key = scrypt(this.passphrase, labelAndSalt, { N: 2 ** logN, r: 8, p: 1, dkLen: 32 });
        const fileKey = decryptFileKey(s.body, key);
        if (fileKey !== null)
          return fileKey;
      }
      return null;
    }
  };
  function encryptFileKey(fileKey, key) {
    const nonce = new Uint8Array(12);
    return chacha20poly1305(key, nonce).encrypt(fileKey);
  }
  function decryptFileKey(body, key) {
    if (body.length !== 32) {
      throw Error("invalid stanza");
    }
    const nonce = new Uint8Array(12);
    try {
      return chacha20poly1305(key, nonce).decrypt(body);
    } catch {
      return null;
    }
  }
  function isCryptoKey2(key) {
    return typeof CryptoKey !== "undefined" && key instanceof CryptoKey;
  }

  // dist/stream.js
  var chacha20poly1305Overhead = 16;
  var chunkSize = /* @__PURE__ */ (() => 64 * 1024)();
  var chunkSizeWithOverhead = /* @__PURE__ */ (() => chunkSize + chacha20poly1305Overhead)();
  function decryptSTREAM(key, ciphertext) {
    const streamNonce = new Uint8Array(12);
    const incNonce = () => {
      for (let i = streamNonce.length - 2; i >= 0; i--) {
        streamNonce[i]++;
        if (streamNonce[i] !== 0)
          break;
      }
    };
    const chunkCount = Math.ceil(ciphertext.length / chunkSizeWithOverhead);
    const overhead = chunkCount * chacha20poly1305Overhead;
    const plaintext = new Uint8Array(ciphertext.length - overhead);
    let plaintextSlice = plaintext;
    while (ciphertext.length > chunkSizeWithOverhead) {
      const chunk2 = chacha20poly1305(key, streamNonce).decrypt(ciphertext.subarray(0, chunkSizeWithOverhead));
      plaintextSlice.set(chunk2);
      plaintextSlice = plaintextSlice.subarray(chunk2.length);
      ciphertext = ciphertext.subarray(chunkSizeWithOverhead);
      incNonce();
    }
    streamNonce[11] = 1;
    const chunk = chacha20poly1305(key, streamNonce).decrypt(ciphertext);
    plaintextSlice.set(chunk);
    if (chunk.length === 0 && plaintext.length !== 0) {
      throw Error("empty final chunk");
    }
    if (plaintextSlice.length !== chunk.length) {
      throw Error("stream: internal error: didn't fill expected plaintext buffer");
    }
    return plaintext;
  }
  function encryptSTREAM(key, plaintext) {
    const streamNonce = new Uint8Array(12);
    const incNonce = () => {
      for (let i = streamNonce.length - 2; i >= 0; i--) {
        streamNonce[i]++;
        if (streamNonce[i] !== 0)
          break;
      }
    };
    const chunkCount = plaintext.length === 0 ? 1 : Math.ceil(plaintext.length / chunkSize);
    const overhead = chunkCount * chacha20poly1305Overhead;
    const ciphertext = new Uint8Array(plaintext.length + overhead);
    let ciphertextSlice = ciphertext;
    while (plaintext.length > chunkSize) {
      const chunk2 = chacha20poly1305(key, streamNonce).encrypt(plaintext.subarray(0, chunkSize));
      ciphertextSlice.set(chunk2);
      ciphertextSlice = ciphertextSlice.subarray(chunk2.length);
      plaintext = plaintext.subarray(chunkSize);
      incNonce();
    }
    streamNonce[11] = 1;
    const chunk = chacha20poly1305(key, streamNonce).encrypt(plaintext);
    ciphertextSlice.set(chunk);
    if (ciphertextSlice.length !== chunk.length) {
      throw Error("stream: internal error: didn't fill expected ciphertext buffer");
    }
    return ciphertext;
  }

  // dist/armor.js
  var armor_exports = {};
  __export(armor_exports, {
    decode: () => decode,
    encode: () => encode
  });
  function encode(file) {
    const lines = [];
    lines.push("-----BEGIN AGE ENCRYPTED FILE-----\n");
    for (let i = 0; i < file.length; i += 48) {
      let end = i + 48;
      if (end > file.length)
        end = file.length;
      lines.push(base64.encode(file.subarray(i, end)) + "\n");
    }
    lines.push("-----END AGE ENCRYPTED FILE-----\n");
    return lines.join("");
  }
  function decode(file) {
    const lines = file.trim().replaceAll("\r\n", "\n").split("\n");
    if (lines.shift() !== "-----BEGIN AGE ENCRYPTED FILE-----") {
      throw Error("invalid header");
    }
    if (lines.pop() !== "-----END AGE ENCRYPTED FILE-----") {
      throw Error("invalid footer");
    }
    function isLineLengthValid(i, l) {
      if (i === lines.length - 1) {
        return l.length > 0 && l.length <= 64 && l.length % 4 === 0;
      }
      return l.length === 64;
    }
    if (!lines.every((l, i) => isLineLengthValid(i, l))) {
      throw Error("invalid line length");
    }
    if (!lines.every((l) => /^[A-Za-z0-9+/=]+$/.test(l))) {
      throw Error("invalid base64");
    }
    return base64.decode(lines.join(""));
  }

  // dist/webauthn.js
  var webauthn_exports = {};
  __export(webauthn_exports, {
    WebAuthnIdentity: () => WebAuthnIdentity,
    WebAuthnRecipient: () => WebAuthnRecipient,
    createCredential: () => createCredential
  });

  // dist/cbor.js
  function readTypeAndArgument(b) {
    if (b.length === 0) {
      throw Error("cbor: unexpected EOF");
    }
    const major = b[0] >> 5;
    const minor = b[0] & 31;
    if (minor <= 23) {
      return [major, minor, b.subarray(1)];
    }
    if (minor === 24) {
      if (b.length < 2) {
        throw Error("cbor: unexpected EOF");
      }
      return [major, b[1], b.subarray(2)];
    }
    if (minor === 25) {
      if (b.length < 3) {
        throw Error("cbor: unexpected EOF");
      }
      return [major, b[1] << 8 | b[2], b.subarray(3)];
    }
    throw Error("cbor: unsupported argument encoding");
  }
  function readUint(b) {
    const [major, minor, rest] = readTypeAndArgument(b);
    if (major !== 0) {
      throw Error("cbor: expected unsigned integer");
    }
    return [minor, rest];
  }
  function readByteString(b) {
    const [major, minor, rest] = readTypeAndArgument(b);
    if (major !== 2) {
      throw Error("cbor: expected byte string");
    }
    if (minor > rest.length) {
      throw Error("cbor: unexpected EOF");
    }
    return [rest.subarray(0, minor), rest.subarray(minor)];
  }
  function readTextString(b) {
    const [major, minor, rest] = readTypeAndArgument(b);
    if (major !== 3) {
      throw Error("cbor: expected text string");
    }
    if (minor > rest.length) {
      throw Error("cbor: unexpected EOF");
    }
    return [new TextDecoder().decode(rest.subarray(0, minor)), rest.subarray(minor)];
  }
  function readArray(b) {
    const [major, minor, r] = readTypeAndArgument(b);
    if (major !== 4) {
      throw Error("cbor: expected array");
    }
    let rest = r;
    const args = [];
    for (let i = 0; i < minor; i++) {
      let arg;
      [arg, rest] = readTextString(rest);
      args.push(arg);
    }
    return [args, rest];
  }
  function encodeUint(n) {
    if (n <= 23) {
      return new Uint8Array([n]);
    }
    if (n <= 255) {
      return new Uint8Array([24, n]);
    }
    if (n <= 65535) {
      return new Uint8Array([25, n >> 8, n & 255]);
    }
    throw Error("cbor: unsigned integer too large");
  }
  function encodeByteString(b) {
    if (b.length <= 23) {
      return new Uint8Array([2 << 5 | b.length, ...b]);
    }
    if (b.length <= 255) {
      return new Uint8Array([2 << 5 | 24, b.length, ...b]);
    }
    if (b.length <= 65535) {
      return new Uint8Array([2 << 5 | 25, b.length >> 8, b.length & 255, ...b]);
    }
    throw Error("cbor: byte string too long");
  }
  function encodeTextString(s) {
    const b = new TextEncoder().encode(s);
    if (b.length <= 23) {
      return new Uint8Array([3 << 5 | b.length, ...b]);
    }
    if (b.length <= 255) {
      return new Uint8Array([3 << 5 | 24, b.length, ...b]);
    }
    if (b.length <= 65535) {
      return new Uint8Array([3 << 5 | 25, b.length >> 8, b.length & 255, ...b]);
    }
    throw Error("cbor: text string too long");
  }
  function encodeArray(args) {
    const body = args.flatMap((x) => [...encodeTextString(x)]);
    if (args.length <= 23) {
      return new Uint8Array([4 << 5 | args.length, ...body]);
    }
    if (args.length <= 255) {
      return new Uint8Array([4 << 5 | 24, args.length, ...body]);
    }
    if (args.length <= 65535) {
      return new Uint8Array([4 << 5 | 25, args.length >> 8, args.length & 255, ...body]);
    }
    throw Error("cbor: array too long");
  }

  // dist/webauthn.js
  var defaultAlgorithms = [
    { type: "public-key", alg: -8 },
    // Ed25519
    { type: "public-key", alg: -7 },
    // ECDSA with P-256 and SHA-256
    { type: "public-key", alg: -257 }
    // RSA PKCS#1 v1.5 with SHA-256
  ];
  async function createCredential(options) {
    const cred = await navigator.credentials.create({
      publicKey: {
        rp: { name: "", id: options.rpId },
        user: {
          name: options.keyName,
          id: randomBytes(8),
          // avoid overwriting existing keys
          displayName: ""
        },
        pubKeyCredParams: defaultAlgorithms,
        authenticatorSelection: {
          requireResidentKey: options.type !== "security-key",
          residentKey: options.type !== "security-key" ? "required" : "discouraged",
          userVerification: "required"
          // prf requires UV
        },
        hints: options.type === "security-key" ? ["security-key"] : [],
        extensions: { prf: {} },
        challenge: new Uint8Array([0]).buffer
        // unused without attestation
      }
    });
    if (!cred.getClientExtensionResults().prf?.enabled) {
      throw Error("PRF extension not available (need macOS 15+, Chrome 132+)");
    }
    const rpId = options.rpId ?? new URL(window.origin).hostname;
    return encodeIdentity(cred, rpId);
  }
  var prefix = "AGE-PLUGIN-FIDO2PRF-";
  function encodeIdentity(credential, rpId) {
    const res = credential.response;
    const version = encodeUint(1);
    const credId = encodeByteString(new Uint8Array(credential.rawId));
    const rp = encodeTextString(rpId);
    const transports = encodeArray(res.getTransports());
    const identityData = new Uint8Array([...version, ...credId, ...rp, ...transports]);
    return bech32.encode(prefix, bech32.toWords(identityData), false).toUpperCase();
  }
  function decodeIdentity(identity) {
    const res = bech32.decodeToBytes(identity);
    if (!identity.startsWith(prefix + "1")) {
      throw Error("invalid identity");
    }
    const [version, rest1] = readUint(res.bytes);
    if (version !== 1) {
      throw Error("unsupported identity version");
    }
    const [credId, rest2] = readByteString(rest1);
    const [rpId, rest3] = readTextString(rest2);
    const [transports] = readArray(rest3);
    return [credId, rpId, transports];
  }
  var label = "age-encryption.org/fido2prf";
  var WebAuthnInternal = class {
    credId;
    transports;
    rpId;
    constructor(options) {
      if (options?.identity) {
        const [credId, rpId, transports] = decodeIdentity(options.identity);
        this.credId = credId;
        this.transports = transports;
        this.rpId = rpId;
      } else {
        this.rpId = options?.rpId;
      }
    }
    async getCredential(nonce) {
      const assertion = await navigator.credentials.get({
        publicKey: {
          allowCredentials: this.credId ? [{
            id: this.credId,
            transports: this.transports,
            type: "public-key"
          }] : [],
          challenge: randomBytes(16),
          extensions: { prf: { eval: prfInputs(nonce) } },
          userVerification: "required",
          // prf requires UV
          rpId: this.rpId
        }
      });
      const results = assertion.getClientExtensionResults().prf?.results;
      if (results === void 0) {
        throw Error("PRF extension not available (need macOS 15+, Chrome 132+)");
      }
      return results;
    }
  };
  var WebAuthnRecipient = class extends WebAuthnInternal {
    /**
     * Implements {@link Recipient.wrapFileKey}.
     */
    async wrapFileKey(fileKey) {
      const nonce = randomBytes(16);
      const results = await this.getCredential(nonce);
      const key = deriveKey(results);
      return [new Stanza([label, base64nopad.encode(nonce)], encryptFileKey(fileKey, key))];
    }
  };
  var WebAuthnIdentity = class extends WebAuthnInternal {
    /**
     * Implements {@link Identity.unwrapFileKey}.
     */
    async unwrapFileKey(stanzas) {
      for (const s of stanzas) {
        if (s.args.length < 1 || s.args[0] !== label) {
          continue;
        }
        if (s.args.length !== 2) {
          throw Error("invalid prf stanza");
        }
        const nonce = base64nopad.decode(s.args[1]);
        if (nonce.length !== 16) {
          throw Error("invalid prf stanza");
        }
        const results = await this.getCredential(nonce);
        const key = deriveKey(results);
        const fileKey = decryptFileKey(s.body, key);
        if (fileKey !== null)
          return fileKey;
      }
      return null;
    }
  };
  function prfInputs(nonce) {
    const prefix2 = new TextEncoder().encode(label);
    const first = new Uint8Array(prefix2.length + nonce.length + 1);
    first.set(prefix2, 0);
    first[prefix2.length] = 1;
    first.set(nonce, prefix2.length + 1);
    const second = new Uint8Array(prefix2.length + nonce.length + 1);
    second.set(prefix2, 0);
    second[prefix2.length] = 2;
    second.set(nonce, prefix2.length + 1);
    return { first, second };
  }
  function deriveKey(results) {
    if (results.second === void 0) {
      throw Error("Missing second PRF result");
    }
    const prf = new Uint8Array(results.first.byteLength + results.second.byteLength);
    prf.set(new Uint8Array(results.first), 0);
    prf.set(new Uint8Array(results.second), results.first.byteLength);
    return extract(sha256, prf, label);
  }

  // dist/index.js
  var Encrypter = class {
    passphrase = null;
    scryptWorkFactor = 18;
    recipients = [];
    /**
     * Set the passphrase to encrypt the file(s) with. This method can only be
     * called once, and can't be called if {@link Encrypter.addRecipient} has
     * been called.
     *
     * The passphrase is passed through the scrypt key derivation function, but
     * it needs to have enough entropy to resist offline brute-force attacks.
     * You should use at least 8-10 random alphanumeric characters, or 4-5
     * random words from a list of at least 2000 words.
     *
     * @param s - The passphrase to encrypt the file with.
     */
    setPassphrase(s) {
      if (this.passphrase !== null) {
        throw new Error("can encrypt to at most one passphrase");
      }
      if (this.recipients.length !== 0) {
        throw new Error("can't encrypt to both recipients and passphrases");
      }
      this.passphrase = s;
    }
    /**
     * Set the scrypt work factor to use when encrypting the file(s) with a
     * passphrase. The default is 18. Using a lower value will require stronger
     * passphrases to resist offline brute-force attacks.
     *
     * @param logN - The base-2 logarithm of the scrypt work factor.
     */
    setScryptWorkFactor(logN) {
      this.scryptWorkFactor = logN;
    }
    /**
     * Add a recipient to encrypt the file(s) for. This method can be called
     * multiple times to encrypt the file(s) for multiple recipients.
     *
     * @param s - The recipient to encrypt the file for. Either a string
     * beginning with `age1...` or an object implementing the {@link Recipient}
     * interface.
     */
    addRecipient(s) {
      if (this.passphrase !== null) {
        throw new Error("can't encrypt to both recipients and passphrases");
      }
      if (typeof s === "string") {
        this.recipients.push(new X25519Recipient(s));
      } else {
        this.recipients.push(s);
      }
    }
    /**
     * Encrypt a file using the configured passphrase or recipients.
     *
     * @param file - The file to encrypt. If a string is passed, it will be
     * encoded as UTF-8.
     *
     * @returns A promise that resolves to the encrypted file as a Uint8Array.
     */
    async encrypt(file) {
      if (typeof file === "string") {
        file = new TextEncoder().encode(file);
      }
      const fileKey = randomBytes(16);
      const stanzas = [];
      let recipients = this.recipients;
      if (this.passphrase !== null) {
        recipients = [new ScryptRecipient(this.passphrase, this.scryptWorkFactor)];
      }
      for (const recipient of recipients) {
        stanzas.push(...await recipient.wrapFileKey(fileKey));
      }
      const hmacKey = hkdf(sha256, fileKey, void 0, "header", 32);
      const mac = hmac(sha256, hmacKey, encodeHeaderNoMAC(stanzas));
      const header = encodeHeader(stanzas, mac);
      const nonce = randomBytes(16);
      const streamKey = hkdf(sha256, fileKey, nonce, "payload", 32);
      const payload = encryptSTREAM(streamKey, file);
      const out = new Uint8Array(header.length + nonce.length + payload.length);
      out.set(header);
      out.set(nonce, header.length);
      out.set(payload, header.length + nonce.length);
      return out;
    }
  };
  var Decrypter = class {
    identities = [];
    /**
     * Add a passphrase to decrypt password-encrypted file(s) with. This method
     * can be called multiple times to try multiple passphrases.
     *
     * @param s - The passphrase to decrypt the file with.
     */
    addPassphrase(s) {
      this.identities.push(new ScryptIdentity(s));
    }
    /**
     * Add an identity to decrypt file(s) with. This method can be called
     * multiple times to try multiple identities.
     *
     * @param s - The identity to decrypt the file with. Either a string
     * beginning with `AGE-SECRET-KEY-1...`, an X25519 private
     * {@link https://developer.mozilla.org/en-US/docs/Web/API/CryptoKey | CryptoKey}
     * object, or an object implementing the {@link Identity} interface.
     *
     * A CryptoKey object must have
     * {@link https://developer.mozilla.org/en-US/docs/Web/API/CryptoKey/type | type}
     * `private`,
     * {@link https://developer.mozilla.org/en-US/docs/Web/API/CryptoKey/algorithm | algorithm}
     * `{name: 'X25519'}`, and
     * {@link https://developer.mozilla.org/en-US/docs/Web/API/CryptoKey/usages | usages}
     * `["deriveBits"]`. For example:
     * ```js
     * const keyPair = await crypto.subtle.generateKey({ name: "X25519" }, false, ["deriveBits"])
     * decrypter.addIdentity(key.privateKey)
     * ```
     */
    addIdentity(s) {
      if (typeof s === "string" || isCryptoKey3(s)) {
        this.identities.push(new X25519Identity(s));
      } else {
        this.identities.push(s);
      }
    }
    async decrypt(file, outputFormat) {
      const h = parseHeader(file);
      const fileKey = await this.unwrapFileKey(h.stanzas);
      if (fileKey === null) {
        throw Error("no identity matched any of the file's recipients");
      }
      const hmacKey = hkdf(sha256, fileKey, void 0, "header", 32);
      const mac = hmac(sha256, hmacKey, h.headerNoMAC);
      if (!compareBytes(h.MAC, mac)) {
        throw Error("invalid header HMAC");
      }
      const nonce = h.rest.subarray(0, 16);
      const streamKey = hkdf(sha256, fileKey, nonce, "payload", 32);
      const payload = h.rest.subarray(16);
      const out = decryptSTREAM(streamKey, payload);
      if (outputFormat === "text")
        return new TextDecoder().decode(out);
      return out;
    }
    /**
     * Decrypt the file key from a detached header. This is a low-level
     * function that can be used to implement delegated decryption logic.
     * Most users won't need this.
     *
     * It is the caller's responsibility to keep track of what file the
     * returned file key decrypts, and to ensure the file key is not used
     * for any other purpose.
     *
     * @param header - The file's textual header, including the MAC.
     *
     * @returns The file key used to encrypt the file.
     */
    async decryptHeader(header) {
      const h = parseHeader(header);
      const fileKey = await this.unwrapFileKey(h.stanzas);
      if (fileKey === null) {
        throw Error("no identity matched any of the file's recipients");
      }
      const hmacKey = hkdf(sha256, fileKey, void 0, "header", 32);
      const mac = hmac(sha256, hmacKey, h.headerNoMAC);
      if (!compareBytes(h.MAC, mac)) {
        throw Error("invalid header HMAC");
      }
      return fileKey;
    }
    async unwrapFileKey(stanzas) {
      for (const identity of this.identities) {
        const fileKey = await identity.unwrapFileKey(stanzas);
        if (fileKey !== null)
          return fileKey;
      }
      return null;
    }
  };
  function compareBytes(a, b) {
    if (a.length !== b.length) {
      return false;
    }
    let acc = 0;
    for (let i = 0; i < a.length; i++) {
      acc |= a[i] ^ b[i];
    }
    return acc === 0;
  }
  function isCryptoKey3(key) {
    return typeof CryptoKey !== "undefined" && key instanceof CryptoKey;
  }
  return __toCommonJS(dist_exports);
})();
/*! Bundled license information:

@noble/hashes/esm/utils.js:
  (*! noble-hashes - MIT License (c) 2022 Paul Miller (paulmillr.com) *)

@scure/base/lib/esm/index.js:
  (*! scure-base - MIT License (c) 2022 Paul Miller (paulmillr.com) *)

@noble/ciphers/esm/utils.js:
  (*! noble-ciphers - MIT License (c) 2023 Paul Miller (paulmillr.com) *)

@noble/curves/esm/abstract/utils.js:
  (*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) *)

@noble/curves/esm/abstract/modular.js:
  (*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) *)

@noble/curves/esm/abstract/montgomery.js:
  (*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) *)

@noble/curves/esm/ed25519.js:
  (*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) *)
*/
