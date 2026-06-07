// Command amend-issuer produces an unsigned RFC 9925 version of an X.509
// issuer certificate, using for its Subject field the exact byte-for-byte
// encoding of a child certificate's Issuer field.
//
// This works around encoding mismatches in the issuer↔subject comparison that
// are not supported by the Go verifier, without access to any private key.
package main

import (
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
)

// idAlgUnsigned is the id-alg-unsigned OBJECT IDENTIFIER from RFC 9925. It is
// used as the signatureAlgorithm and TBSCertificate.signature of an unsigned
// certificate, which carries subject information without an issuer signature.
var idAlgUnsigned = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 6, 36}

// amendIssuer returns an unsigned RFC 9925 version of the issuer certificate
// whose Subject field is the exact encoding of the child certificate's Issuer
// field. The result keeps the issuer's public key, validity, and extensions, so
// that children carrying the differently-encoded Issuer verify against it.
func amendIssuer(issuerPEM, childPEM []byte) ([]byte, error) {
	issuerDER, err := decodeCertificate(issuerPEM)
	if err != nil {
		return nil, fmt.Errorf("issuer: %w", err)
	}
	childDER, err := decodeCertificate(childPEM)
	if err != nil {
		return nil, fmt.Errorf("child: %w", err)
	}

	issuer, err := x509.ParseCertificate(issuerDER)
	if err != nil {
		return nil, fmt.Errorf("parsing issuer: %w", err)
	}
	child, err := x509.ParseCertificate(childDER)
	if err != nil {
		return nil, fmt.Errorf("parsing child: %w", err)
	}

	// The amendment only helps if the issuer actually signed the child: the
	// amended certificate keeps the issuer's public key, which is what verifies
	// the child's signature. Checking it here also catches swapped inputs.
	// CheckSignatureFrom does not compare names, so it succeeds despite the
	// encoding mismatch we are working around.
	if err := child.CheckSignatureFrom(issuer); err != nil {
		return nil, fmt.Errorf("issuer did not sign child: %w", err)
	}

	amended, err := unsignedCertificate(issuerDER, child.RawIssuer)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: amended}), nil
}

// decodeCertificate returns the DER bytes of the first CERTIFICATE PEM block.
func decodeCertificate(pemBytes []byte) ([]byte, error) {
	for rest := pemBytes; ; {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			return nil, errors.New("no CERTIFICATE PEM block found")
		}
		if block.Type == "CERTIFICATE" {
			return block.Bytes, nil
		}
	}
}

// unsignedCertificate rebuilds certDER as an unsigned RFC 9925 certificate,
// replacing the Subject and Issuer names with name and stripping the signature.
// Every other field (serial number, validity, public key, extensions) is kept
// byte-for-byte from the original.
func unsignedCertificate(certDER, name []byte) ([]byte, error) {
	// Certificate ::= SEQUENCE { tbsCertificate, signatureAlgorithm, signatureValue }
	var cert asn1.RawValue
	if _, err := asn1.Unmarshal(certDER, &cert); err != nil {
		return nil, fmt.Errorf("malformed certificate: %w", err)
	}
	if cert.Class != asn1.ClassUniversal || cert.Tag != asn1.TagSequence || !cert.IsCompound {
		return nil, errors.New("malformed certificate: not a SEQUENCE")
	}

	// Read the tbsCertificate, discarding the trailing signature fields.
	var tbs asn1.RawValue
	if _, err := asn1.Unmarshal(cert.Bytes, &tbs); err != nil {
		return nil, fmt.Errorf("malformed tbsCertificate: %w", err)
	}

	// TBSCertificate ::= SEQUENCE {
	//     version      [0] EXPLICIT Version DEFAULT v1,
	//     serialNumber     CertificateSerialNumber,
	//     signature        AlgorithmIdentifier,
	//     issuer           Name,
	//     validity         Validity,
	//     subject          Name,
	//     subjectPublicKeyInfo SubjectPublicKeyInfo,
	//     ... [1] [2] [3] optional fields }
	rest := tbs.Bytes

	// version is [0] EXPLICIT (tag 0xA0) and optional; keep it verbatim if present.
	var version []byte
	var err error
	if len(rest) > 0 && rest[0] == 0xA0 {
		if version, rest, err = next(rest); err != nil {
			return nil, err
		}
	}
	serial, rest, err := next(rest)
	if err != nil {
		return nil, err
	}
	if _, rest, err = next(rest); err != nil { // signature, discarded
		return nil, err
	}
	if _, rest, err = next(rest); err != nil { // issuer, replaced
		return nil, err
	}
	validity, rest, err := next(rest)
	if err != nil {
		return nil, err
	}
	if _, rest, err = next(rest); err != nil { // subject, replaced
		return nil, err
	}
	spki, extra, err := next(rest) // subjectPublicKeyInfo; extra holds optional fields
	if err != nil {
		return nil, err
	}

	// AlgorithmIdentifier ::= SEQUENCE { algorithm OBJECT IDENTIFIER }, with the
	// parameters omitted as required by RFC 9925 for id-alg-unsigned.
	unsignedAlg, err := asn1.Marshal(struct{ Algorithm asn1.ObjectIdentifier }{idAlgUnsigned})
	if err != nil {
		return nil, err
	}
	// signatureValue MUST be a BIT STRING of length zero (encoded 03 01 00).
	emptySignature, err := asn1.Marshal(asn1.BitString{})
	if err != nil {
		return nil, err
	}

	var tbsBody []byte
	tbsBody = append(tbsBody, version...)
	tbsBody = append(tbsBody, serial...)
	tbsBody = append(tbsBody, unsignedAlg...)
	tbsBody = append(tbsBody, name...) // issuer, copied from subject per RFC 9925
	tbsBody = append(tbsBody, validity...)
	tbsBody = append(tbsBody, name...) // subject
	tbsBody = append(tbsBody, spki...)
	tbsBody = append(tbsBody, extra...)
	tbsDER, err := sequence(tbsBody)
	if err != nil {
		return nil, err
	}

	var certBody []byte
	certBody = append(certBody, tbsDER...)
	certBody = append(certBody, unsignedAlg...)
	certBody = append(certBody, emptySignature...)
	return sequence(certBody)
}

// next reads one DER element from in, returning its full encoding and the rest.
func next(in []byte) (element, rest []byte, err error) {
	var v asn1.RawValue
	rest, err = asn1.Unmarshal(in, &v)
	if err != nil {
		return nil, nil, fmt.Errorf("malformed tbsCertificate: %w", err)
	}
	return v.FullBytes, rest, nil
}

// sequence wraps body in a DER SEQUENCE.
func sequence(body []byte) ([]byte, error) {
	return asn1.Marshal(asn1.RawValue{
		Class: asn1.ClassUniversal, Tag: asn1.TagSequence, IsCompound: true, Bytes: body,
	})
}
