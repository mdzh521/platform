package k8s

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func (c *clusterClient) getJSON(path string, target any) error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("kubernetes api returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func (c *clusterClient) getNamespaceResourceCount(namespace, resource string) (int, error) {
	var payload struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/%s", namespace, resource), &payload); err != nil {
		return 0, err
	}
	return len(payload.Items), nil
}

func (c *clusterClient) patchJSON(path, contentType string, body []byte, target any) error {
	return c.doJSON(http.MethodPatch, path, contentType, body, target)
}

func (c *clusterClient) putJSON(path string, body any, target any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.doJSON(http.MethodPut, path, "application/json", data, target)
}

func (c *clusterClient) doJSON(method, path, contentType string, body []byte, target any) error {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("kubernetes api returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	if target == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func (c *clusterClient) delete(path string) error {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("kubernetes api returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

func (c *clusterClient) getText(path string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return "", err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "text/plain, application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("kubernetes api returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return string(body), nil
}

func parseTokenCredential(cluster Cluster) (tokenCredential, error) {
	raw := extractStoredCredential(cluster.CredentialJSON)
	if raw == "" {
		return tokenCredential{}, errors.New("cluster credential is required for token auth")
	}

	var credential tokenCredential
	if strings.HasPrefix(strings.TrimSpace(raw), "{") {
		if err := json.Unmarshal([]byte(raw), &credential); err != nil {
			return tokenCredential{}, fmt.Errorf("invalid token credential json: %w", err)
		}
	}
	if credential.Token == "" {
		credential.Token = raw
	}
	if credential.Token == "" {
		return tokenCredential{}, errors.New("token credential is empty")
	}
	return credential, nil
}

func buildFromKubeconfig(cluster Cluster) (*clusterClient, error) {
	raw := extractStoredCredential(cluster.CredentialJSON)
	if raw == "" {
		return nil, errors.New("cluster kubeconfig is empty")
	}

	var cfg kubeConfig
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, fmt.Errorf("invalid kubeconfig: %w", err)
	}
	if cfg.CurrentContext == "" && len(cfg.Contexts) > 0 {
		cfg.CurrentContext = cfg.Contexts[0].Name
	}

	var clusterRef, userRef string
	for _, item := range cfg.Contexts {
		if item.Name == cfg.CurrentContext {
			clusterRef = item.Context.Cluster
			userRef = item.Context.User
			break
		}
	}
	if clusterRef == "" {
		return nil, errors.New("kubeconfig context not found")
	}

	var server, caData string
	var insecureSkipVerify bool
	for _, item := range cfg.Clusters {
		if item.Name == clusterRef {
			server = item.Cluster.Server
			caData = item.Cluster.CertificateAuthorityData
			insecureSkipVerify = item.Cluster.InsecureSkipTLSVerify
			break
		}
	}
	if server == "" {
		return nil, errors.New("kubeconfig cluster server not found")
	}

	var token string
	var clientCertificate string
	var clientCertificateData string
	var clientKey string
	var clientKeyData string
	var execCommand string
	var execArgs []string
	var execEnv []execEnvItem
	var authProvider string
	for _, item := range cfg.Users {
		if item.Name == userRef {
			token = item.User.Token
			clientCertificate = item.User.ClientCertificate
			clientCertificateData = item.User.ClientCertificateData
			clientKey = item.User.ClientKey
			clientKeyData = item.User.ClientKeyData
			execCommand = item.User.Exec.Command
			execArgs = item.User.Exec.Args
			execEnv = item.User.Exec.Env
			authProvider = item.User.AuthProvider.Name
			break
		}
	}
	if token == "" {
		if execCommand != "" {
			execToken, err := resolveExecToken(execCommand, execArgs, execEnv)
			if err != nil {
				return nil, err
			}
			token = execToken
		}
		if authProvider != "" {
			return nil, fmt.Errorf("kubeconfig uses auth-provider (%s); current version only supports static token kubeconfig or direct token mode", authProvider)
		}
		if token == "" && strings.TrimSpace(clientCertificate) == "" && strings.TrimSpace(clientCertificateData) == "" {
			return nil, errors.New("kubeconfig token user not found; current version only supports static token kubeconfig, direct token mode, aws exec kubeconfig, or client certificate kubeconfig")
		}
	}

	clientCertPEM, err := loadPEMCredential(clientCertificateData, clientCertificate, "CERTIFICATE")
	if err != nil {
		return nil, err
	}
	clientKeyPEM, err := loadPEMCredential(clientKeyData, clientKey, "PRIVATE KEY")
	if err != nil {
		return nil, err
	}

	httpClient, err := buildHTTPClient(caData, insecureSkipVerify, clientCertPEM, clientKeyPEM)
	if err != nil {
		return nil, err
	}
	return &clusterClient{
		baseURL:    strings.TrimRight(server, "/"),
		httpClient: httpClient,
		token:      token,
	}, nil
}

func buildHTTPClient(caCert string, insecureSkipVerify bool, clientCertPEM, clientKeyPEM []byte) (*http.Client, error) {
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: insecureSkipVerify}
	if strings.TrimSpace(caCert) != "" {
		pemData := []byte(caCert)
		if !strings.Contains(caCert, "BEGIN CERTIFICATE") {
			if decoded, err := base64.StdEncoding.DecodeString(caCert); err == nil {
				pemData = decoded
			}
		}
		certPool := x509.NewCertPool()
		if ok := certPool.AppendCertsFromPEM(pemData); !ok {
			return nil, errors.New("invalid cluster ca certificate")
		}
		tlsConfig.RootCAs = certPool
	}
	if len(clientCertPEM) > 0 || len(clientKeyPEM) > 0 {
		if len(clientCertPEM) == 0 || len(clientKeyPEM) == 0 {
			return nil, errors.New("client certificate auth requires both certificate and key")
		}
		clientCert, err := tls.X509KeyPair(clientCertPEM, clientKeyPEM)
		if err != nil {
			return nil, fmt.Errorf("invalid client certificate auth material: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}, nil
}

func loadPEMCredential(data, path, marker string) ([]byte, error) {
	data = strings.TrimSpace(data)
	if data != "" {
		decoded, err := decodePEMValue(data)
		if err != nil {
			return nil, fmt.Errorf("invalid kubeconfig %s data: %w", strings.ToLower(marker), err)
		}
		if !strings.Contains(string(decoded), marker) {
			return nil, fmt.Errorf("invalid kubeconfig %s data", strings.ToLower(marker))
		}
		return decoded, nil
	}

	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	pemData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read kubeconfig %s file: %w", strings.ToLower(marker), err)
	}
	if !strings.Contains(string(pemData), marker) {
		return nil, fmt.Errorf("invalid kubeconfig %s file", strings.ToLower(marker))
	}
	return pemData, nil
}

func decodePEMValue(value string) ([]byte, error) {
	if strings.Contains(value, "BEGIN ") {
		return []byte(value), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func extractStoredCredential(stored string) string {
	if strings.TrimSpace(stored) == "" {
		return ""
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(stored), &payload); err == nil && payload["credential"] != "" {
		return payload["credential"]
	}
	return stored
}
