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
