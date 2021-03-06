package factory_test

import (
	"testing"

	"github.com/jenkins-x-labs/helmboot/pkg/fakes/fakejxfactory"
	"github.com/jenkins-x-labs/helmboot/pkg/secretmgr"
	"github.com/jenkins-x-labs/helmboot/pkg/secretmgr/factory"
	"github.com/jenkins-x-labs/helmboot/pkg/secretmgr/fake"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/jxfactory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	modifiedYaml = `secrets:
  adminUser:
    username: admin
    password: dummypwd 
  hmacToken:  TODO
  pipelineUser:
    username: someuser 
    token: dummmytoken 
    email: me@foo.com
`
)

func TestFakeSecretManager(t *testing.T) {
	f := fakejxfactory.NewFakeFactory()
	sm := AssertSecretsManager(t, secretmgr.KindFake, f)

	// lets assume its a fake one
	fakeSM, ok := sm.(*fake.FakeSecretManager)
	require.True(t, ok, "SecretManager should be Fake but was %#v", sm)
	assert.Equal(t, modifiedYaml, fakeSM.SecretsYAML, "FakeSecretManager should contain the correct YAML")
}

func TestLocalSecretManager(t *testing.T) {
	f := fakejxfactory.NewFakeFactory()
	AssertSecretsManager(t, secretmgr.KindLocal, f)

	// lets assume its a fake one
	kubeClient, ns, err := f.CreateKubeClient()
	require.NoError(t, err, "faked to create KubeClient")
	secret, err := kubeClient.CoreV1().Secrets(ns).Get(secretmgr.LocalSecret, metav1.GetOptions{})
	require.NoError(t, err, "failed to get Secret %s in namespace %s", secretmgr.LocalSecret, ns)

	secretYaml := string(secret.Data[secretmgr.LocalSecretKey])
	assert.Equal(t, modifiedYaml, secretYaml, "Secret %s in namespace %s has wrong Secrets YAML", secretmgr.LocalSecret, ns)
}

func AssertSecretsManager(t *testing.T, kind string, f jxfactory.Factory) secretmgr.SecretManager {
	requirments := config.NewRequirementsConfig()
	sm, err := factory.NewSecretManager(kind, f, requirments)
	require.NoError(t, err, "failed to create a SecretManager of kind %s", kind)
	require.NotNil(t, sm, "SecretManager of kind %s", kind)

	err = sm.UpsertSecrets(dummyCallback, secretmgr.DefaultSecretsYaml)
	require.NoError(t, err, "failed to modify secrets for SecretManager of kind %s", kind)

	actualYaml := ""
	testCb := func(secretsYaml string) (string, error) {
		actualYaml = secretsYaml
		return secretsYaml, nil
	}

	err = sm.UpsertSecrets(testCb, secretmgr.DefaultSecretsYaml)
	require.NoError(t, err, "failed to get the secrets from the SecretManager of kind %s", kind)

	assert.Equal(t, modifiedYaml, actualYaml, "should have got the YAML from the secret manager kind %s", kind)
	return sm
}

func dummyCallback(secretsYaml string) (string, error) {
	return modifiedYaml, nil
}
