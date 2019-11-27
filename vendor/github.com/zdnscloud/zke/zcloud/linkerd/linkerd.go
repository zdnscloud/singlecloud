package linkerd

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	"github.com/zdnscloud/cement/uuid"

	"github.com/zdnscloud/zke/zcloud/linkerd/pkg/tls"
)

const (
	caLifeTime = 10 * 365
	issuerName = "identity.linkerd."
)

func GetDeployConfig(clusterDomain string) (map[string]interface{}, error) {
	spValidatorCA, err := generateCertificateAuthority("linkerd-sp-validator.linkerd.svc", caLifeTime)
	if err != nil {
		return nil, fmt.Errorf("gen linkerd-sp-validator certificate failed: %s", err.Error())
	}

	proxyInjectCA, err := generateCertificateAuthority("linkerd-proxy-injector.linkerd.svc", caLifeTime)
	if err != nil {
		return nil, fmt.Errorf("gen linkerd-proxy-injector certificate failed: %s", err.Error())
	}

	tapCA, err := generateCertificateAuthority("linkerd-tap.linkerd.svc", caLifeTime)
	if err != nil {
		return nil, fmt.Errorf("gen linkerd-tap certificate failed: %s", err.Error())
	}

	installUUID, err := uuid.Gen()
	if err != nil {
		return nil, fmt.Errorf("gen linkerd config install uuid failed: %s", err.Error())
	}

	root, err := tls.GenerateRootCAWithDefaults(issuerName + clusterDomain)
	if err != nil {
		return nil, fmt.Errorf("gen root certificate for identity failed: %s", err.Error())
	}

	return map[string]interface{}{
		"ClusterDomain":                 clusterDomain,
		"LinkerdProxyInjectorTLSCrtPEM": b64enc(proxyInjectCA.Cert),
		"LinkerdProxyInjectorTLSKeyPEM": b64enc(proxyInjectCA.Key),
		"LinkerdSpValidatorTLSCrtPEM":   b64enc(spValidatorCA.Cert),
		"LinkerdSpValidatorTLSKeyPEM":   b64enc(spValidatorCA.Key),
		"LinkerdTapTLSCrtPEM":           b64enc(tapCA.Cert),
		"LinkerdTapTLSKeyPEM":           b64enc(tapCA.Key),
		"LinkerdIdentityIsserCrtPEM":    b64enc(strings.TrimSpace(root.Cred.Crt.EncodeCertificatePEM())),
		"LinkerdIdentityIsserKeyPEM":    b64enc(strings.TrimSpace(root.Cred.EncodePrivateKeyPEM())),
		"TrustAnchorsPEM":               strings.TrimSpace(root.Cred.Crt.EncodeCertificatePEM()),
		"LinkerdConfigInstallUUID":      installUUID,
		"LinkerdIdentityIssuerExpiry":   root.Cred.Crt.Certificate.NotAfter.Format(time.RFC3339),
	}, nil
}

func b64enc(v string) string {
	return base64.StdEncoding.EncodeToString([]byte(v))
}

type certificate struct {
	Cert string
	Key  string
}

func generateCertificateAuthority(cn string, daysValid int) (certificate, error) {
	ca := certificate{}

	template, err := getBaseCertTemplate(cn, nil, nil, daysValid)
	if err != nil {
		return ca, err
	}

	template.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign
	template.IsCA = true

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return ca, fmt.Errorf("error generating rsa key: %s", err)
	}

	ca.Cert, ca.Key, err = getCertAndKey(template, priv, template, priv)
	if err != nil {
		return ca, err
	}

	return ca, nil
}

func getBaseCertTemplate(cn string, ips []interface{}, alternateDNS []interface{}, daysValid int) (*x509.Certificate, error) {
	ipAddresses, err := getNetIPs(ips)
	if err != nil {
		return nil, err
	}
	dnsNames, err := getAlternateDNSStrs(alternateDNS)
	if err != nil {
		return nil, err
	}
	serialNumberUpperBound := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberUpperBound)
	if err != nil {
		return nil, err
	}
	return &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{CommonName: cn},
		IPAddresses:           ipAddresses,
		DNSNames:              dnsNames,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * time.Duration(daysValid)),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}, nil
}

func getNetIPs(ips []interface{}) ([]net.IP, error) {
	if ips == nil {
		return []net.IP{}, nil
	}
	var ipStr string
	var ok bool
	var netIP net.IP
	netIPs := make([]net.IP, len(ips))
	for i, ip := range ips {
		ipStr, ok = ip.(string)
		if !ok {
			return nil, fmt.Errorf("error parsing ip: %v is not a string", ip)
		}
		netIP = net.ParseIP(ipStr)
		if netIP == nil {
			return nil, fmt.Errorf("error parsing ip: %s", ipStr)
		}
		netIPs[i] = netIP
	}
	return netIPs, nil
}

func getAlternateDNSStrs(alternateDNS []interface{}) ([]string, error) {
	if alternateDNS == nil {
		return []string{}, nil
	}
	var dnsStr string
	var ok bool
	alternateDNSStrs := make([]string, len(alternateDNS))
	for i, dns := range alternateDNS {
		dnsStr, ok = dns.(string)
		if !ok {
			return nil, fmt.Errorf(
				"error processing alternate dns name: %v is not a string",
				dns,
			)
		}
		alternateDNSStrs[i] = dnsStr
	}
	return alternateDNSStrs, nil
}

func getCertAndKey(template *x509.Certificate, signeeKey *rsa.PrivateKey, parent *x509.Certificate, signingKey *rsa.PrivateKey) (string, string, error) {
	derBytes, err := x509.CreateCertificate(rand.Reader, template, parent, &signeeKey.PublicKey, signingKey)
	if err != nil {
		return "", "", fmt.Errorf("error creating certificate: %s", err)
	}

	certBuffer := bytes.Buffer{}
	if err := pem.Encode(&certBuffer, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return "", "", fmt.Errorf("error pem-encoding certificate: %s", err)
	}

	keyBuffer := bytes.Buffer{}
	if err := pem.Encode(
		&keyBuffer,
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(signeeKey),
		},
	); err != nil {
		return "", "", fmt.Errorf("error pem-encoding key: %s", err)
	}

	return string(certBuffer.Bytes()), string(keyBuffer.Bytes()), nil
}
