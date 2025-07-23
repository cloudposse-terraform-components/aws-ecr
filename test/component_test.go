package test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cloudposse/test-helpers/pkg/atmos"
	helper "github.com/cloudposse/test-helpers/pkg/atmos/component-helper"
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

type ComponentSuite struct {
	helper.TestSuite
}

func (s *ComponentSuite) TestBasic() {
	const component = "ecr/basic"
	const stack = "default-test"
	const awsRegion = "us-east-2"
	const repositoryName = "infrastructure"

	defer s.DestroyAtmosComponent(s.T(), component, stack, nil)
	options, _ := s.DeployAtmosComponent(s.T(), component, stack, nil)
	assert.NotNil(s.T(), options)

	awsAccountId := aws.GetAccountId(s.T())

	arn := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/repository/%s", awsAccountId, awsRegion, repositoryName)

	assert.Empty(s.T(), atmos.Output(s.T(), options, "ecr_user_name"))
	assert.Empty(s.T(), atmos.Output(s.T(), options, "ecr_user_arn"))
	assert.Empty(s.T(), atmos.Output(s.T(), options, "ecr_user_unique_id"))

	repository := aws.GetECRRepo(s.T(), awsRegion, repositoryName)
	assert.Equal(s.T(), repositoryName, *repository.RepositoryName)
	assert.Equal(s.T(), arn, *repository.RepositoryArn)
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

	s.DriftTest(component, stack, &inputs)
}

func (s *ComponentSuite) TestMicroserviceA() {
	const component = "ecr/microservice-a"
	const stack = "default-test"
	const awsRegion = "us-east-2"
	const repositoryName = "microservice-a"

	defer s.DestroyAtmosComponent(s.T(), component, stack, nil)
	options, _ := s.DeployAtmosComponent(s.T(), component, stack, nil)
	assert.NotNil(s.T(), options)

	awsAccountId := aws.GetAccountId(s.T())

	arn := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/repository/%s", awsAccountId, awsRegion, repositoryName)

	assert.Empty(s.T(), atmos.Output(s.T(), options, "ecr_user_name"))
	assert.Empty(s.T(), atmos.Output(s.T(), options, "ecr_user_arn"))
	assert.Empty(s.T(), atmos.Output(s.T(), options, "ecr_user_unique_id"))

	repository := aws.GetECRRepo(s.T(), awsRegion, repositoryName)
	assert.Equal(s.T(), repositoryName, *repository.RepositoryName)
	assert.Equal(s.T(), arn, *repository.RepositoryArn)
	assert.EqualValues(s.T(), "IMMUTABLE", repository.ImageTagMutability)
	assert.True(s.T(), repository.ImageScanningConfiguration.ScanOnPush)
	assert.EqualValues(s.T(), "AES256", repository.EncryptionConfiguration.EncryptionType)

	lifecyclePolicyString := aws.GetECRRepoLifecyclePolicy(s.T(), awsRegion, repository)
	lifecyclePolicy := LifecyclePolicy{}
	json.Unmarshal([]byte(lifecyclePolicyString), &lifecyclePolicy)
	
	expectedLifecyclePolicy := LifecyclePolicy{
		Rules: []LifecyclePolicyRule{
			{
				RulePriority: 10,
				Description:  "only keep 10 images",
				Selection: LifecyclePolicyRuleSelection{
					TagStatus:     "any",
					CountType:     "imageCountMoreThan",
					CountNumber:   10,
				},
				Action: map[string]string{
					"type": "expire",
				},
			}
		},
	}
	assert.EqualValues(s.T(), expectedLifecyclePolicy, lifecyclePolicy)

	s.DriftTest(component, stack, &inputs)
}



func (s *ComponentSuite) TestMicroserviceB() {
	const component = "ecr/microservice-b"
	const stack = "default-test"
	const awsRegion = "us-east-2"
	const repositoryName = "microservice-b"

	defer s.DestroyAtmosComponent(s.T(), component, stack, nil)
	options, _ := s.DeployAtmosComponent(s.T(), component, stack, nil)
	assert.NotNil(s.T(), options)

	awsAccountId := aws.GetAccountId(s.T())

	arn := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/repository/%s", awsAccountId, awsRegion, repositoryName)

	assert.Empty(s.T(), atmos.Output(s.T(), options, "ecr_user_name"))
	assert.Empty(s.T(), atmos.Output(s.T(), options, "ecr_user_arn"))
	assert.Empty(s.T(), atmos.Output(s.T(), options, "ecr_user_unique_id"))

	repository := aws.GetECRRepo(s.T(), awsRegion, repositoryName)
	assert.Equal(s.T(), repositoryName, *repository.RepositoryName)
	assert.Equal(s.T(), arn, *repository.RepositoryArn)
	assert.EqualValues(s.T(), "IMMUTABLE", repository.ImageTagMutability)
	assert.True(s.T(), repository.ImageScanningConfiguration.ScanOnPush)
	assert.EqualValues(s.T(), "AES256", repository.EncryptionConfiguration.EncryptionType)

	lifecyclePolicyString := aws.GetECRRepoLifecyclePolicy(s.T(), awsRegion, repository)
	lifecyclePolicy := LifecyclePolicy{}
	json.Unmarshal([]byte(lifecyclePolicyString), &lifecyclePolicy)
	/**
        lifecycle_rules:
          - priority: 1
            description: "only keep 10 images"
            selection:
              tag_status: tagged
              tag_prefix_list: ["prod"]
              count_type: imageCountMoreThan
              count_number: 10
            action:
              type: expire
          - priority: 10
            description: "dev 10 rule"
            selection:
              tag_status: tagged
              tag_prefix_list: ["dev"]
              count_type: imageCountMoreThan
              count_number: 10
            action:
              type: expire
          - priority: 15
            description: "dev 10 rule"
            selection:
              tag_status: tagged
              tag_pattern_list: ["dev-*"]
              count_type: imageCountMoreThan
              count_number: 10
            action:
              type: expire
          - priority: 20
            description: "default rule"
            selection:
              tag_status: untagged
              count_type: imageCountMoreThan
              count_number: 10
            action:
              type: expire
          - priority: 40
            description: "Any tag rule"
            selection:
              tag_status: any
              count_type: sinceImagePushed
              count_unit: days
              count_number: 10
            action:
              type: expire
	*/
	expectedLifecyclePolicy := LifecyclePolicy{
		Rules: []LifecyclePolicyRule{
			{
				RulePriority: 1,
				Description:  "only keep 10 images",
				Selection: LifecyclePolicyRuleSelection{
					TagStatus:     "tagged",
					TagPrefixList: []string{"prod"},
					CountType:     "imageCountMoreThan",
					CountNumber:   10,
				},
				Action: map[string]string{
					"type": "expire",
				},
			},
			{
				RulePriority: 10,
				Description:  "dev 10 rule",
				Selection: LifecyclePolicyRuleSelection{
					TagStatus:   "tagged",
					TagPrefixList: []string{"dev"},
					CountType:   "imageCountMoreThan",
					CountNumber: 10,
				},
				Action: map[string]string{
					"type": "expire",
				},
			},
			{
				RulePriority: 15,
				Description:  "dev 10 rule",
				Selection: LifecyclePolicyRuleSelection{
					TagStatus:   "tagged",
					TagPatternList: []string{"dev-*"},
					CountType:   "imageCountMoreThan",
					CountNumber: 10,
				},
				Action: map[string]string{
					"type": "expire",
				},
			},
			{
				RulePriority: 20,
				Description:  "default rule",
				Selection: LifecyclePolicyRuleSelection{
					TagStatus:   "untagged",
					CountType:   "imageCountMoreThan",
					CountNumber: 10,
				},
				Action: map[string]string{
					"type": "expire",
				},
			},
			{
				RulePriority: 40,
				Description:  "Any tag rule",
				Selection: LifecyclePolicyRuleSelection{
					TagStatus:   "any",
					CountType:   "sinceImagePushed",
					CountUnit:   "days",
					CountNumber: 10,
				},
				Action: map[string]string{
					"type": "expire",
				},
			},
		},
	}
	assert.EqualValues(s.T(), expectedLifecyclePolicy, lifecyclePolicy)

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
