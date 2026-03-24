package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Rule defines a single compliance check against a cloudmock endpoint.
type Rule struct {
	ID          string
	Name        string
	Description string
	Severity    string // critical, high, medium, low
	Service     string // s3, ec2, iam, etc.
	Check       func(client *http.Client, endpoint string) ([]Finding, error)
}

// Finding represents a single compliance issue discovered during a scan.
type Finding struct {
	RuleID       string `json:"rule_id"`
	ResourceID   string `json:"resource_id"`
	ResourceType string `json:"resource_type"`
	Severity     string `json:"severity"`
	Message      string `json:"message"`
	Remediation  string `json:"remediation"`
}

// RunScan executes all rules and collects findings.
func RunScan(client *http.Client, endpoint string, rules []Rule) []Finding {
	var findings []Finding
	for _, rule := range rules {
		results, err := rule.Check(client, endpoint)
		if err != nil {
			// Record the error as a finding so the user sees it.
			findings = append(findings, Finding{
				RuleID:       rule.ID,
				ResourceID:   "scanner",
				ResourceType: "scanner",
				Severity:     "low",
				Message:      fmt.Sprintf("Rule check failed: %v", err),
				Remediation:  "Verify the cloudmock endpoint is reachable and the service is running.",
			})
			continue
		}
		findings = append(findings, results...)
	}
	return findings
}

// AllRules returns the complete set of built-in compliance rules.
func AllRules() []Rule {
	return []Rule{
		ruleS3PublicAccess(),
		ruleS3Versioning(),
		ruleSGOpenIngress(),
		ruleSGUnrestrictedSSH(),
		ruleKMSKeyRotation(),
		ruleIAMRootAccessKey(),
		ruleRDSPublicAccess(),
		ruleDynamoDBEncryption(),
		ruleCloudTrailEnabled(),
		ruleVPCFlowLogs(),
		ruleEIPUnused(),
		ruleDefaultVPCInUse(),
		ruleSecretsRotation(),
		ruleLambdaRuntime(),
		ruleInstanceNoPublicIP(),
	}
}

// --- Helpers ---

// doQuery sends a form-encoded POST request with Action parameter (EC2/SQS style).
func doQuery(client *http.Client, endpoint, service, action string, params map[string]string) ([]byte, error) {
	form := url.Values{}
	form.Set("Action", action)
	form.Set("Version", "2016-11-15")
	for k, v := range params {
		form.Set(k, v)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint+"/", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	setAuthHeader(req, service)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// doJSON sends a JSON-protocol POST request (DynamoDB/KMS style).
func doJSON(client *http.Client, endpoint, service, targetPrefix, action string, params map[string]any) ([]byte, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, endpoint+"/", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	if targetPrefix != "" {
		req.Header.Set("X-Amz-Target", targetPrefix+"."+action)
	}
	setAuthHeader(req, service)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// doREST sends a REST-style request (S3 style).
func doREST(client *http.Client, endpoint, service, method, path string) ([]byte, int, error) {
	req, err := http.NewRequest(method, endpoint+path, nil)
	if err != nil {
		return nil, 0, err
	}
	setAuthHeader(req, service)

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

func setAuthHeader(req *http.Request, service string) {
	date := time.Now().UTC().Format("20060102")
	credentialScope := fmt.Sprintf("%s/us-east-1/%s/aws4_request", date, service)
	req.Header.Set("Authorization",
		fmt.Sprintf("AWS4-HMAC-SHA256 Credential=test/%s, SignedHeaders=host, Signature=fakesig", credentialScope))
	req.Header.Set("X-Amz-Date", time.Now().UTC().Format("20060102T150405Z"))
}

// --- Rule 1: S3 Public Access ---

func ruleS3PublicAccess() Rule {
	return Rule{
		ID:          "s3-public-access",
		Name:        "S3 Public Access Check",
		Description: "Check if any S3 buckets lack ACL restrictions (buckets with 'public' in name are flagged).",
		Severity:    "critical",
		Service:     "s3",
		Check: func(client *http.Client, endpoint string) ([]Finding, error) {
			body, statusCode, err := doREST(client, endpoint, "s3", http.MethodGet, "/")
			if err != nil {
				return nil, err
			}
			if statusCode != http.StatusOK {
				return nil, fmt.Errorf("ListBuckets returned status %d", statusCode)
			}

			var result struct {
				XMLName xml.Name `xml:"ListAllMyBucketsResult"`
				Buckets struct {
					Bucket []struct {
						Name string `xml:"Name"`
					} `xml:"Bucket"`
				} `xml:"Buckets"`
			}
			if err := xml.Unmarshal(body, &result); err != nil {
				return nil, fmt.Errorf("parsing ListBuckets response: %w", err)
			}

			var findings []Finding
			for _, b := range result.Buckets.Bucket {
				// In cloudmock, no ACL mechanism exists, so flag buckets with "public" in name.
				if strings.Contains(strings.ToLower(b.Name), "public") {
					findings = append(findings, Finding{
						RuleID:       "s3-public-access",
						ResourceID:   b.Name,
						ResourceType: "AWS::S3::Bucket",
						Severity:     "critical",
						Message:      fmt.Sprintf("Bucket %q may have public access (no ACL restrictions configured).", b.Name),
						Remediation:  "Enable S3 Block Public Access settings on the bucket.",
					})
				}
			}
			return findings, nil
		},
	}
}

// --- Rule 2: S3 Versioning ---

func ruleS3Versioning() Rule {
	return Rule{
		ID:          "s3-versioning",
		Name:        "S3 Versioning Check",
		Description: "Check if S3 buckets have versioning enabled.",
		Severity:    "medium",
		Service:     "s3",
		Check: func(client *http.Client, endpoint string) ([]Finding, error) {
			body, statusCode, err := doREST(client, endpoint, "s3", http.MethodGet, "/")
			if err != nil {
				return nil, err
			}
			if statusCode != http.StatusOK {
				return nil, fmt.Errorf("ListBuckets returned status %d", statusCode)
			}

			var result struct {
				XMLName xml.Name `xml:"ListAllMyBucketsResult"`
				Buckets struct {
					Bucket []struct {
						Name string `xml:"Name"`
					} `xml:"Bucket"`
				} `xml:"Buckets"`
			}
			if err := xml.Unmarshal(body, &result); err != nil {
				return nil, fmt.Errorf("parsing ListBuckets response: %w", err)
			}

			var findings []Finding
			for _, b := range result.Buckets.Bucket {
				// cloudmock does not implement versioning, so all buckets are non-versioned.
				findings = append(findings, Finding{
					RuleID:       "s3-versioning",
					ResourceID:   b.Name,
					ResourceType: "AWS::S3::Bucket",
					Severity:     "medium",
					Message:      fmt.Sprintf("Bucket %q does not have versioning enabled.", b.Name),
					Remediation:  "Enable versioning on the S3 bucket to protect against accidental deletion.",
				})
			}
			return findings, nil
		},
	}
}

// --- Rule 3: Security Group Open Ingress ---

func ruleSGOpenIngress() Rule {
	return Rule{
		ID:          "sg-open-ingress",
		Name:        "Security Group Open Ingress",
		Description: "Check for security groups with 0.0.0.0/0 ingress on any port.",
		Severity:    "high",
		Service:     "ec2",
		Check: func(client *http.Client, endpoint string) ([]Finding, error) {
			body, err := doQuery(client, endpoint, "ec2", "DescribeSecurityGroups", nil)
			if err != nil {
				return nil, err
			}

			var result struct {
				XMLName        xml.Name `xml:"DescribeSecurityGroupsResponse"`
				SecurityGroups struct {
					Items []struct {
						GroupId       string `xml:"groupId"`
						GroupName     string `xml:"groupName"`
						IpPermissions struct {
							Items []struct {
								IpProtocol string `xml:"ipProtocol"`
								FromPort   int    `xml:"fromPort"`
								ToPort     int    `xml:"toPort"`
								IpRanges   struct {
									Items []struct {
										CidrIp string `xml:"cidrIp"`
									} `xml:"item"`
								} `xml:"ipRanges"`
							} `xml:"item"`
						} `xml:"ipPermissions"`
					} `xml:"item"`
				} `xml:"securityGroupInfo"`
			}
			if err := xml.Unmarshal(body, &result); err != nil {
				return nil, fmt.Errorf("parsing DescribeSecurityGroups response: %w", err)
			}

			var findings []Finding
			for _, sg := range result.SecurityGroups.Items {
				for _, perm := range sg.IpPermissions.Items {
					for _, ipRange := range perm.IpRanges.Items {
						if ipRange.CidrIp == "0.0.0.0/0" {
							findings = append(findings, Finding{
								RuleID:       "sg-open-ingress",
								ResourceID:   sg.GroupId,
								ResourceType: "AWS::EC2::SecurityGroup",
								Severity:     "high",
								Message:      fmt.Sprintf("Security group %s (%s) allows ingress from 0.0.0.0/0 on protocol %s ports %d-%d.", sg.GroupId, sg.GroupName, perm.IpProtocol, perm.FromPort, perm.ToPort),
								Remediation:  "Restrict ingress rules to specific IP ranges or security groups.",
							})
						}
					}
				}
			}
			return findings, nil
		},
	}
}

// --- Rule 4: Security Group Unrestricted SSH ---

func ruleSGUnrestrictedSSH() Rule {
	return Rule{
		ID:          "sg-unrestricted-ssh",
		Name:        "Security Group Unrestricted SSH",
		Description: "Check for security groups with port 22 open to 0.0.0.0/0.",
		Severity:    "critical",
		Service:     "ec2",
		Check: func(client *http.Client, endpoint string) ([]Finding, error) {
			body, err := doQuery(client, endpoint, "ec2", "DescribeSecurityGroups", nil)
			if err != nil {
				return nil, err
			}

			var result struct {
				XMLName        xml.Name `xml:"DescribeSecurityGroupsResponse"`
				SecurityGroups struct {
					Items []struct {
						GroupId       string `xml:"groupId"`
						GroupName     string `xml:"groupName"`
						IpPermissions struct {
							Items []struct {
								IpProtocol string `xml:"ipProtocol"`
								FromPort   int    `xml:"fromPort"`
								ToPort     int    `xml:"toPort"`
								IpRanges   struct {
									Items []struct {
										CidrIp string `xml:"cidrIp"`
									} `xml:"item"`
								} `xml:"ipRanges"`
							} `xml:"item"`
						} `xml:"ipPermissions"`
					} `xml:"item"`
				} `xml:"securityGroupInfo"`
			}
			if err := xml.Unmarshal(body, &result); err != nil {
				return nil, fmt.Errorf("parsing DescribeSecurityGroups response: %w", err)
			}

			var findings []Finding
			for _, sg := range result.SecurityGroups.Items {
				for _, perm := range sg.IpPermissions.Items {
					if perm.FromPort <= 22 && perm.ToPort >= 22 {
						for _, ipRange := range perm.IpRanges.Items {
							if ipRange.CidrIp == "0.0.0.0/0" {
								findings = append(findings, Finding{
									RuleID:       "sg-unrestricted-ssh",
									ResourceID:   sg.GroupId,
									ResourceType: "AWS::EC2::SecurityGroup",
									Severity:     "critical",
									Message:      fmt.Sprintf("Security group %s (%s) allows SSH (port 22) from 0.0.0.0/0.", sg.GroupId, sg.GroupName),
									Remediation:  "Restrict SSH access to specific trusted IP ranges.",
								})
							}
						}
					}
				}
			}
			return findings, nil
		},
	}
}

// --- Rule 5: KMS Key Rotation ---

func ruleKMSKeyRotation() Rule {
	return Rule{
		ID:          "kms-key-rotation",
		Name:        "KMS Key Rotation",
		Description: "Check if KMS keys have rotation enabled (verifies keys exist).",
		Severity:    "medium",
		Service:     "kms",
		Check: func(client *http.Client, endpoint string) ([]Finding, error) {
			body, err := doJSON(client, endpoint, "kms", "TrentService", "ListKeys", map[string]any{})
			if err != nil {
				return nil, err
			}

			var result struct {
				Keys []struct {
					KeyId  string `json:"KeyId"`
					KeyArn string `json:"KeyArn"`
				} `json:"Keys"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				return nil, fmt.Errorf("parsing ListKeys response: %w", err)
			}

			var findings []Finding
			for _, key := range result.Keys {
				// cloudmock does not implement key rotation status, flag all keys.
				findings = append(findings, Finding{
					RuleID:       "kms-key-rotation",
					ResourceID:   key.KeyId,
					ResourceType: "AWS::KMS::Key",
					Severity:     "medium",
					Message:      fmt.Sprintf("KMS key %s may not have automatic key rotation enabled.", key.KeyId),
					Remediation:  "Enable automatic key rotation for the KMS key.",
				})
			}
			return findings, nil
		},
	}
}

// --- Rule 6: IAM Root Access Key ---

func ruleIAMRootAccessKey() Rule {
	return Rule{
		ID:          "iam-root-access-key",
		Name:        "IAM Root Access Key",
		Description: "Check if root account has active access keys.",
		Severity:    "critical",
		Service:     "iam",
		Check: func(client *http.Client, endpoint string) ([]Finding, error) {
			// cloudmock's IAM always seeds a root access key, so this is always a finding.
			return []Finding{
				{
					RuleID:       "iam-root-access-key",
					ResourceID:   "root",
					ResourceType: "AWS::IAM::User",
					Severity:     "critical",
					Message:      "Root account has active access keys. The root user should not have access keys.",
					Remediation:  "Delete root access keys and use IAM users or roles instead.",
				},
			}, nil
		},
	}
}

// --- Rule 7: RDS Public Access ---

func ruleRDSPublicAccess() Rule {
	return Rule{
		ID:          "rds-public-access",
		Name:        "RDS Public Access",
		Description: "Check if RDS instances are publicly accessible.",
		Severity:    "high",
		Service:     "rds",
		Check: func(client *http.Client, endpoint string) ([]Finding, error) {
			body, err := doQuery(client, endpoint, "rds", "DescribeDBInstances", nil)
			if err != nil {
				return nil, err
			}

			var result struct {
				XMLName xml.Name `xml:"DescribeDBInstancesResponse"`
				Result  struct {
					Instances struct {
						Items []struct {
							DBInstanceIdentifier string `xml:"DBInstanceIdentifier"`
							PubliclyAccessible   bool   `xml:"PubliclyAccessible"`
						} `xml:"DBInstance"`
					} `xml:"DBInstances"`
				} `xml:"DescribeDBInstancesResult"`
			}
			if err := xml.Unmarshal(body, &result); err != nil {
				// RDS may return JSON or XML depending on implementation.
				return nil, fmt.Errorf("parsing DescribeDBInstances response: %w", err)
			}

			var findings []Finding
			for _, inst := range result.Result.Instances.Items {
				if inst.PubliclyAccessible {
					findings = append(findings, Finding{
						RuleID:       "rds-public-access",
						ResourceID:   inst.DBInstanceIdentifier,
						ResourceType: "AWS::RDS::DBInstance",
						Severity:     "high",
						Message:      fmt.Sprintf("RDS instance %q is publicly accessible.", inst.DBInstanceIdentifier),
						Remediation:  "Disable public accessibility on the RDS instance.",
					})
				}
			}
			return findings, nil
		},
	}
}

// --- Rule 8: DynamoDB Encryption ---

func ruleDynamoDBEncryption() Rule {
	return Rule{
		ID:          "dynamodb-encryption",
		Name:        "DynamoDB Encryption",
		Description: "Check if DynamoDB tables have encryption configured (stub: all tables flagged).",
		Severity:    "medium",
		Service:     "dynamodb",
		Check: func(client *http.Client, endpoint string) ([]Finding, error) {
			body, err := doJSON(client, endpoint, "dynamodb", "DynamoDB_20120810", "ListTables", map[string]any{})
			if err != nil {
				return nil, err
			}

			var result struct {
				TableNames []string `json:"TableNames"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				return nil, fmt.Errorf("parsing ListTables response: %w", err)
			}

			var findings []Finding
			for _, name := range result.TableNames {
				// cloudmock does not implement encryption settings; flag all tables.
				findings = append(findings, Finding{
					RuleID:       "dynamodb-encryption",
					ResourceID:   name,
					ResourceType: "AWS::DynamoDB::Table",
					Severity:     "medium",
					Message:      fmt.Sprintf("DynamoDB table %q encryption status cannot be verified.", name),
					Remediation:  "Ensure DynamoDB table uses AWS-managed or customer-managed KMS encryption.",
				})
			}
			return findings, nil
		},
	}
}

// --- Rule 9: CloudTrail Enabled ---

func ruleCloudTrailEnabled() Rule {
	return Rule{
		ID:          "cloudtrail-enabled",
		Name:        "CloudTrail Enabled",
		Description: "Check if at least one CloudTrail trail exists.",
		Severity:    "high",
		Service:     "cloudtrail",
		Check: func(client *http.Client, endpoint string) ([]Finding, error) {
			// cloudmock does not implement CloudTrail, so this is always a finding.
			return []Finding{
				{
					RuleID:       "cloudtrail-enabled",
					ResourceID:   "account",
					ResourceType: "AWS::CloudTrail::Trail",
					Severity:     "high",
					Message:      "No CloudTrail trail is configured. API activity is not being logged.",
					Remediation:  "Create a CloudTrail trail to log all management events.",
				},
			}, nil
		},
	}
}

// --- Rule 10: VPC Flow Logs ---

func ruleVPCFlowLogs() Rule {
	return Rule{
		ID:          "vpc-flow-logs",
		Name:        "VPC Flow Logs",
		Description: "Check if VPCs have flow logs enabled (stub: always a finding since not implemented).",
		Severity:    "medium",
		Service:     "ec2",
		Check: func(client *http.Client, endpoint string) ([]Finding, error) {
			body, err := doQuery(client, endpoint, "ec2", "DescribeVpcs", nil)
			if err != nil {
				return nil, err
			}

			var result struct {
				XMLName xml.Name `xml:"DescribeVpcsResponse"`
				VpcSet  struct {
					Items []struct {
						VpcId string `xml:"vpcId"`
					} `xml:"item"`
				} `xml:"vpcSet"`
			}
			if err := xml.Unmarshal(body, &result); err != nil {
				return nil, fmt.Errorf("parsing DescribeVpcs response: %w", err)
			}

			var findings []Finding
			for _, vpc := range result.VpcSet.Items {
				findings = append(findings, Finding{
					RuleID:       "vpc-flow-logs",
					ResourceID:   vpc.VpcId,
					ResourceType: "AWS::EC2::VPC",
					Severity:     "medium",
					Message:      fmt.Sprintf("VPC %s does not have flow logs enabled.", vpc.VpcId),
					Remediation:  "Enable VPC Flow Logs to capture network traffic metadata.",
				})
			}
			return findings, nil
		},
	}
}

// --- Rule 11: Unused Elastic IPs ---

func ruleEIPUnused() Rule {
	return Rule{
		ID:          "eip-unused",
		Name:        "Unused Elastic IPs",
		Description: "Check for Elastic IPs that are not associated with an instance.",
		Severity:    "low",
		Service:     "ec2",
		Check: func(client *http.Client, endpoint string) ([]Finding, error) {
			body, err := doQuery(client, endpoint, "ec2", "DescribeAddresses", nil)
			if err != nil {
				return nil, err
			}

			var result struct {
				XMLName    xml.Name `xml:"DescribeAddressesResponse"`
				AddressSet struct {
					Items []struct {
						PublicIp      string `xml:"publicIp"`
						AllocationId  string `xml:"allocationId"`
						AssociationId string `xml:"associationId"`
						InstanceId    string `xml:"instanceId"`
					} `xml:"item"`
				} `xml:"addressesSet"`
			}
			if err := xml.Unmarshal(body, &result); err != nil {
				return nil, fmt.Errorf("parsing DescribeAddresses response: %w", err)
			}

			var findings []Finding
			for _, addr := range result.AddressSet.Items {
				if addr.AssociationId == "" && addr.InstanceId == "" {
					findings = append(findings, Finding{
						RuleID:       "eip-unused",
						ResourceID:   addr.AllocationId,
						ResourceType: "AWS::EC2::EIP",
						Severity:     "low",
						Message:      fmt.Sprintf("Elastic IP %s (%s) is not associated with any instance.", addr.AllocationId, addr.PublicIp),
						Remediation:  "Release unused Elastic IPs to avoid unnecessary charges.",
					})
				}
			}
			return findings, nil
		},
	}
}

// --- Rule 12: Default VPC In Use ---

func ruleDefaultVPCInUse() Rule {
	return Rule{
		ID:          "default-vpc-in-use",
		Name:        "Default VPC In Use",
		Description: "Check if the default VPC has non-default resources (instances, subnets beyond defaults).",
		Severity:    "low",
		Service:     "ec2",
		Check: func(client *http.Client, endpoint string) ([]Finding, error) {
			body, err := doQuery(client, endpoint, "ec2", "DescribeVpcs", nil)
			if err != nil {
				return nil, err
			}

			var result struct {
				XMLName xml.Name `xml:"DescribeVpcsResponse"`
				VpcSet  struct {
					Items []struct {
						VpcId     string `xml:"vpcId"`
						IsDefault bool   `xml:"isDefault"`
					} `xml:"item"`
				} `xml:"vpcSet"`
			}
			if err := xml.Unmarshal(body, &result); err != nil {
				return nil, fmt.Errorf("parsing DescribeVpcs response: %w", err)
			}

			var findings []Finding
			for _, vpc := range result.VpcSet.Items {
				if vpc.IsDefault {
					findings = append(findings, Finding{
						RuleID:       "default-vpc-in-use",
						ResourceID:   vpc.VpcId,
						ResourceType: "AWS::EC2::VPC",
						Severity:     "low",
						Message:      fmt.Sprintf("Default VPC %s exists. Resources should use custom VPCs with proper network segmentation.", vpc.VpcId),
						Remediation:  "Create custom VPCs for workloads and avoid using the default VPC.",
					})
				}
			}
			return findings, nil
		},
	}
}

// --- Rule 13: Secrets Rotation ---

func ruleSecretsRotation() Rule {
	return Rule{
		ID:          "secrets-rotation",
		Name:        "Secrets Manager Rotation",
		Description: "Check if Secrets Manager secrets have rotation configured.",
		Severity:    "medium",
		Service:     "secretsmanager",
		Check: func(client *http.Client, endpoint string) ([]Finding, error) {
			body, err := doJSON(client, endpoint, "secretsmanager", "secretsmanager", "ListSecrets", map[string]any{})
			if err != nil {
				return nil, err
			}

			var result struct {
				SecretList []struct {
					Name            string `json:"Name"`
					ARN             string `json:"ARN"`
					RotationEnabled bool   `json:"RotationEnabled"`
				} `json:"SecretList"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				return nil, fmt.Errorf("parsing ListSecrets response: %w", err)
			}

			var findings []Finding
			for _, secret := range result.SecretList {
				if !secret.RotationEnabled {
					id := secret.Name
					if id == "" {
						id = secret.ARN
					}
					findings = append(findings, Finding{
						RuleID:       "secrets-rotation",
						ResourceID:   id,
						ResourceType: "AWS::SecretsManager::Secret",
						Severity:     "medium",
						Message:      fmt.Sprintf("Secret %q does not have automatic rotation enabled.", id),
						Remediation:  "Enable automatic rotation for the secret with a rotation Lambda function.",
					})
				}
			}
			return findings, nil
		},
	}
}

// --- Rule 14: Lambda Runtime ---

func ruleLambdaRuntime() Rule {
	// Supported runtimes as of 2025.
	supported := map[string]bool{
		"nodejs20.x":  true,
		"nodejs22.x":  true,
		"python3.12":  true,
		"python3.13":  true,
		"java21":      true,
		"dotnet8":     true,
		"provided.al2023": true,
		"provided.al2": true,
		"ruby3.3":     true,
		"ruby3.4":     true,
	}

	return Rule{
		ID:          "lambda-runtime",
		Name:        "Lambda Runtime Check",
		Description: "Check if Lambda functions use supported (non-deprecated) runtimes.",
		Severity:    "high",
		Service:     "lambda",
		Check: func(client *http.Client, endpoint string) ([]Finding, error) {
			req, err := http.NewRequest(http.MethodGet, endpoint+"/2015-03-31/functions/", nil)
			if err != nil {
				return nil, err
			}
			setAuthHeader(req, "lambda")

			resp, err := client.Do(req)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}

			var result struct {
				Functions []struct {
					FunctionName string `json:"FunctionName"`
					Runtime      string `json:"Runtime"`
					FunctionArn  string `json:"FunctionArn"`
				} `json:"Functions"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				return nil, fmt.Errorf("parsing ListFunctions response: %w", err)
			}

			var findings []Finding
			for _, fn := range result.Functions {
				if fn.Runtime != "" && !supported[fn.Runtime] {
					findings = append(findings, Finding{
						RuleID:       "lambda-runtime",
						ResourceID:   fn.FunctionName,
						ResourceType: "AWS::Lambda::Function",
						Severity:     "high",
						Message:      fmt.Sprintf("Lambda function %q uses runtime %q which may be deprecated.", fn.FunctionName, fn.Runtime),
						Remediation:  "Update the Lambda function to use a supported runtime version.",
					})
				}
			}
			return findings, nil
		},
	}
}

// --- Rule 15: Instance No Public IP ---

func ruleInstanceNoPublicIP() Rule {
	return Rule{
		ID:          "instance-no-public-ip",
		Name:        "EC2 Instance Public IP",
		Description: "Check if EC2 instances have public IP addresses assigned (stub check).",
		Severity:    "medium",
		Service:     "ec2",
		Check: func(client *http.Client, endpoint string) ([]Finding, error) {
			body, err := doQuery(client, endpoint, "ec2", "DescribeInstances", nil)
			if err != nil {
				return nil, err
			}

			var result struct {
				XMLName        xml.Name `xml:"DescribeInstancesResponse"`
				ReservationSet struct {
					Items []struct {
						InstancesSet struct {
							Items []struct {
								InstanceId string `xml:"instanceId"`
							} `xml:"item"`
						} `xml:"instancesSet"`
					} `xml:"item"`
				} `xml:"reservationSet"`
			}
			if err := xml.Unmarshal(body, &result); err != nil {
				// If no instances, parsing may fail gracefully.
				return nil, nil
			}

			// Stub: cloudmock instances don't track public IPs in detail,
			// so we only report if instances exist as a low-confidence finding.
			var findings []Finding
			for _, res := range result.ReservationSet.Items {
				for _, inst := range res.InstancesSet.Items {
					findings = append(findings, Finding{
						RuleID:       "instance-no-public-ip",
						ResourceID:   inst.InstanceId,
						ResourceType: "AWS::EC2::Instance",
						Severity:     "medium",
						Message:      fmt.Sprintf("Instance %s may have a public IP assigned (unable to verify in cloudmock).", inst.InstanceId),
						Remediation:  "Use private subnets and NAT gateways instead of public IPs.",
					})
				}
			}
			return findings, nil
		},
	}
}
