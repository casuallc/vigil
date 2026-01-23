/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cli

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// setupTLSCommands 设置TLS相关命令
func (c *CLI) setupTLSCommands() *cobra.Command {
	// TLS command - 作为父命令来组织所有TLS相关的子命令
	tlsCmd := &cobra.Command{
		Use:   "tls",
		Short: "TLS certificate operations",
		Long:  "Manage TLS certificates for HTTPS",
	}

	// 添加子命令
	tlsCmd.AddCommand(c.setupTLSGenerateCommand())

	return tlsCmd
}

// setupTLSGenerateCommand 设置tls generate命令
func (c *CLI) setupTLSGenerateCommand() *cobra.Command {
	var (  
		certPath string
		keyPath  string
		host     string
	)

	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate TLS certificate",
		Long:  "Generate a self-signed TLS certificate for HTTPS",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleTLSGenerate(certPath, keyPath, host)
		},
	}

	generateCmd.Flags().StringVarP(&certPath, "cert", "c", "cert.pem", "Certificate file path")
	generateCmd.Flags().StringVarP(&keyPath, "key", "k", "key.pem", "Private key file path")
	generateCmd.Flags().StringVarP(&host, "host", "H", "localhost", "Hostname or IP address for the certificate")

	return generateCmd
}

// handleTLSGenerate 处理tls generate命令
func (c *CLI) handleTLSGenerate(certPath, keyPath, host string) error {
	// 生成私钥
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %v", err)
	}

	// 创建证书模板
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Vigil"},
			CommonName:   host,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour), // 1 year
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		BasicConstraintsValid: true,
	}

	// 添加主机名或IP地址
	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
	} else {
		template.DNSNames = append(template.DNSNames, host)
	}

	// 生成证书
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		return fmt.Errorf("failed to generate certificate: %v", err)
	}

	// 保存证书
	certOut, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("failed to create certificate file: %v", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	}); err != nil {
		return fmt.Errorf("failed to encode certificate: %v", err)
	}

	// 保存私钥
	keyOut, err := os.Create(keyPath)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %v", err)
	}
	defer keyOut.Close()

	if err := pem.Encode(keyOut, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	}); err != nil {
		return fmt.Errorf("failed to encode private key: %v", err)
	}

	fmt.Printf("TLS certificate generated successfully:\n")
	fmt.Printf("Certificate: %s\n", certPath)
	fmt.Printf("Private key: %s\n", keyPath)

	return nil
}
