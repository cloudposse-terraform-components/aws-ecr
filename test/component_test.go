package test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/cloudposse/test-helpers/pkg/atmos"
	helper "github.com/cloudposse/test-helpers/pkg/atmos/component-helper"
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/random"
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

type ComponentSuite struct {
	helper.TestSuite
}

func (s *ComponentSuite) TestBasic() {
	const component = "ecr/basic"
	const stack = "default-test"
	const awsRegion = "us-east-2"

	suffix := strings.ToLower(random.UniqueId())

	inputs := map[string]interface{}{
		"images" : []string{
			fmt.Sprintf("infrastructure-%s", suffix),
			fmt.Sprintf("microservice-a-%s", suffix),
			fmt.Sprintf("microservice-b-%s", suffix),
			fmt.Sprintf("microservice-c-%s", suffix),
		},
	}

	defer s.DestroyAtmosComponent(s.T(), component, stack, &inputs)
	options, _ := s.DeployAtmosComponent(s.T(), component, stack, &inputs)
	assert.NotNil(s.T(), options)

	awsAccountId := aws.GetAccountId(s.T())

	repositoryHost := atmos.Output(s.T(), options, "repository_host")
	assert.Equal(s.T(), fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", awsAccountId, awsRegion), repositoryHost)

	assert.Empty(s.T(), atmos.Output(s.T(), options, "ecr_user_name"))
	assert.Empty(s.T(), atmos.Output(s.T(), options, "ecr_user_arn"))
	assert.Empty(s.T(), atmos.Output(s.T(), options, "ecr_user_unique_id"))

	arnMaps := map[string]string{}
	atmos.OutputStruct(s.T(), options, "ecr_repo_arn_map", &arnMaps)

	urlMaps := map[string]string{}
	atmos.OutputStruct(s.T(), options, "ecr_repo_url_map", &urlMaps)

	for name, arn := range arnMaps {
		repository := aws.GetECRRepo(s.T(), awsRegion, name)
		assert.Equal(s.T(), name, *repository.RepositoryName)
		assert.Equal(s.T(), arn, *repository.RepositoryArn)
		assert.Equal(s.T(), urlMaps[name], *repository.RepositoryUri)
		assert.EqualValues(s.T(), "IMMUTABLE", repository.ImageTagMutability)
		assert.True(s.T(), repository.ImageScanningConfiguration.ScanOnPush)
		assert.EqualValues(s.T(), "AES256", repository.EncryptionConfiguration.EncryptionType)

		lifecyclePolicyString := aws.GetECRRepoLifecyclePolicy(s.T(), awsRegion, repository)
		lifecyclePolicy := LifecyclePolicy{}
		json.Unmarshal([]byte(lifecyclePolicyString), &lifecyclePolicy)

		expectedLifecyclePolicy := LifecyclePolicy{
			Rules: []LifecyclePolicyRule{
				{
					RulePriority: 1,
					Description:  "Protects images tagged with prefix prod",
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
		assert.EqualValues(s.T(), expectedLifecyclePolicy, lifecyclePolicy)
	}

	s.DriftTest(component, stack, &inputs)
}

func (s *ComponentSuite) TestEnabledFlag() {
	const component = "ecr/disabled"
	const stack = "default-test"
	const awsRegion = "us-east-2"

	s.VerifyEnabledFlag(component, stack, nil)
}

func TestRunSuite(t *testing.T) {
	suite := new(ComponentSuite)
	helper.Run(t, suite)
}
