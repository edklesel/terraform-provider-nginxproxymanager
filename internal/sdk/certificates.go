// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package go_npm_proxy_manager

import (
	"encoding/json"
	"fmt"
	"net/http"

	version "github.com/hashicorp/go-version"
)

type LetsencryptCertificateDates struct {
	From *int `json:"from,omitempty"`
	To   *int `json:"to,omitempty"`
}

type LetsencryptCertificate struct {
	CN     *string                      `json:"cn,omitempty"`
	Issuer *string                      `json:"issuer,omitempty"`
	Dates  *LetsencryptCertificateDates `json:"dates,omitempty"`
}

type CertificateMeta struct {
	Certificate            *string                 `json:"certificate,omitempty"`
	CertificateKey         *string                 `json:"certificate_key,omitempty"`
	DNSChallenge           *bool                   `json:"dns_challenge,omitempty"`
	DNSProvider            *string                 `json:"dns_provider,omitempty"`
	DNSProviderCredentials *string                 `json:"dns_provider_credentials,omitempty"`
	LetsencryptAgree       *bool                   `json:"letsencrypt_agree,omitempty" proto.constraint:"< 2.13"`
	LetsencryptCertificate *LetsencryptCertificate `json:"letsencrypt_certificate,omitempty"`
	LetsencryptEmail       *string                 `json:"letsencrypt_email,omitempty"  proto.constraint:"< 2.13"`
	PropagationSeconds     *int                    `json:"propagation_seconds,omitempty"`
}

type CertificateBase struct {
	Provider    *string          `json:"provider" proto:"other"`
	NiceName    *string          `json:"nice_name,omitempty"`
	DomainNames *[]string        `json:"domain_names,omitempty"`
	Meta        *CertificateMeta `json:"meta,omitempty"`
}

type Certificate struct {
	CertificateBase
	ObjectMeta
}

type CertificateUpload struct {
	Certificate             *string
	CertificateKey          *string
	IntermediateCertificate *string
}

// Certificates are not normal CRUD objects, so we create separate methods for those.
type certObject struct {
	client          *Client
	apiPath         string
	writeOnlyFields []string
}

// The List, Get and Delete functions are the same for Certificates.
func (o certObject) List() ([]Certificate, error) {
	return _list[Certificate](o.client, o.apiPath)
}
func (o certObject) Get(id int) (Certificate, error) {
	return _get[Certificate](o.client, o.apiPath, id)
}
func (o certObject) Delete(id int) error {
	return _delete[Certificate](o.client, o.apiPath, id)
}

// But the Create function is different
// Here, create just created the Certificate object in NPM, it doesn't
//
//	validate or upload the certificate/key.
func (o certObject) Create(newCert CertificateBase) (cert Certificate, err error) {

	// Minor input validation
	if *newCert.Provider != "other" && *newCert.Provider != "letsencrypt" {
		return cert, fmt.Errorf("provider field must either be 'letsencrypt' or 'other', got %s", *newCert.Provider)
	}

	body, err := json.Marshal(newCert)
	if err != nil {
		return cert, fmt.Errorf("unable to marshal Certificate into json: %s", err)
	}

	response, err := o.client.NewRequest(
		o.apiPath,
		http.MethodPost,
		&body,
		true,
		returnPointer("application/json"),
	)
	if err != nil {
		return cert, fmt.Errorf("error making request to create Certificate: %s", err)
	}

	err = json.Unmarshal(response, &cert)
	if err != nil {
		return cert, fmt.Errorf("unable to unmarshal json response: %s", err)
	}

	setZeroSlices(&cert)

	return
}

// Validate a custom Certificate.
func (o certObject) Validate(newCert CertificateUpload) (err error) {
	_, err = o.newCertificateRequest(newCert, "validate")
	return
}

// Upload a certificate with key to NPM..
func (o certObject) Upload(id int, newCert CertificateUpload) (cert CertificateUpload, err error) {
	_cert, err := o.newCertificateRequest(newCert, fmt.Sprintf("%d/upload", id))
	if err != nil {
		return
	}
	err = json.Unmarshal(_cert, &cert)
	if err != nil {
		return cert, fmt.Errorf("unable to unmarshal response: %s", err)
	}
	return
}

// This function the base certificate multipart request.
func (o certObject) newCertificateRequest(newCert CertificateUpload, path string) (res []byte, err error) {
	values := map[string]string{
		"certificate":     *newCert.Certificate,
		"certificate_key": *newCert.CertificateKey,
	}
	if newCert.IntermediateCertificate != nil {
		values["intermediate_certificate"] = *newCert.IntermediateCertificate
	}

	_body, content_type, err := newMultipartBody(values)
	if err != nil {
		return
	}
	body := _body.Bytes()

	return o.client.NewRequest(
		fmt.Sprintf("%s/%s", o.apiPath, path),
		http.MethodPost,
		&body,
		true,
		&content_type,
	)
}

func (o certObject) ClientVersion() *version.Version {
	return o.client.Version
}

func (o certObject) GetWriteOnlyFields() []string {
	return o.writeOnlyFields
}

type DNSProvider struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Credentials string `json:"credentials"`
}

// DNSProviders lists the DNS providers the running NPM supports for DNS
// challenges, with an example of the credentials each one expects.
func (o certObject) DNSProviders() (providers []DNSProvider, err error) {
	response, err := o.client.NewRequest(
		fmt.Sprintf("%s/dns-providers", o.apiPath),
		http.MethodGet,
		nil,
		true,
		nil,
	)
	if err != nil {
		return providers, fmt.Errorf("unable to list DNS providers: %s", err)
	}
	if err = json.Unmarshal(response, &providers); err != nil {
		return providers, fmt.Errorf("unable to unmarshal json response: %s", err)
	}
	return providers, nil
}
