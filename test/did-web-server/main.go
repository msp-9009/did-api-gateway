package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// DIDDocument represents a minimal DID Document for testing
type DIDDocument struct {
	Context            interface{}          `json:"@context,omitempty"`
	ID                 string               `json:"id"`
	VerificationMethod []VerificationMethod `json:"verificationMethod"`
	Authentication     []interface{}        `json:"authentication"`
}

type VerificationMethod struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Controller   string                 `json:"controller"`
	PublicKeyJwk map[string]interface{} `json:"publicKeyJwk,omitempty"`
}

var (
	port    = flag.Int("port", 8888, "HTTP server port")
	domain  = flag.String("domain", "localhost:8888", "Domain name for DID (e.g., localhost:8888)")
	pubKeyX = flag.String("pubkey", "", "Ed25519 public key in base64url format (32 bytes)")
)

func main() {
	flag.Parse()

	// Create sample DID document if pubkey not provided
	samplePubKey := "dGVzdF9wdWJsaWNfa2V5XzMyX2J5dGVzX2hlcmVfMTIz" // Sample base64url
	if *pubKeyX != "" {
		samplePubKey = *pubKeyX
	}

	did := fmt.Sprintf("did:web:%s", *domain)

	didDoc := DIDDocument{
		Context: []interface{}{
			"https://www.w3.org/ns/did/v1",
			"https://w3id.org/security/suites/ed25519-2020/v1",
		},
		ID: did,
		VerificationMethod: []VerificationMethod{
			{
				ID:         did + "#key-1",
				Type:       "Ed25519VerificationKey2020",
				Controller: did,
				PublicKeyJwk: map[string]interface{}{
					"kty": "OKP",
					"crv": "Ed25519",
					"x":   samplePubKey,
				},
			},
		},
		Authentication: []interface{}{
			did + "#key-1",
		},
	}

	// Set up HTTP server
	mux := http.NewServeMux()

	// Serve DID document at /.well-known/did.json
	mux.HandleFunc("/.well-known/did.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if err := json.NewEncoder(w).Encode(didDoc); err != nil {
			http.Error(w, "Failed to encode DID document", http.StatusInternalServerError)
			return
		}
		log.Printf("Served DID document for %s", did)
	})

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Root handler - show instructions
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>DID:Web Test Server</title>
    <style>
        body { font-family: sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
        pre { background: #f4f4f4; padding: 15px; border-radius: 5px; overflow-x: auto; }
        .success { color: green; }
        .info { color: blue; }
    </style>
</head>
<body>
    <h1>🌐 DID:Web Test Server</h1>
    <p class="success">✅ Server is running!</p>
    
    <h2>DID Information</h2>
    <p><strong>DID:</strong> <code>%s</code></p>
    <p><strong>DID Document URL:</strong> <a href="/.well-known/did.json">%s/.well-known/did.json</a></p>
    
    <h2>Test with Gateway</h2>
    <p>Use this DID to test the gateway's did:web resolver:</p>
    <pre>curl 'http://localhost:8080/v1/auth/challenge?did=%s'</pre>
    
    <h2>View DID Document</h2>
    <p>Click here to view the DID document: <a href="/.well-known/did.json">/.well-known/did.json</a></p>
    
    <h2>Custom Public Key</h2>
    <p>To use your own Ed25519 public key, restart the server with:</p>
    <pre>./did-web-test-server -pubkey YOUR_BASE64URL_PUBKEY -domain localhost:8888</pre>
</body>
</html>
`, did, *domain, did)
	})

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("🚀 DID:Web Test Server starting on %s", addr)
	log.Printf("📝 DID: %s", did)
	log.Printf("🔗 DID Document: http://%s/.well-known/did.json", *domain)
	log.Printf("💡 Open http://localhost:%d in your browser for instructions", *port)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
