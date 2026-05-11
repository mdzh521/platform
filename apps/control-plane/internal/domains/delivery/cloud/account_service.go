package cloud

import (
	"backend-center/internal/domains/shared/scopeutil"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

type AccountService struct {
	base *baseService
}

func (s *AccountService) List() ([]CloudAccountView, error) {
	var items []CloudAccount
	if err := s.base.db.Order("created_at desc").Find(&items).Error; err != nil {
		return nil, err
	}
	result := make([]CloudAccountView, 0, len(items))
	for _, item := range items {
		result = append(result, accountToView(item))
	}
	return result, nil
}

func (s *AccountService) Create(input CloudAccountInput) (CloudAccountView, error) {
	provider := normalizeProvider(input.Provider)
	if provider == "" {
		return CloudAccountView{}, errors.New("请选择云平台")
	}
	if strings.TrimSpace(input.Name) == "" {
		return CloudAccountView{}, errors.New("云账号名称不能为空")
	}
	if strings.TrimSpace(input.Region) == "" {
		return CloudAccountView{}, errors.New("默认区域不能为空")
	}
	if strings.TrimSpace(input.AccessKey) == "" {
		return CloudAccountView{}, errors.New("AccessKey 不能为空")
	}
	if strings.TrimSpace(input.SecretKey) == "" {
		return CloudAccountView{}, errors.New("SecretKey 不能为空")
	}
	if err := scopeutil.ValidateAccountDefaults(s.base.db, input.DefaultProjectID, input.DefaultEnvironmentID); err != nil {
		return CloudAccountView{}, err
	}

	accessEncrypted, err := s.base.cipher.Encrypt(strings.TrimSpace(input.AccessKey))
	if err != nil {
		return CloudAccountView{}, err
	}
	secretEncrypted, err := s.base.cipher.Encrypt(strings.TrimSpace(input.SecretKey))
	if err != nil {
		return CloudAccountView{}, err
	}

	model := CloudAccount{
		Name:                 strings.TrimSpace(input.Name),
		Provider:             provider,
		AccessKeyEncrypted:   accessEncrypted,
		SecretKeyEncrypted:   secretEncrypted,
		Region:               strings.TrimSpace(input.Region),
		RoleARN:              strings.TrimSpace(input.RoleARN),
		DefaultTagsJSON:      mustJSONString(cleanStringList(input.DefaultTags)),
		DefaultZonesJSON:     mustJSONString(cleanStringList(input.DefaultZones)),
		DefaultProjectID:     input.DefaultProjectID,
		DefaultEnvironmentID: input.DefaultEnvironmentID,
		Status:               "configured",
	}
	if err := s.base.db.Create(&model).Error; err != nil {
		return CloudAccountView{}, err
	}
	return accountToView(model), nil
}

func (s *AccountService) Test(id uint) (CloudAccountView, error) {
	var item CloudAccount
	if err := s.base.db.First(&item, id).Error; err != nil {
		return CloudAccountView{}, err
	}
	if strings.TrimSpace(item.Region) == "" || strings.TrimSpace(item.AccessKeyEncrypted) == "" || strings.TrimSpace(item.SecretKeyEncrypted) == "" {
		return CloudAccountView{}, errors.New("云账号配置不完整")
	}

	now := time.Now()
	item.Status = "validated"
	item.LastCheckedAt = &now
	if err := s.base.db.Save(&item).Error; err != nil {
		return CloudAccountView{}, err
	}
	return accountToView(item), nil
}

func (s *AccountService) Delete(id uint) error {
	var item CloudAccount
	if err := s.base.db.First(&item, id).Error; err != nil {
		return err
	}

	var networkCount int64
	if err := s.base.db.Model(&NetworkPlan{}).Where("account_id = ?", id).Count(&networkCount).Error; err != nil {
		return err
	}
	if networkCount > 0 {
		return fmt.Errorf("请先删除该账号下的 Foundation Network（共 %d 条）", networkCount)
	}

	var jobCount int64
	if err := s.base.db.Model(&DeploymentJob{}).Where("account_id = ?", id).Count(&jobCount).Error; err != nil {
		return err
	}
	if jobCount > 0 {
		return fmt.Errorf("请先清理该账号下的 Delivery Job（共 %d 条）", jobCount)
	}

	var resourceCount int64
	if err := s.base.db.Model(&CloudResource{}).Where("account_id = ?", id).Count(&resourceCount).Error; err != nil {
		return err
	}
	if resourceCount > 0 {
		return fmt.Errorf("请先清理该账号下的资源台账（共 %d 条）", resourceCount)
	}

	return s.base.db.Delete(&item).Error
}

func (s *AccountService) ResolveExecutionEnvironment(id uint, provider, region string) (map[string]string, error) {
	var item CloudAccount
	if err := s.base.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	accessKey, err := s.base.cipher.Decrypt(item.AccessKeyEncrypted)
	if err != nil {
		return nil, err
	}
	secretKey, err := s.base.cipher.Decrypt(item.SecretKeyEncrypted)
	if err != nil {
		return nil, err
	}
	provider = normalizeProvider(provider)
	region = strings.TrimSpace(region)
	if region == "" {
		region = strings.TrimSpace(item.Region)
	}
	env := map[string]string{}
	switch provider {
	case "aws":
		env["AWS_ACCESS_KEY_ID"] = accessKey
		env["AWS_SECRET_ACCESS_KEY"] = secretKey
		env["AWS_REGION"] = region
		env["AWS_DEFAULT_REGION"] = region
		env["AWS_EC2_METADATA_DISABLED"] = "true"
	case "alicloud":
		env["ALICLOUD_ACCESS_KEY"] = accessKey
		env["ALICLOUD_SECRET_KEY"] = secretKey
		env["ALICLOUD_REGION"] = region
	}
	return env, nil
}

func (s *AccountService) ListInstanceTypes(id uint, provider, region string) ([]CloudInstanceTypeView, error) {
	provider = normalizeProvider(provider)
	zone := ""
	if provider == "" {
		var account CloudAccount
		if err := s.base.db.First(&account, id).Error; err != nil {
			return nil, err
		}
		provider = normalizeProvider(account.Provider)
		if strings.TrimSpace(region) == "" {
			region = strings.TrimSpace(account.Region)
		}
	}
	switch provider {
	case "aws":
		return s.listAWSInstanceTypes(id, region)
	case "alicloud":
		return s.listAliCloudInstanceTypes(id, region, zone)
	default:
		return nil, fmt.Errorf("当前 provider %s 暂不支持读取机型列表", provider)
	}
}

func (s *AccountService) ListInstanceTypesWithZone(id uint, provider, region, zone string) ([]CloudInstanceTypeView, error) {
	provider = normalizeProvider(provider)
	if provider == "" {
		var account CloudAccount
		if err := s.base.db.First(&account, id).Error; err != nil {
			return nil, err
		}
		provider = normalizeProvider(account.Provider)
		if strings.TrimSpace(region) == "" {
			region = strings.TrimSpace(account.Region)
		}
	}
	switch provider {
	case "aws":
		return s.listAWSInstanceTypes(id, region)
	case "alicloud":
		return s.listAliCloudInstanceTypes(id, region, zone)
	default:
		return nil, fmt.Errorf("当前 provider %s 暂不支持读取机型列表", provider)
	}
}

func (s *AccountService) CreateKeyPair(id uint, input CloudKeyPairInput) (CloudKeyPairView, error) {
	provider := normalizeProvider(input.Provider)
	if provider == "" {
		var account CloudAccount
		if err := s.base.db.First(&account, id).Error; err != nil {
			return CloudKeyPairView{}, err
		}
		provider = normalizeProvider(account.Provider)
		if strings.TrimSpace(input.Region) == "" {
			input.Region = strings.TrimSpace(account.Region)
		}
	}
	switch provider {
	case "aws":
		return s.createAWSKeyPair(id, input)
	case "alicloud":
		return s.createAliCloudKeyPair(id, input)
	default:
		return CloudKeyPairView{}, fmt.Errorf("当前 provider %s 暂不支持云上创建密钥", provider)
	}
}

func (s *AccountService) listAWSInstanceTypes(id uint, region string) ([]CloudInstanceTypeView, error) {
	env, err := s.ResolveExecutionEnvironment(id, "aws", region)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	query := "InstanceTypes[].{instance_type:InstanceType,vcpu:VCpuInfo.DefaultVCpus,memory_mib:MemoryInfo.SizeInMiB,architecture:ProcessorInfo.SupportedArchitectures[0],network:NetworkInfo.NetworkPerformance}"
	cmd := exec.CommandContext(ctx, "aws", "ec2", "describe-instance-types", "--region", strings.TrimSpace(env["AWS_REGION"]), "--output", "json", "--query", query)
	cmd.Env = append(os.Environ(), flattenEnvMap(env)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("读取 AWS 机型失败: %s", strings.TrimSpace(string(output)))
	}

	var items []CloudInstanceTypeView
	if err := json.Unmarshal(output, &items); err != nil {
		return nil, err
	}
	for index := range items {
		items[index].MemoryGiB = formatMemoryGiB(items[index].MemoryMiB)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].InstanceType < items[j].InstanceType
	})
	return items, nil
}

func (s *AccountService) listAliCloudInstanceTypes(id uint, region, zone string) ([]CloudInstanceTypeView, error) {
	env, err := s.ResolveExecutionEnvironment(id, "alicloud", region)
	if err != nil {
		return nil, err
	}

	accessKey := strings.TrimSpace(env["ALICLOUD_ACCESS_KEY"])
	secretKey := strings.TrimSpace(env["ALICLOUD_SECRET_KEY"])
	region = strings.TrimSpace(env["ALICLOUD_REGION"])
	zone = strings.TrimSpace(zone)
	if accessKey == "" || secretKey == "" || region == "" {
		return nil, errors.New("阿里云账号配置不完整")
	}

	params := map[string]string{
		"Action":           "DescribeInstanceTypes",
		"RegionId":         region,
		"PageSize":         "100",
		"PageNumber":       "1",
		"Status":           "Available",
		"AcceptLanguage":   "en-US",
		"Version":          "2014-05-26",
		"Format":           "JSON",
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureVersion": "1.0",
	}
	if zone != "" {
		params["ZoneId"] = zone
	}

	response, err := callAliCloudRPC("https://ecs.aliyuncs.com/", accessKey, secretKey, params)
	if err != nil {
		return nil, err
	}
	var payload struct {
		InstanceTypes struct {
			InstanceType []struct {
				InstanceTypeId       string  `json:"InstanceTypeId"`
				CpuCoreCount         int     `json:"CpuCoreCount"`
				MemorySize           float64 `json:"MemorySize"`
				EniQuantity          int     `json:"EniQuantity"`
				InstanceTypeFamily   string  `json:"InstanceTypeFamily"`
				NetworkCardQuantity  int     `json:"NetworkCardQuantity"`
				LocalStorageCategory string  `json:"LocalStorageCategory"`
			} `json:"InstanceType"`
		} `json:"InstanceTypes"`
		Message   string `json:"Message"`
		Code      string `json:"Code"`
		RequestId string `json:"RequestId"`
	}
	if err := json.Unmarshal(response, &payload); err != nil {
		return nil, err
	}
	if strings.TrimSpace(payload.Code) != "" {
		return nil, fmt.Errorf("读取阿里云机型失败: %s", firstNonEmpty(strings.TrimSpace(payload.Message), strings.TrimSpace(payload.Code)))
	}
	items := make([]CloudInstanceTypeView, 0, len(payload.InstanceTypes.InstanceType))
	for _, item := range payload.InstanceTypes.InstanceType {
		memoryMiB := int64(item.MemorySize * 1024)
		networkLabel := firstNonEmpty(
			strings.TrimSpace(item.InstanceTypeFamily),
			func() string {
				if item.EniQuantity > 0 {
					return fmt.Sprintf("%d ENI", item.EniQuantity)
				}
				if item.NetworkCardQuantity > 0 {
					return fmt.Sprintf("%d NIC", item.NetworkCardQuantity)
				}
				return ""
			}(),
			"enhanced",
		)
		items = append(items, CloudInstanceTypeView{
			InstanceType: item.InstanceTypeId,
			VCPU:         item.CpuCoreCount,
			MemoryMiB:    memoryMiB,
			MemoryGiB:    formatMemoryGiB(memoryMiB),
			Architecture: "x86_64",
			Network:      networkLabel,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].InstanceType < items[j].InstanceType
	})
	return items, nil
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

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+values.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
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

func (s *AccountService) createAWSKeyPair(id uint, input CloudKeyPairInput) (CloudKeyPairView, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return CloudKeyPairView{}, errors.New("密钥名称不能为空")
	}
	env, err := s.ResolveExecutionEnvironment(id, "aws", input.Region)
	if err != nil {
		return CloudKeyPairView{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"aws", "ec2", "create-key-pair",
		"--region", strings.TrimSpace(env["AWS_REGION"]),
		"--key-name", name,
		"--key-type", "rsa",
		"--key-format", "pem",
		"--output", "json",
	)
	cmd.Env = append(os.Environ(), flattenEnvMap(env)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return CloudKeyPairView{}, fmt.Errorf("创建 AWS 密钥失败: %s", strings.TrimSpace(string(output)))
	}
	var payload struct {
		KeyName        string `json:"KeyName"`
		KeyFingerprint string `json:"KeyFingerprint"`
		KeyMaterial    string `json:"KeyMaterial"`
		KeyPairID      string `json:"KeyPairId"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		return CloudKeyPairView{}, err
	}
	return CloudKeyPairView{
		Name:         payload.KeyName,
		Provider:     "aws",
		Region:       strings.TrimSpace(env["AWS_REGION"]),
		Fingerprint:  payload.KeyFingerprint,
		PrivateKey:   payload.KeyMaterial,
		KeyPairID:    payload.KeyPairID,
		CreatedByAPI: true,
	}, nil
}

func (s *AccountService) createAliCloudKeyPair(id uint, input CloudKeyPairInput) (CloudKeyPairView, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return CloudKeyPairView{}, errors.New("密钥名称不能为空")
	}
	env, err := s.ResolveExecutionEnvironment(id, "alicloud", input.Region)
	if err != nil {
		return CloudKeyPairView{}, err
	}

	accessKey := strings.TrimSpace(env["ALICLOUD_ACCESS_KEY"])
	secretKey := strings.TrimSpace(env["ALICLOUD_SECRET_KEY"])
	region := strings.TrimSpace(env["ALICLOUD_REGION"])
	if accessKey == "" || secretKey == "" || region == "" {
		return CloudKeyPairView{}, errors.New("阿里云账号配置不完整")
	}

	params := map[string]string{
		"Action":           "CreateKeyPair",
		"RegionId":         region,
		"KeyPairName":      name,
		"Version":          "2014-05-26",
		"Format":           "JSON",
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureVersion": "1.0",
	}

	response, err := callAliCloudRPC("https://ecs.aliyuncs.com/", accessKey, secretKey, params)
	if err != nil {
		return CloudKeyPairView{}, fmt.Errorf("创建阿里云密钥失败: %s", err.Error())
	}
	var payload struct {
		KeyPairName        string `json:"KeyPairName"`
		KeyPairFingerPrint string `json:"KeyPairFingerPrint"`
		PrivateKeyBody     string `json:"PrivateKeyBody"`
		KeyPairId          string `json:"KeyPairId"`
		Message            string `json:"Message"`
		Code               string `json:"Code"`
	}
	if err := json.Unmarshal(response, &payload); err != nil {
		return CloudKeyPairView{}, err
	}
	if strings.TrimSpace(payload.Code) != "" {
		return CloudKeyPairView{}, fmt.Errorf("创建阿里云密钥失败: %s", firstNonEmpty(strings.TrimSpace(payload.Message), strings.TrimSpace(payload.Code)))
	}
	return CloudKeyPairView{
		Name:         firstNonEmpty(payload.KeyPairName, name),
		Provider:     "alicloud",
		Region:       region,
		Fingerprint:  strings.TrimSpace(payload.KeyPairFingerPrint),
		PrivateKey:   strings.TrimSpace(payload.PrivateKeyBody),
		KeyPairID:    strings.TrimSpace(payload.KeyPairId),
		CreatedByAPI: true,
	}, nil
}

func flattenEnvMap(values map[string]string) []string {
	result := make([]string, 0, len(values))
	for key, value := range values {
		if strings.TrimSpace(key) == "" {
			continue
		}
		result = append(result, key+"="+value)
	}
	sort.Strings(result)
	return result
}

func formatMemoryGiB(memoryMiB int64) string {
	if memoryMiB <= 0 {
		return "0"
	}
	whole := float64(memoryMiB) / 1024.0
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(whole, 'f', 1, 64), "0"), ".")
}
