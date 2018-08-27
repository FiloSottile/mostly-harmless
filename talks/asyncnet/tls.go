// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import "golang.org/x/crypto/cryptobyte"

type ClientHello struct {
	SNI string
}

func ParseClientHello(record []byte) (c *ClientHello, ok bool) {
	c = &ClientHello{}

	/* struct {
		ContentType type;
		ProtocolVersion legacy_record_version;
		uint16 length;
		opaque fragment[TLSPlaintext.length];
	} TLSPlaintext; */

	in := cryptobyte.String(record)
	if !in.Skip(1) || !in.Skip(2) {
		return nil, false
	}
	var msg cryptobyte.String
	if !in.ReadUint16LengthPrefixed(&msg) || !in.Empty() {
		return nil, false
	}

	/* struct {
		HandshakeType msg_type;
		uint24 length;
		select (Handshake.msg_type) {
			case client_hello: ClientHello;
		}
	} Handshake; */

	var msgType uint8
	if !msg.ReadUint8(&msgType) {
		return nil, false
	}
	var ch cryptobyte.String
	if !msg.ReadUint24LengthPrefixed(&ch) || !msg.Empty() {
		return nil, false
	}

	/* struct {
		ProtocolVersion legacy_version = 0x0303;
		Random random;
		opaque legacy_session_id<0..32>;
		CipherSuite cipher_suites<2..2^16-2>;
		opaque legacy_compression_methods<1..2^8-1>;
		Extension extensions<8..2^16-1>;
	} ClientHello; */

	if !ch.Skip(2) || !ch.Skip(32) {
		return nil, false
	}
	var skip cryptobyte.String
	if !ch.ReadUint8LengthPrefixed(&skip) ||
		!ch.ReadUint16LengthPrefixed(&skip) ||
		!ch.ReadUint8LengthPrefixed(&skip) {
		return nil, false
	}
	var exts cryptobyte.String
	if !ch.ReadUint16LengthPrefixed(&exts) || !ch.Empty() {
		return nil, false
	}

	/* struct {
	    ExtensionType extension_type;
	    opaque extension_data<0..2^16-1>;
	} Extension; */

	for !exts.Empty() {
		var extensionType uint16
		if !exts.ReadUint16(&extensionType) {
			return nil, false
		}
		var ex cryptobyte.String
		if !exts.ReadUint16LengthPrefixed(&ex) {
			return nil, false
		}

		if extensionType != 0 /* server_name */ {
			continue
		}

		/* struct {
			ServerName server_name_list<1..2^16-1>
		} ServerNameList; */

		var snl cryptobyte.String
		if !ex.ReadUint16LengthPrefixed(&snl) || !ex.Empty() {
			return nil, false
		}

		for !snl.Empty() {
			/* struct {
				NameType name_type;
				opaque HostName<1..2^16-1>;
			} ServerName; */

			var nameType uint8
			if !snl.ReadUint8(&nameType) {
				return nil, false
			}
			var hostName cryptobyte.String
			if !snl.ReadUint16LengthPrefixed(&hostName) {
				return nil, false
			}

			if nameType != 0 /* host_name */ {
				return nil, false
			}
			c.SNI = string(hostName)
		}
	}

	return c, true
}
