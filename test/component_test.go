package test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cloudposse/test-helpers/pkg/atmos"
	helper "github.com/cloudposse/test-helpers/pkg/atmos/aws-component-helper"
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/stretchr/testify/assert"
)

type LifecyclePolicyRuleSelection struct {
	TagStatus     string   `json:"tagStatus"`
	TagPrefixList []string `json:"tagPrefixList"`
	CountType     string   `json:"countType"`
	CountNumber   int      `json:"countNumber"`
}
type LifecyclePolicyRule struct {
	RulePriority int                          `json:"rulePriority"`
	Description  string                       `json:"description"`
	Selection    LifecyclePolicyRuleSelection `json:"selection"`
	Action       map[string]string            `json:"action"`
}
type LifecyclePolicy struct {
	Rules []LifecyclePolicyRule `json:"rules"`
}

func TestComponent(t *testing.T) {
	awsRegion := "us-east-2"

	fixture := helper.NewFixture(t, "../", awsRegion, "test/fixtures")

	defer fixture.TearDown()
	fixture.SetUp(&atmos.Options{})

	fixture.Suite("default", func(t *testing.T, suite *helper.Suite) {
		suite.Test(t, "basic", func(t *testing.T, atm *helper.Atmos) {
			defer atm.GetAndDestroy("ecr/basic", "default-test", map[string]interface{}{})
			component := atm.GetAndDeploy("ecr/basic", "default-test", map[string]interface{}{})
			assert.NotNil(t, component)

			repositoryHost := atm.Output(component, "repository_host")
			assert.Equal(t, fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", fixture.AwsAccountId, awsRegion), repositoryHost)

			ecrUserName := atm.Output(component, "ecr_user_name")
			assert.Empty(t, ecrUserName)

			ecrUserArn := atm.Output(component, "ecr_user_arn")
			assert.Empty(t, ecrUserArn)

			ecrUserUniqueId := atm.Output(component, "ecr_user_unique_id")
			assert.Empty(t, ecrUserUniqueId)

			arnMaps := map[string]string{}
			atm.OutputStruct(component, "ecr_repo_arn_map", &arnMaps)

			urlMaps := map[string]string{}
			atm.OutputStruct(component, "ecr_repo_url_map", &urlMaps)

			for name, arn := range arnMaps {
				repository := aws.GetECRRepo(t, awsRegion, name)
				assert.Equal(t, name, *repository.RepositoryName)
				assert.Equal(t, arn, *repository.RepositoryArn)
				assert.Equal(t, urlMaps[name], *repository.RepositoryUri)
				assert.EqualValues(t, "IMMUTABLE", repository.ImageTagMutability)
				assert.True(t, repository.ImageScanningConfiguration.ScanOnPush)
				assert.EqualValues(t, "AES256", repository.EncryptionConfiguration.EncryptionType)

				lifecyclePolicyString := aws.GetECRRepoLifecyclePolicy(t, awsRegion, repository)
				lifecyclePolicy := LifecyclePolicy{}
				json.Unmarshal([]byte(lifecyclePolicyString), &lifecyclePolicy)

				expectedLifecyclePolicy := LifecyclePolicy{
					Rules: []LifecyclePolicyRule{
						{
							RulePriority: 1,
							Description:  "Protects images tagged with prod",
							Selection: LifecyclePolicyRuleSelection{
								TagStatus:     "tagged",
								TagPrefixList: []string{"prod"},
								CountType:     "imageCountMoreThan",
								CountNumber:   999999,
							},
							Action: map[string]string{
								"type": "expire",
							},
						},
						{
							RulePriority: 2,
							Description:  "Remove untagged images",
							Selection: LifecyclePolicyRuleSelection{
								TagStatus:   "untagged",
								CountType:   "imageCountMoreThan",
								CountNumber: 1,
							},
							Action: map[string]string{
								"type": "expire",
							},
						},
						{
							RulePriority: 3,
							Description:  "Rotate images when reach 500 images stored",
							Selection: LifecyclePolicyRuleSelection{
								TagStatus:   "any",
								CountType:   "imageCountMoreThan",
								CountNumber: 500,
							},
							Action: map[string]string{
								"type": "expire",
							},
						},
					},
				}
				assert.EqualValues(t, expectedLifecyclePolicy, lifecyclePolicy)
			}
		})
	})
}
