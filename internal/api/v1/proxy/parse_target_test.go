package proxy

import (
	"reflect"
	"testing"
)

func TestParseK8sTarget(t *testing.T) {
	cases := []struct {
		path string
		want k8sTarget
	}{
		{
			path: "api/v1/namespaces",
			want: k8sTarget{resource: "namespaces"},
		},
		{
			path: "/api/v1/namespaces",
			want: k8sTarget{resource: "namespaces"},
		},
		{
			path: "api/v1/namespaces/kube-system",
			want: k8sTarget{resource: "namespaces", name: "kube-system"},
		},
		{
			path: "api/v1/namespaces/kube-system/status",
			want: k8sTarget{resource: "namespaces", name: "kube-system", subresource: "status"},
		},
		{
			path: "api/v1/namespaces/default/deployments/nginx",
			want: k8sTarget{namespace: "default", resource: "deployments", name: "nginx"},
		},
		{
			path: "apis/apps/v1/namespaces/default/deployments/nginx/scale",
			want: k8sTarget{namespace: "default", resource: "deployments", name: "nginx", subresource: "scale"},
		},
		{
			path: "apis/rbac.authorization.k8s.io/v1/clusterroles/admin",
			want: k8sTarget{resource: "clusterroles", name: "admin"},
		},
		{
			path: "apis/rbac.authorization.k8s.io/v1/clusterrolebindings/some-binding",
			want: k8sTarget{resource: "clusterrolebindings", name: "some-binding"},
		},
		{
			path: "apis/rbac.authorization.k8s.io/v1/namespaces/default/roles/pod-reader",
			want: k8sTarget{namespace: "default", resource: "roles", name: "pod-reader"},
		},
	}
	for _, tc := range cases {
		got := parseK8sTarget(tc.path)
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("path %q: got %+v want %+v", tc.path, got, tc.want)
		}
	}
}
