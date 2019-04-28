package g53

import ()

type Rcode uint8

const (
	R_NOERROR    Rcode = 0  ///< 0: No error (RFC1035)
	R_FORMERR    Rcode = 1  ///< 1: Format error (RFC1035)
	R_SERVFAIL   Rcode = 2  ///< 2: Server failure (RFC1035)
	R_NXDOMAIN   Rcode = 3  ///< 3: Name Error (RFC1035)
	R_NOTIMP     Rcode = 4  ///< 4: Not Implemented (RFC1035)
	R_REFUSED    Rcode = 5  ///< 5: Refused (RFC1035)
	R_YXDOMAIN   Rcode = 6  ///< 6: Name unexpectedly exists (RFC2136)
	R_YXRRSET    Rcode = 7  ///< 7: RRset unexpectedly exists (RFC2136)
	R_NXRRSET    Rcode = 8  ///< 8: RRset should exist but not (RFC2136)
	R_NOTAUTH    Rcode = 9  ///< 9: Server isn't authoritative (RFC2136)
	R_NOTZONE    Rcode = 10 ///< 10: Name is not within the zone (RFC2136)
	R_RESERVED11 Rcode = 11 ///< 11: Reserved for future use (RFC1035)
	R_RESERVED12 Rcode = 12 ///< 12: Reserved for future use (RFC1035)
	R_RESERVED13 Rcode = 13 ///< 13: Reserved for future use (RFC1035)
	R_RESERVED14 Rcode = 14 ///< 14: Reserved for future use (RFC1035)
	R_RESERVED15 Rcode = 15 ///< 15: Reserved for future use (RFC1035)
	R_BADSIG     Rcode = 16 ///< 16: TSIG verify failed for TSIG Error(RFC2845)
	R_BADKEY     Rcode = 17 ///< 17: TSIG no such key for TSIG Error(RFC2845)
	R_BADTIME    Rcode = 18 ///< 18: TSIG time expired for TSIG Error(RFC2845)
)

var RcodeStr = map[Rcode]string{
	R_NOERROR:    "NOERROR",
	R_FORMERR:    "FORMERR",
	R_SERVFAIL:   "SERVFAIL",
	R_NXDOMAIN:   "NXDOMAIN",
	R_NOTIMP:     "NOTIMP",
	R_REFUSED:    "REFUSED",
	R_YXDOMAIN:   "YXDOMAIN",
	R_YXRRSET:    "YXRRSET",
	R_NXRRSET:    "NXRRSET",
	R_NOTAUTH:    "NOTAUTH",
	R_NOTZONE:    "NOTZONE",
	R_RESERVED11: "RESERVED11",
	R_RESERVED12: "RESERVED12",
	R_RESERVED13: "RESERVED13",
	R_RESERVED14: "RESERVED14",
	R_RESERVED15: "RESERVED15",
	R_BADSIG:     "BADSIG",
	R_BADKEY:     "BADKEY",
	R_BADTIME:    "BADTIME",
}

func (c Rcode) String() string {
	return RcodeStr[c]
}
