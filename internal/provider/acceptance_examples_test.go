package provider

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// examplesFunctionsDir is the on-disk home of the per-function examples, relative to this package.
const examplesFunctionsDir = "../../examples/functions"

// TestAcc_Examples_Runnable applies every examples/functions/<name>/function.tf under the real provider and asserts it plans and applies without error.
//
// These files are not decorative: tfplugindocs embeds each one verbatim into its function's registry doc (via the `tffile` template helper), so the example a user copy-pastes is exactly this file. Nothing else executes them, `terraform fmt` only checks formatting, and the hand-written acceptance tests use their own inline configs, so without this a renamed argument, a wrong arity, or a typo'd function call ships as broken copy-paste and is only caught by a user. This runs the whole examples tree through Terraform on every CI run.
//
// A handful of examples read sidecar data via file(...) (a payload to sign, an .ini to decode). We copy the example into a temp dir and materialise those fixtures beside it rather than committing them: the allowlist .gitignore would need widening for each esoteric extension, and pkcs7_sign would mean checking a private key into the repo. The fixtures are the same shapes the format's own acceptance tests use, so they are known-good input.
func TestAcc_Examples_Runnable(t *testing.T) {
	entries, err := os.ReadDir(examplesFunctionsDir)
	if err != nil {
		t.Fatalf("read examples dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			src := filepath.Join(examplesFunctionsDir, name, "function.tf")
			cfg, err := os.ReadFile(src)
			if err != nil {
				t.Fatalf("read %s: %v", src, err)
			}

			// Assemble a self-contained copy of the example so file(...) references and path.module resolve. StaticDirectory copies this whole directory (fixtures included) into Terraform's working dir.
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "function.tf"), cfg, 0o600); err != nil {
				t.Fatalf("write config copy: %v", err)
			}
			// The examples are bare output/locals with no terraform{} block. The inline-Config test path auto-injects a required_providers block, but ConfigDirectory treats the directory as authoritative and does not, so `provider::burnham::` would be an unknown-provider error. Supply the same block the framework would, matching its reattach source address.
			if err := os.WriteFile(filepath.Join(dir, "providers.tf"), []byte(providersConfig()), 0o600); err != nil {
				t.Fatalf("write providers config: %v", err)
			}
			writeExampleFixtures(t, name, dir)

			resource.UnitTest(t, resource.TestCase{
				TerraformVersionChecks:   testAccTerraformVersionChecks,
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					ConfigDirectory: config.StaticDirectory(dir),
				}},
			})
		})
	}
}

// providersConfig returns a terraform{} block declaring the burnham provider at the source address terraform-plugin-testing uses for its in-process reattach, so Terraform resolves `provider::burnham::` to the test provider instead of trying to install one. Mirrors the framework's getProviderAddr, including its host/namespace env overrides.
func providersConfig() string {
	host := "registry.terraform.io"
	if v := os.Getenv("TF_ACC_PROVIDER_HOST"); v != "" {
		host = v
	}
	namespace := "hashicorp"
	if v := os.Getenv("TF_ACC_PROVIDER_NAMESPACE"); v != "" {
		namespace = v
	}
	return `terraform {
  required_providers {
    burnham = {
      source = "` + host + `/` + namespace + `/burnham"
    }
  }
}
`
}

// examplePayload is the byte string the crypto examples sign or derive keys from. Any valid UTF-8 works; file() rejects non-UTF-8 content.
const examplePayload = "burnham deterministic signing payload fixture\n"

// examplePlist is a minimal valid Configuration profile for the plistdecode example, matching the shape used in the plist acceptance tests.
const examplePlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
	<key>PayloadEnabled</key>
	<true/>
</dict>
</plist>
`

// writeExampleFixtures materialises the sidecar files a given example reads via file(...). Examples with no external data need nothing, so most names fall through.
func writeExampleFixtures(t *testing.T, name, dir string) {
	t.Helper()
	switch name {
	case "ecdsa_p256_key_from_seed", "ed25519_key_from_seed", "x509_self_sign":
		writeFixture(t, dir, "payload.bin", []byte(examplePayload))
	case "pkcs7_sign":
		writeFixture(t, dir, "payload.bin", []byte(examplePayload))
		key, cert := genSignerPEM(t)
		writeFixture(t, dir, "signer.key.pem", key)
		writeFixture(t, dir, "signer.cert.pem", cert)
	case "inidecode":
		writeFixture(t, dir, "app.ini", []byte("[database]\nhost = db.example.com\nport = 5432\n"))
	case "kdldecode":
		writeFixture(t, dir, "config.kdl", []byte("title \"Hello\"\n"))
	case "hujsondecode":
		writeFixture(t, dir, "policy.hujson", []byte("{// comment\n\"key\": \"value\",}\n"))
	case "vdfdecode":
		writeFixture(t, dir, "appmanifest_730.acf", []byte("\"Config\"\n{\n\t\"key\"\t\t\"value\"\n}\n"))
	case "regdecode":
		// .reg files are CRLF-delimited and lead with the version banner.
		writeFixture(t, dir, "policy.reg", []byte("Windows Registry Editor Version 5.00\r\n\r\n[HKEY_LOCAL_MACHINE\\SOFTWARE\\Example]\r\n\"Setting\"=\"value\"\r\n"))
	case "plistdecode":
		writeFixture(t, dir, "profile.mobileconfig", []byte(examplePlist))
	}
}

func writeFixture(t *testing.T, dir, filename string, content []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filename), content, 0o600); err != nil {
		t.Fatalf("write fixture %s: %v", filename, err)
	}
}

// genSignerPEM returns a matching ECDSA P-256 PKCS#8 private key and self-signed certificate in PEM form, for the pkcs7_sign example's caller-supplied-identity path. pkcs7_sign requires the cert's public key to match the private key; a fresh self-signed pair satisfies that.
func genSignerPEM(t *testing.T) (keyPEM, certPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate signer key: %v", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal signer key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "signer.example"},
		NotBefore:    time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:     time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create signer cert: %v", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	return keyPEM, certPEM
}
