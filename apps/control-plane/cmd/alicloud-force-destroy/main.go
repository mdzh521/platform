package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"backend-center/internal/infra/secure"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v3"
)

type config struct {
	JWTSecret     string `yaml:"jwt_secret"`
	MySQLHost     string `yaml:"mysql_host"`
	MySQLPort     string `yaml:"mysql_port"`
	MySQLDatabase string `yaml:"mysql_database"`
	MySQLUser     string `yaml:"mysql_user"`
	MySQLPassword string `yaml:"mysql_password"`
}

type accountRecord struct {
	ID                 uint
	Region             string
	AccessKeyEncrypted string
	SecretKeyEncrypted string
}

func main() {
	jobID := flag.Int("job-id", 0, "source apply job id")
	securityGroupID := flag.String("delete-security-group", "", "delete a single security group in the same account/region context")
	keyPairName := flag.String("delete-key-pair", "", "delete a single key pair in the same account/region context")
	flag.Parse()

	if *jobID == 0 && strings.TrimSpace(*securityGroupID) == "" && strings.TrimSpace(*keyPairName) == "" {
		fail("job-id、delete-security-group、delete-key-pair 至少传一个")
	}

	cfg, err := loadConfig("/Users/alex/ops/platform-center/apps/control-plane/config/platform-center.yaml")
	if err != nil {
		fail(err.Error())
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", cfg.MySQLUser, cfg.MySQLPassword, cfg.MySQLHost, cfg.MySQLPort, cfg.MySQLDatabase)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fail(err.Error())
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fail(err.Error())
	}

	var (
		accountID uint
		region    string
		workspace string
	)
	if *jobID > 0 {
		accountID, region, workspace, err = loadJob(db, *jobID)
		if err != nil {
			fail(err.Error())
		}
	} else {
		accountID, region, err = loadSecurityGroupContext(db, strings.TrimSpace(*securityGroupID))
		if err != nil {
			fail(err.Error())
		}
	}

	account, err := loadAccount(db, accountID)
	if err != nil {
		fail(err.Error())
	}

	cipher := secure.New(cfg.JWTSecret)
	accessKey, err := cipher.Decrypt(account.AccessKeyEncrypted)
	if err != nil {
		fail(err.Error())
	}
	secretKey, err := cipher.Decrypt(account.SecretKeyEncrypted)
	if err != nil {
		fail(err.Error())
	}
	if strings.TrimSpace(region) == "" {
		region = strings.TrimSpace(account.Region)
	}
	if strings.TrimSpace(region) == "" {
		fail("未找到可用 region")
	}
	if *jobID > 0 && strings.TrimSpace(workspace) == "" {
		workspace = fmt.Sprintf("/runner-data/jobs/job-%d", *jobID)
	}

	env := map[string]string{
		"ALICLOUD_ACCESS_KEY": accessKey,
		"ALICLOUD_SECRET_KEY": secretKey,
		"ALICLOUD_REGION":     region,
	}

	if strings.TrimSpace(*securityGroupID) != "" {
		fmt.Printf("alicloud delete security group=%s region=%s\n", strings.TrimSpace(*securityGroupID), region)
		if err := deleteAliCloudSecurityGroup(accessKey, secretKey, region, strings.TrimSpace(*securityGroupID)); err != nil {
			fail(err.Error())
		}
		fmt.Println("alicloud security group delete completed")
		return
	}
	if strings.TrimSpace(*keyPairName) != "" {
		fmt.Printf("alicloud delete key pair=%s region=%s\n", strings.TrimSpace(*keyPairName), region)
		if err := deleteAliCloudKeyPair(accessKey, secretKey, region, strings.TrimSpace(*keyPairName)); err != nil {
			fail(err.Error())
		}
		fmt.Println("alicloud key pair delete completed")
		return
	}

	fmt.Printf("alicloud force destroy job=%d region=%s workspace=%s\n", *jobID, region, workspace)
	if err := terraform(env, workspace, "init", "-input=false", "-no-color"); err != nil {
		fail(err.Error())
	}
	if err := terraform(env, workspace, "destroy", "-auto-approve", "-no-color"); err != nil {
		fail(err.Error())
	}

	fmt.Println("alicloud force destroy completed")
}

func loadConfig(path string) (config, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return config{}, err
	}
	var cfg config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return config{}, err
	}
	if strings.TrimSpace(cfg.MySQLHost) == "" || strings.TrimSpace(cfg.MySQLHost) == "mysql" {
		cfg.MySQLHost = "127.0.0.1"
	}
	if strings.TrimSpace(cfg.MySQLPort) == "" {
		cfg.MySQLPort = "3306"
	}
	return cfg, nil
}

func loadJob(db *sql.DB, jobID int) (uint, string, string, error) {
	row := db.QueryRow(`select account_id, provider, input_json, workspace_path, network_plan_id from deployment_jobs where id = ?`, jobID)
	var (
		accountID uint
		provider  string
		inputRaw  sql.NullString
		workspace sql.NullString
		networkID sql.NullInt64
	)
	if err := row.Scan(&accountID, &provider, &inputRaw, &workspace, &networkID); err != nil {
		return 0, "", "", err
	}
	if strings.TrimSpace(provider) != "alicloud" {
		return 0, "", "", fmt.Errorf("job %d provider=%s，当前脚本只支持 alicloud", jobID, provider)
	}

	region := ""
	if inputRaw.Valid && strings.TrimSpace(inputRaw.String) != "" {
		var payload map[string]any
		if err := json.Unmarshal([]byte(inputRaw.String), &payload); err == nil {
			if nested, ok := payload["input"].(map[string]any); ok {
				if value, ok := nested["region"]; ok {
					region = normalizeStringValue(value)
				}
			}
		}
	}
	if region == "" && networkID.Valid && networkID.Int64 > 0 {
		row := db.QueryRow(`select region from network_plans where id = ?`, networkID.Int64)
		var networkRegion sql.NullString
		if err := row.Scan(&networkRegion); err == nil {
			region = strings.TrimSpace(networkRegion.String)
		}
	}
	return accountID, region, strings.TrimSpace(workspace.String), nil
}

func normalizeStringValue(value any) string {
	if value == nil {
		return ""
	}
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" || text == "<nil>" {
		return ""
	}
	return text
}

func loadAccount(db *sql.DB, accountID uint) (accountRecord, error) {
	var rec accountRecord
	row := db.QueryRow(`select id, region, access_key_encrypted, secret_key_encrypted from cloud_accounts where id = ?`, accountID)
	if err := row.Scan(&rec.ID, &rec.Region, &rec.AccessKeyEncrypted, &rec.SecretKeyEncrypted); err != nil {
		return accountRecord{}, err
	}
	return rec, nil
}

func loadSecurityGroupContext(db *sql.DB, securityGroupID string) (uint, string, error) {
	rows, err := db.Query(`select account_id, metadata_json from cloud_resources where resource_type = 'alicloud_security_group' and cloud_id = ? order by id desc`, securityGroupID)
	if err != nil {
		return 0, "", err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			accountID uint
			metaRaw   sql.NullString
		)
		if err := rows.Scan(&accountID, &metaRaw); err != nil {
			return 0, "", err
		}
		region := ""
		if metaRaw.Valid && strings.TrimSpace(metaRaw.String) != "" {
			var meta map[string]any
			if err := json.Unmarshal([]byte(metaRaw.String), &meta); err == nil {
				if metadata, ok := meta["metadata"].(map[string]any); ok {
					region = normalizeStringValue(metadata["region_id"])
				}
			}
		}
		return accountID, region, nil
	}
	return 0, "", fmt.Errorf("未找到安全组 %s 的账号上下文", securityGroupID)
}

func terraform(env map[string]string, workspace string, args ...string) error {
	commandArgs := []string{
		"compose", "-f", "/Users/alex/ops/platform-center/ops/compose/docker-compose.yml",
		"exec", "-T",
	}
	for _, key := range []string{
		"ALICLOUD_ACCESS_KEY",
		"ALICLOUD_SECRET_KEY",
		"ALICLOUD_REGION",
	} {
		if value := strings.TrimSpace(env[key]); value != "" {
			commandArgs = append(commandArgs, "-e", key+"="+value)
		}
	}
	commandArgs = append(commandArgs,
		"terraform-runner",
		"terraform",
		fmt.Sprintf("-chdir=%s", workspace),
	)
	commandArgs = append(commandArgs, args...)

	cmd := exec.Command("docker", commandArgs...)
	cmd.Env = os.Environ()
	output, err := runWithTimeout(cmd, 20*time.Minute)
	if err != nil {
		return fmt.Errorf("terraform %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(output)))
	}
	if len(output) > 0 {
		fmt.Println(strings.TrimSpace(string(output)))
	}
	return nil
}

func deleteAliCloudSecurityGroup(accessKey, secretKey, region, securityGroupID string) error {
	resp, err := callAliCloudRPC("https://ecs.aliyuncs.com/", accessKey, secretKey, map[string]string{
		"Action":           "DeleteSecurityGroup",
		"RegionId":         region,
		"SecurityGroupId":  securityGroupID,
		"Version":          "2014-05-26",
		"Format":           "JSON",
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureVersion": "1.0",
	})
	if err != nil {
		return err
	}
	var payload struct {
		Code      string `json:"Code"`
		Message   string `json:"Message"`
		RequestID string `json:"RequestId"`
	}
	if err := json.Unmarshal(resp, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.Code) != "" {
		return fmt.Errorf("删除阿里云安全组失败: %s", firstNonEmpty(strings.TrimSpace(payload.Message), strings.TrimSpace(payload.Code)))
	}
	return nil
}

func deleteAliCloudKeyPair(accessKey, secretKey, region, keyPairName string) error {
	resp, err := callAliCloudRPC("https://ecs.aliyuncs.com/", accessKey, secretKey, map[string]string{
		"Action":           "DeleteKeyPairs",
		"RegionId":         region,
		"KeyPairNames":     fmt.Sprintf("[\"%s\"]", keyPairName),
		"Version":          "2014-05-26",
		"Format":           "JSON",
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureVersion": "1.0",
	})
	if err != nil {
		return err
	}
	var payload struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	if err := json.Unmarshal(resp, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.Code) != "" {
		return fmt.Errorf("删除阿里云密钥失败: %s", firstNonEmpty(strings.TrimSpace(payload.Message), strings.TrimSpace(payload.Code)))
	}
	return nil
}

func callAliCloudRPC(endpoint, accessKey, secretKey string, params map[string]string) ([]byte, error) {
	values := url.Values{}
	for key, value := range params {
		if strings.TrimSpace(value) == "" {
			continue
		}
		values.Set(key, value)
	}
	values.Set("AccessKeyId", accessKey)
	values.Set("SignatureNonce", strconv.FormatInt(time.Now().UnixNano(), 10))
	values.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05Z"))

	canonicalized := values.Encode()
	stringToSign := "GET&%2F&" + aliCloudPercentEncode(canonicalized)
	mac := hmac.New(sha1.New, []byte(secretKey+"&"))
	mac.Write([]byte(stringToSign))
	values.Set("Signature", base64.StdEncoding.EncodeToString(mac.Sum(nil)))

	req, err := http.NewRequest(http.MethodGet, endpoint+"?"+values.Encode(), nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("阿里云接口返回 %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func aliCloudPercentEncode(value string) string {
	encoded := url.QueryEscape(value)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func runWithTimeout(cmd *exec.Cmd, timeout time.Duration) ([]byte, error) {
	if timeout <= 0 {
		return cmd.CombinedOutput()
	}
	timer := time.AfterFunc(timeout, func() {
		_ = cmd.Process.Kill()
	})
	defer timer.Stop()
	return cmd.CombinedOutput()
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
