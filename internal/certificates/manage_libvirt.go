/*
SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company and cobaltcore-dev contributors
SPDX-License-Identifier: Apache-2.0

Licensed under the Apache License, LibVirtVersion 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package certificates

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	v1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logger "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cobaltcode-dev/kvm-node-agent/internal/sys"
)

func GetSecretAndCertName(host string) (string, string) {
	certName := fmt.Sprintf("libvirt-%s", host)
	secretName := fmt.Sprintf("tls-%s", certName)
	return secretName, certName
}

var (
	pki = os.Getenv("PKI_PATH")
)

// EnsureCertificate ensures that a certificate exists for the given host and IPs
// TODO: move this code to a controller, so the node-agent doesn't need to have the rights
// to create certificates for any host
func EnsureCertificate(ctx context.Context, c client.Client, host string) error {
	log := logger.FromContext(ctx)

	var ipAddresses []string
	if ips, err := net.LookupIP(sys.Hostname); err != nil {
		if ip, ok := os.LookupEnv("HOST_IP_ADDRESS"); !ok {
			return fmt.Errorf("failed to resolve hostname %s: %w", sys.Hostname, err)
		} else {
			ipAddresses = append(ipAddresses, ip)
		}
	} else {
		for _, ip := range ips {
			if ipv4 := ip.To4(); ipv4 != nil {
				ipAddresses = append(ipAddresses, ipv4.String())
			}
		}
	}

	apiVersion := "cert-manager.io/v1"
	secretName, certName := GetSecretAndCertName(host)

	certificate := cmapi.Certificate{
		TypeMeta: metav1.TypeMeta{
			Kind:       cmapi.CertificateKind,
			APIVersion: apiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      certName,
			Namespace: sys.Namespace,
		},
	}

	update, err := controllerutil.CreateOrUpdate(ctx, c, &certificate, func() error {
		certificate.Spec = cmapi.CertificateSpec{
			SecretName: secretName,
			PrivateKey: &cmapi.CertificatePrivateKey{
				Algorithm: cmapi.RSAKeyAlgorithm,
				Encoding:  cmapi.PKCS1,
				Size:      4096,
			},
			// Values for testing, increase for production to something sensible
			Duration:    &metav1.Duration{Duration: 8 * time.Hour},
			RenewBefore: &metav1.Duration{Duration: 2 * time.Hour},
			IsCA:        false,
			Usages: []cmapi.KeyUsage{
				cmapi.UsageServerAuth,
				cmapi.UsageClientAuth,
				cmapi.UsageCertSign,
				cmapi.UsageDigitalSignature,
				cmapi.UsageKeyEncipherment,
			},
			Subject: &cmapi.X509Subject{
				Organizations: []string{"nova"},
			},
			CommonName:  host,
			DNSNames:    []string{host},
			IPAddresses: ipAddresses,
			IssuerRef: v1.ObjectReference{
				Name:  os.Getenv("ISSUER_NAME"),
				Kind:  cmapi.IssuerKind,
				Group: "cert-manager.io",
			},
		}
		return nil
	})

	if err != nil {
		return err
	}

	if update != controllerutil.OperationResultNone {
		log.Info(fmt.Sprintf("Certificate %s %s", certName, update))
	}

	return nil
}

var secretToFileMap = map[string][]string{
	"ca.crt":  {"CA/cacert.pem", "qemu/ca-cert.pem"},
	"tls.crt": {"libvirt/servercert.pem", "qemu/server-cert.pem"},
	"tls.key": {"libvirt/private/serverkey.pem", "qemu/server-key.pem"},
}

var symLinkMap = map[string][]string{
	"servercert.pem":  {"libvirt/clientcert.pem"},
	"serverkey.pem":   {"libvirt/private/clientkey.pem"},
	"server-cert.pem": {"qemu/client-cert.pem"},
	"server-key.pem":  {"qemu/client-key.pem"},
}

func UpdateTLSCertificate(ctx context.Context, data map[string][]byte) error {
	log := logger.FromContext(ctx)
	log.Info("updating TLS certificates for libvirt", "path", pki)

	// write files
	for source, targets := range secretToFileMap {
		for _, target := range targets {
			// prepend the pki path for the target
			target = filepath.Join(pki, target)

			if _, ok := data[source]; !ok {
				return fmt.Errorf("missing data for secret key %s", source)
			}

			// ensure the target directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(target), err)
			}

			// write the file
			if err := os.WriteFile(target, data[source], 0640); err != nil {
				return fmt.Errorf("failed to write targetFile %s: %w", target, err)
			}
		}
	}

	// handle symlinks
	for source, targets := range symLinkMap {
		for _, target := range targets {
			// prepend the pki path for both, source and target
			target = filepath.Join(pki, target)

			// ensure the target directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(target), err)
			}

			// check if the target exists and is correct, else create symlink
			fileInfo, err := os.Lstat(target)
			if err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("failed to stat target %s: %w", target, err)
				}
			} else {
				// check if the target is a symlink, and correct it if necessary
				if fileInfo.Mode()&os.ModeSymlink != 0 {
					// if the target is a symlink, check if it points to the correct source
					link, err := os.Readlink(target)
					if err != nil {
						return fmt.Errorf("failed to read symlink %s: %w", target, err)
					}

					// if the link is correctly pointing to the source, continue
					if filepath.Clean(link) == filepath.Clean(source) {
						continue
					}

					// link is not pointing to the source, remove it
					if err := os.Remove(target); err != nil {
						return fmt.Errorf("failed to remove symlink %s: %w", target, err)
					}
				}
			}

			// create symlink
			if err := os.Symlink(source, target); err != nil {
				return fmt.Errorf("failed to create symlink %s -> %s: %w", target, source, err)
			}
		}
	}
	return nil
}
