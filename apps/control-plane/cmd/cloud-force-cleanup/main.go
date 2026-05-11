package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

type terraformImport struct {
	Address string
	ID      string
}

type cleanupPlan struct {
	AccountID         uint
	Region            string
	InstanceIDs       []string
	AllocationIDs     []string
	SecurityGroupID   string
	InstanceProfile   string
	RoleName          string
	ManagedPolicyARNs []string
	Imports           []terraformImport
}

func main() {
	jobID := flag.Int("job-id", 0, "source apply job id")
	flag.Parse()

	if *jobID == 0 {
		fail("job-id 不能为空")
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

	plan, err := loadCleanupPlan(db, *jobID)
	if err != nil {
		fail(err.Error())
	}

	account, err := loadAccount(db, plan.AccountID)
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
	if strings.TrimSpace(plan.Region) == "" {
		plan.Region = strings.TrimSpace(account.Region)
	}
	if strings.TrimSpace(plan.Region) == "" {
		fail("未找到可用 region")
	}

	env := map[string]string{
		"AWS_ACCESS_KEY_ID":         accessKey,
		"AWS_SECRET_ACCESS_KEY":     secretKey,
		"AWS_REGION":                plan.Region,
		"AWS_DEFAULT_REGION":        plan.Region,
		"AWS_EC2_METADATA_DISABLED": "true",
	}

	fmt.Printf("force cleanup job=%d region=%s instances=%s eips=%s sg=%s profile=%s role=%s\n",
		*jobID,
		plan.Region,
		strings.Join(plan.InstanceIDs, ","),
		strings.Join(plan.AllocationIDs, ","),
		plan.SecurityGroupID,
		plan.InstanceProfile,
		plan.RoleName,
	)

	if err := runTerraformCleanup(env, *jobID, plan); err != nil {
		fail(err.Error())
	}

	fmt.Println("force cleanup completed")
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

func loadCleanupPlan(db *sql.DB, jobID int) (cleanupPlan, error) {
	row := db.QueryRow(`select account_id, provider, input_json from deployment_jobs where id = ?`, jobID)
	var (
		accountID uint
		provider  string
		inputRaw  sql.NullString
	)
	if err := row.Scan(&accountID, &provider, &inputRaw); err != nil {
		return cleanupPlan{}, err
	}
	if strings.TrimSpace(provider) != "aws" {
		return cleanupPlan{}, fmt.Errorf("job %d provider=%s，当前脚本只支持 aws", jobID, provider)
	}

	plan := cleanupPlan{
		AccountID:         accountID,
		ManagedPolicyARNs: []string{"arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"},
	}
	if inputRaw.Valid && strings.TrimSpace(inputRaw.String) != "" {
		var inputPayload map[string]any
		if err := json.Unmarshal([]byte(inputRaw.String), &inputPayload); err == nil {
			plan.Region = readNestedString(inputPayload, "input", "region")
		}
	}

	rows, err := db.Query(`
		select resource_type, cloud_id, metadata_json
		from deployment_resources
		where job_id = ?
		order by id
	`, jobID)
	if err != nil {
		return cleanupPlan{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			resourceType string
			cloudID      sql.NullString
			metaRaw      sql.NullString
		)
		if err := rows.Scan(&resourceType, &cloudID, &metaRaw); err != nil {
			return cleanupPlan{}, err
		}
		id := strings.TrimSpace(cloudID.String)

		meta := map[string]any{}
		if metaRaw.Valid && strings.TrimSpace(metaRaw.String) != "" {
			_ = json.Unmarshal([]byte(metaRaw.String), &meta)
		}
		nameTag := readNestedString(meta, "metadata", "tags", "Name")

		switch strings.TrimSpace(resourceType) {
		case "aws_instance":
			if id != "" {
				plan.InstanceIDs = appendUnique(plan.InstanceIDs, id)
				if nameTag != "" {
					plan.Imports = append(plan.Imports, terraformImport{
						Address: fmt.Sprintf("aws_instance.server[%q]", nameTag),
						ID:      id,
					})
				}
			}
		case "aws_eip":
			if id != "" {
				plan.AllocationIDs = appendUnique(plan.AllocationIDs, id)
				if nameTag != "" {
					plan.Imports = append(plan.Imports, terraformImport{
						Address: fmt.Sprintf("aws_eip.server[%q]", strings.TrimSuffix(nameTag, "-eip")),
						ID:      id,
					})
				}
			}
		case "aws_security_group":
			if plan.SecurityGroupID == "" && id != "" {
				plan.SecurityGroupID = id
				plan.Imports = append(plan.Imports, terraformImport{
					Address: "aws_security_group.server",
					ID:      id,
				})
			}
		case "aws_iam_instance_profile":
			if plan.InstanceProfile == "" {
				plan.InstanceProfile = firstNonEmpty(id, readNestedString(meta, "metadata", "name"))
				plan.Imports = append(plan.Imports, terraformImport{
					Address: "aws_iam_instance_profile.server",
					ID:      plan.InstanceProfile,
				})
			}
		case "aws_iam_role":
			if plan.RoleName == "" {
				plan.RoleName = firstNonEmpty(id, readNestedString(meta, "metadata", "name"))
				plan.Imports = append(plan.Imports, terraformImport{
					Address: "aws_iam_role.server",
					ID:      plan.RoleName,
				})
			}
		}
	}
	if err := rows.Err(); err != nil {
		return cleanupPlan{}, err
	}

	if plan.RoleName != "" {
		plan.Imports = append(plan.Imports, terraformImport{
			Address: "aws_iam_role_policy_attachment.ssm",
			ID:      fmt.Sprintf("%s/%s", plan.RoleName, "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"),
		})
	}

	if plan.AccountID == 0 {
		return cleanupPlan{}, errors.New("未找到对应任务的账号信息")
	}
	return plan, nil
}

func loadAccount(db *sql.DB, accountID uint) (accountRecord, error) {
	var rec accountRecord
	row := db.QueryRow(`select id, region, access_key_encrypted, secret_key_encrypted from cloud_accounts where id = ?`, accountID)
	if err := row.Scan(&rec.ID, &rec.Region, &rec.AccessKeyEncrypted, &rec.SecretKeyEncrypted); err != nil {
		return accountRecord{}, err
	}
	return rec, nil
}

func runTerraformCleanup(env map[string]string, jobID int, plan cleanupPlan) error {
	workspace := fmt.Sprintf("/runner-data/jobs/job-%d", jobID)
	if err := terraform(env, workspace, "init", "-input=false", "-no-color"); err != nil {
		return err
	}
	for _, item := range uniqueImports(plan.Imports) {
		if err := runTerraformImport(env, workspace, item); err != nil {
			return err
		}
	}
	return terraform(env, workspace, "destroy", "-auto-approve", "-no-color")
}

func runTerraformImport(env map[string]string, workspace string, item terraformImport) error {
	err := terraform(env, workspace, "import", "-no-color", item.Address, item.ID)
	if err != nil && strings.Contains(err.Error(), "Resource already managed by Terraform") {
		return nil
	}
	return err
}

func terraform(env map[string]string, workspace string, args ...string) error {
	commandArgs := []string{
		"compose", "-f", "/Users/alex/ops/platform-center/ops/compose/docker-compose.yml",
		"exec", "-T",
	}
	for _, key := range []string{
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_REGION",
		"AWS_DEFAULT_REGION",
		"AWS_EC2_METADATA_DISABLED",
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
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		} else {
			message = fmt.Sprintf("%s (%v)", message, err)
		}
		return fmt.Errorf("terraform %s failed: %s", strings.Join(args, " "), message)
	}
	return nil
}

func runWithTimeout(cmd *exec.Cmd, timeout time.Duration) ([]byte, error) {
	done := make(chan struct{})
	var (
		output []byte
		err    error
	)
	go func() {
		output, err = cmd.CombinedOutput()
		close(done)
	}()
	select {
	case <-done:
		return output, err
	case <-time.After(timeout):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return output, fmt.Errorf("command timed out after %s", timeout)
	}
}

func readNestedString(data map[string]any, path ...string) string {
	current := any(data)
	for _, part := range path {
		asMap, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = asMap[part]
	}
	switch value := current.(type) {
	case string:
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func uniqueImports(items []terraformImport) []terraformImport {
	seen := map[string]struct{}{}
	result := make([]terraformImport, 0, len(items))
	for _, item := range items {
		key := item.Address + "=>" + item.ID
		if strings.TrimSpace(item.Address) == "" || strings.TrimSpace(item.ID) == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}
	return result
}

func appendUnique(items []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return items
	}
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
