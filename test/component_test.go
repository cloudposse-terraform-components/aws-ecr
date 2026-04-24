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

// LifecyclePolicyRuleSelection mirrors the selection block of an ECR
// lifecycle policy rule. All fields are tagged omitempty so that a single
// struct can represent every combination the module emits.
type LifecyclePolicyRuleSelection struct {
	TagStatus      string   `json:"tagStatus,omitempty"`
	TagPrefixList  []string `json:"tagPrefixList,omitempty"`
	TagPatternList []string `json:"tagPatternList,omitempty"`
	StorageClass   string   `json:"storageClass,omitempty"`
	CountType      string   `json:"countType,omitempty"`
	CountNumber    int      `json:"countNumber,omitempty"`
	CountUnit      string   `json:"countUnit,omitempty"`
}

// LifecyclePolicyRuleAction mirrors the action block. Using omitempty here
// is essential for the regression test against issue #158 in
// cloudposse/terraform-aws-ecr: a missing targetStorageClass must not be
// serialized as "targetStorageClass":null, and Unmarshal followed by Marshal
// should round-trip to a rule without the field.
type LifecyclePolicyRuleAction struct {
	Type               string `json:"type,omitempty"`
	TargetStorageClass string `json:"targetStorageClass,omitempty"`
}

type LifecyclePolicyRule struct {
	RulePriority int                          `json:"rulePriority"`
	Description  string                       `json:"description"`
	Selection    LifecyclePolicyRuleSelection `json:"selection"`
	Action       LifecyclePolicyRuleAction    `json:"action"`
}

type LifecyclePolicy struct {
	Rules []LifecyclePolicyRule `json:"rules"`
}

// findRuleByDescription returns a pointer to the rule in policy matching
// description, or nil if none is found.
func findRuleByDescription(policy LifecyclePolicy, description string) *LifecyclePolicyRule {
	for i := range policy.Rules {
		if policy.Rules[i].Description == description {
			return &policy.Rules[i]
		}
	}
	return nil
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
		"images": []string{
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
				Action: LifecyclePolicyRuleAction{Type: "expire"},
			},
			{
				RulePriority: 2,
				Description:  "Remove untagged images",
				Selection: LifecyclePolicyRuleSelection{
					TagStatus:   "untagged",
					CountType:   "imageCountMoreThan",
					CountNumber: 1,
				},
				Action: LifecyclePolicyRuleAction{Type: "expire"},
			},
			{
				RulePriority: 3,
				Description:  "Rotate images when reach 500 images stored",
				Selection: LifecyclePolicyRuleSelection{
					TagStatus:   "any",
					CountType:   "imageCountMoreThan",
					CountNumber: 500,
				},
				Action: LifecyclePolicyRuleAction{Type: "expire"},
			},
		},
	}

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
		err := json.Unmarshal([]byte(lifecyclePolicyString), &lifecyclePolicy)
		assert.NoError(s.T(), err)
		assert.EqualValues(s.T(), expectedLifecyclePolicy, lifecyclePolicy)
	}

	s.DriftTest(component, stack, &inputs)
}

func (s *ComponentSuite) TestImmutabilityExclusions() {
	const component = "ecr/immutability-exclusions"
	const stack = "default-test"
	const awsRegion = "us-east-2"

	suffix := strings.ToLower(random.UniqueId())

	inputs := map[string]interface{}{
		"images": []string{
			fmt.Sprintf("infrastructure-%s", suffix),
			fmt.Sprintf("microservice-a-%s", suffix),
			fmt.Sprintf("microservice-b-%s", suffix),
			fmt.Sprintf("microservice-c-%s", suffix),
		},
	}

	defer s.DestroyAtmosComponent(s.T(), component, stack, &inputs)
	options, _ := s.DeployAtmosComponent(s.T(), component, stack, &inputs)
	assert.NotNil(s.T(), options)

	arnMaps := map[string]string{}
	atmos.OutputStruct(s.T(), options, "ecr_repo_arn_map", &arnMaps)

	for name := range arnMaps {
		repository := aws.GetECRRepo(s.T(), awsRegion, name)
		assert.EqualValues(s.T(), "IMMUTABLE_WITH_EXCLUSION", repository.ImageTagMutability)
		assert.True(s.T(), repository.ImageScanningConfiguration.ScanOnPush)
	}

	s.DriftTest(component, stack, &inputs)
}

// TestCustomLifecycleRules deploys the component with a rich set of
// custom_lifecycle_rules that cover both the regression from upstream issue
// #158 (https://github.com/cloudposse/terraform-aws-ecr/issues/158) and the
// v1.0.1 archive/transition feature. A successful deploy is itself the
// primary oracle — v1.0.1's bug caused ECR's PutLifecyclePolicy to 400 —
// and the JSON-level assertions below pin the specific regression so it
// can't silently come back.
func (s *ComponentSuite) TestCustomLifecycleRules() {
	const component = "ecr/custom-lifecycle-rules"
	const stack = "default-test"
	const awsRegion = "us-east-2"

	suffix := strings.ToLower(random.UniqueId())

	inputs := map[string]interface{}{
		"images": []string{
			fmt.Sprintf("clifecycle-a-%s", suffix),
			fmt.Sprintf("clifecycle-b-%s", suffix),
		},
	}

	defer s.DestroyAtmosComponent(s.T(), component, stack, &inputs)
	options, _ := s.DeployAtmosComponent(s.T(), component, stack, &inputs)
	assert.NotNil(s.T(), options)

	arnMaps := map[string]string{}
	atmos.OutputStruct(s.T(), options, "ecr_repo_arn_map", &arnMaps)
	assert.NotEmpty(s.T(), arnMaps)

	for name := range arnMaps {
		repository := aws.GetECRRepo(s.T(), awsRegion, name)
		lifecyclePolicyString := aws.GetECRRepoLifecyclePolicy(s.T(), awsRegion, repository)

		// Raw-JSON regression checks for issue #158. Before v1.0.2, the
		// module serialized `"targetStorageClass":null` on every expire
		// action and `"storageClass":"standard"` on every selection that
		// didn't set it — both caused ECR to reject the policy at apply.
		assert.NotContains(s.T(), lifecyclePolicyString, `"targetStorageClass":null`,
			"regression: cloudposse/terraform-aws-ecr#158 — targetStorageClass=null leak on %s", name)
		assert.NotContains(s.T(), lifecyclePolicyString, `"storageClass":"standard"`,
			"regression: cloudposse/terraform-aws-ecr#158 — spurious storageClass=standard injected on %s", name)

		policy := LifecyclePolicy{}
		err := json.Unmarshal([]byte(lifecyclePolicyString), &policy)
		assert.NoError(s.T(), err, "lifecycle policy must be valid JSON on %s", name)

		// Issue #158 reproducer: tagged + tagPatternList + expire with no
		// storageClass. Must round-trip cleanly with no extra fields.
		latestRule := findRuleByDescription(policy, "Keep only last 10 images tagged 'latest'")
		if assert.NotNil(s.T(), latestRule, "missing 'latest' rule on %s", name) {
			assert.Equal(s.T(), "tagged", latestRule.Selection.TagStatus)
			assert.Equal(s.T(), []string{"latest"}, latestRule.Selection.TagPatternList)
			assert.Equal(s.T(), "imageCountMoreThan", latestRule.Selection.CountType)
			assert.Equal(s.T(), 10, latestRule.Selection.CountNumber)
			assert.Empty(s.T(), latestRule.Selection.StorageClass, "storageClass must be omitted when not set")
			assert.Equal(s.T(), "expire", latestRule.Action.Type)
			assert.Empty(s.T(), latestRule.Action.TargetStorageClass, "targetStorageClass must be omitted for expire actions")
		}

		// v1.0.1 feature: transition to archive. Selection has no
		// storageClass, action carries targetStorageClass=archive.
		archiveRule := findRuleByDescription(policy, "Archive tagged images older than 30 days")
		if assert.NotNil(s.T(), archiveRule, "missing archive transition rule on %s", name) {
			assert.Equal(s.T(), "tagged", archiveRule.Selection.TagStatus)
			assert.Equal(s.T(), []string{"v"}, archiveRule.Selection.TagPrefixList)
			assert.Equal(s.T(), "sinceImagePushed", archiveRule.Selection.CountType)
			assert.Equal(s.T(), "days", archiveRule.Selection.CountUnit)
			assert.Empty(s.T(), archiveRule.Selection.StorageClass)
			assert.Equal(s.T(), "transition", archiveRule.Action.Type)
			assert.Equal(s.T(), "archive", archiveRule.Action.TargetStorageClass)
		}

		// v1.0.1 + v1.0.2 feature: expire from archive. Selection has
		// storageClass=archive; ECR requires countType=sinceImageTransitioned.
		expireArchiveRule := findRuleByDescription(policy, "Expire images that have been archived for more than 90 days")
		if assert.NotNil(s.T(), expireArchiveRule, "missing archive-expire rule on %s", name) {
			assert.Equal(s.T(), "tagged", expireArchiveRule.Selection.TagStatus)
			assert.Equal(s.T(), []string{"v"}, expireArchiveRule.Selection.TagPrefixList)
			assert.Equal(s.T(), "archive", expireArchiveRule.Selection.StorageClass)
			assert.Equal(s.T(), "sinceImageTransitioned", expireArchiveRule.Selection.CountType)
			assert.Equal(s.T(), "days", expireArchiveRule.Selection.CountUnit)
			assert.Equal(s.T(), 90, expireArchiveRule.Selection.CountNumber)
			assert.Equal(s.T(), "expire", expireArchiveRule.Action.Type)
			assert.Empty(s.T(), expireArchiveRule.Action.TargetStorageClass)
		}
	}

	s.DriftTest(component, stack, &inputs)
}

func (s *ComponentSuite) TestEnabledFlag() {
	const component = "ecr/disabled"
	const stack = "default-test"

	s.VerifyEnabledFlag(component, stack, nil)
}

func TestRunSuite(t *testing.T) {
	suite := new(ComponentSuite)
	helper.Run(t, suite)
}
