package audit

import "testing"

func TestBuildWriteLogDraft_nonWriteMethod(t *testing.T) {
	_, ok := BuildWriteLogDraft("get", "users", "users", nil, false)
	if ok {
		t.Fatal("expected skip for GET")
	}
}

func TestBuildWriteLogDraft_userPostWithName(t *testing.T) {
	body := []byte(`{"name":"alice","apiVersion":"v1"}`)
	d, ok := BuildWriteLogDraft("post", "users", "users", body, false)
	if !ok {
		t.Fatal("expected ok")
	}
	if d.Operation != "post" || d.OperationDomain != "users" || d.SpecificInformation != "alice" {
		t.Fatalf("unexpected draft: %+v", d)
	}
}

func TestBuildWriteLogDraft_ldapImport(t *testing.T) {
	d, ok := BuildWriteLogDraft("post", "ldap/foo/import", "ldap/:name/import", []byte(`{}`), true)
	if !ok {
		t.Fatal("expected ok")
	}
	if d.Operation != "import" {
		t.Fatalf("expected import, got %q", d.Operation)
	}
}

func TestBuildWriteLogDraft_clusterScopedPut(t *testing.T) {
	d, ok := BuildWriteLogDraft("put", "clusters/c1/pods/p1", "clusters/:name/pods/:podName", nil, false)
	if !ok {
		t.Fatal("expected ok")
	}
	if d.OperationDomain != "clusters_pods" {
		t.Fatalf("domain: %+v", d)
	}
	if d.SpecificInformation != "[c1] p1" {
		t.Fatalf("info: %+v", d)
	}
}

func TestBuildWriteLogDraft_patchDefaultToPut(t *testing.T) {
	d, ok := BuildWriteLogDraft("patch", "clusters/c1/configmaps/cfg1", "clusters/:name/configmaps/:configMapName", []byte(`{"data":{"k":"v"}}`), false)
	if !ok {
		t.Fatal("expected ok")
	}
	if d.Operation != "put" {
		t.Fatalf("expected put for generic patch, got %q", d.Operation)
	}
}

func TestBuildWriteLogDraft_clusterNamespaceCreateResourceDomain(t *testing.T) {
	body := []byte(`{"metadata":{"name":"d1"}}`)
	d, ok := BuildWriteLogDraft("post", "clusters/c1/namespaces/ns1/deployments", "clusters/:name/namespaces/:namespace/deployments", body, false)
	if !ok {
		t.Fatal("expected ok")
	}
	if d.OperationDomain != "clusters_deployments" {
		t.Fatalf("expected domain clusters_deployments, got %q", d.OperationDomain)
	}
}

func TestBuildWriteLogDraft_workloadPatchSemantic(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
		want string
	}{
		{
			name: "restart deployment",
			path: "clusters/c1/namespaces/ns1/deployments/d1",
			body: `{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"2026-04-08T10:00:00Z"}}}}}`,
			want: "restart",
		},
		{
			name: "scale deployment",
			path: "clusters/c1/namespaces/ns1/deployments/d1/scale",
			body: `{"spec":{"replicas":3}}`,
			want: "scale",
		},
		{
			name: "pause deployment",
			path: "clusters/c1/namespaces/ns1/deployments/d1",
			body: `{"spec":{"paused":true}}`,
			want: "pause",
		},
		{
			name: "resume deployment",
			path: "clusters/c1/namespaces/ns1/deployments/d1",
			body: `{"spec":{"paused":false}}`,
			want: "resume",
		},
		{
			name: "rollback deployment",
			path: "clusters/c1/namespaces/ns1/deployments/d1/rollback",
			body: `{}`,
			want: "rollback",
		},
		{
			name: "reschedule deployment",
			path: "clusters/c1/namespaces/ns1/deployments/d1/reschedule",
			body: `{}`,
			want: "reschedule",
		},
	}
	for _, tt := range tests {
		d, ok := BuildWriteLogDraft("patch", tt.path, "clusters/:name/namespaces/:namespace/deployments/:name", []byte(tt.body), false)
		if !ok {
			t.Fatalf("%s: expected ok", tt.name)
		}
		if d.Operation != tt.want {
			t.Fatalf("%s: want %q, got %q", tt.name, tt.want, d.Operation)
		}
	}
}

func TestInferWriteOperation(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   string
		want   string
	}{
		{name: "put keep put", method: "put", path: "clusters/c1/users/u1", body: `{}`, want: "put"},
		{name: "post keep post", method: "post", path: "clusters/c1/users", body: `{}`, want: "post"},
		{name: "delete keep delete", method: "delete", path: "clusters/c1/users/u1", body: `{}`, want: "delete"},
		{name: "patch generic to put", method: "patch", path: "clusters/c1/configmaps/c1", body: `{"data":{"k":"v"}}`, want: "put"},
		{name: "patch restart semantic", method: "patch", path: "clusters/c1/namespaces/ns1/deployments/d1", body: `{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"x"}}}}}`, want: "restart"},
	}
	for _, tt := range tests {
		got := InferWriteOperation(tt.method, tt.path, []byte(tt.body))
		if got != tt.want {
			t.Fatalf("%s: want %q got %q", tt.name, tt.want, got)
		}
	}
}
