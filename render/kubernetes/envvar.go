package kubernetes

import (
	k8s "github.com/YakLabs/k8s-client"
)

// createEnvVarFromField creates a v1.EnvVar with a ValueFrom set to a ObjectFieldSelector
func createEnvVarFromField(key, fieldPath string) k8s.EnvVar {
	return k8s.EnvVar{
		Name: key,
		ValueFrom: &k8s.EnvVarSource{
			FieldRef: &k8s.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  fieldPath,
			},
		},
	}
}

// createEnvVarFromSecret creates a v1.EnvVar with a ValueFrom set to a SecretKeySelector
func createEnvVarFromSecret(envKey, secretName, secretKey string) k8s.EnvVar {
	return k8s.EnvVar{
		Name: envKey,
		ValueFrom: &k8s.EnvVarSource{
			SecretKeyRef: &k8s.SecretKeySelector{
				LocalObjectReference: k8s.LocalObjectReference{
					Name: secretName,
				},
				Key: secretKey,
			},
		},
	}
}
