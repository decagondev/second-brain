package gcp

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"example.com/second-brain/internal/core/runtime"
)

// ServiceRoles are each runtime service account's roles, all in the cell's own project
// (sb-runtime(7), Services, exactly). Nothing here names another project.
var ServiceRoles = map[string][]string{
	"api":     {"roles/cloudsql.client", "roles/secretmanager.secretAccessor", "roles/cloudkms.cryptoKeyEncrypterDecrypter", "roles/pubsub.publisher"},
	"ingest":  {"roles/cloudsql.client", "roles/secretmanager.secretAccessor", "roles/pubsub.subscriber"},
	"web":     {},
	"jobs":    {"roles/cloudsql.client", "roles/storage.objectCreator", "roles/pubsub.publisher"},
	"updater": {"roles/run.admin", "roles/containeranalysis.occurrences.editor"},
}

// UpdaterPermissions are what the updater's identity uses through CloudRun.
var UpdaterPermissions = []string{
	"containeranalysis.occurrences.create",
	"run.jobs.create", "run.jobs.get", "run.jobs.update",
	"run.services.create", "run.services.get", "run.services.update",
}

// ServiceAccount is a service's account in the cell's project.
func ServiceAccount(project, service string) string {
	return "sb-" + service + "@" + project + ".iam.gserviceaccount.com"
}

// CloudRun deploys a cell's services in its own project (runtime.Deployer). Its token is
// the updater job's; Binary Authorization refuses any image the cell hasn't attested.
type CloudRun struct {
	Project  Project
	Location string
	Run      string // Cloud Run Admin API base; "" is Google's
	Analysis string // Container Analysis API base; "" is Google's
}

var _ runtime.Deployer = CloudRun{}

func (c CloudRun) bases() (string, string) {
	run, ca := c.Run, c.Analysis
	if run == "" {
		run = "https://run.googleapis.com"
	}
	if ca == "" {
		ca = "https://containeranalysis.googleapis.com"
	}
	return run, ca
}

func digestOnly(image string) error {
	at := strings.LastIndex(image, "@sha256:")
	if at < 0 || len(image)-at-len("@sha256:") != 64 {
		return fmt.Errorf("gcp: %q isn't pinned by digest; tags are never deployed", image)
	}
	return nil
}

// Attest records the cell's attestation of image in its own project.
func (c CloudRun) Attest(ctx context.Context, image string) error {
	if err := digestOnly(image); err != nil {
		return err
	}
	_, ca := c.bases()
	body := map[string]any{"resourceUri": image, "noteName": "projects/" + c.Project.ID + "/notes/sb-admitted", "kind": "ATTESTATION"}
	return c.Project.do(ctx, http.MethodPost, ca+"/v1/projects/"+c.Project.ID+"/occurrences", body, nil)
}

func (c CloudRun) path(svc runtime.Service) string {
	kind := "services"
	if svc.Job {
		kind = "jobs"
	}
	run, _ := c.bases()
	return run + "/v2/projects/" + c.Project.ID + "/locations/" + c.Location + "/" + kind
}

type runResource struct {
	Name     string            `json:"name"`
	Labels   map[string]string `json:"labels"`
	Template struct {
		Containers []struct {
			Image string `json:"image"`
		} `json:"containers"`
		ServiceAccount string `json:"serviceAccount"`
	} `json:"template"`
}

// Running returns svc's image and release, or "" when it doesn't exist (only on 404).
func (c CloudRun) Running(ctx context.Context, svc runtime.Service) (string, string, error) {
	var r runResource
	err := c.Project.do(ctx, http.MethodGet, c.path(svc)+"/"+svc.Name, nil, &r)
	if absent(err) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	if len(r.Template.Containers) == 0 {
		return "", "", fmt.Errorf("gcp: %s has no container", svc.Name)
	}
	return r.Template.Containers[0].Image, r.Labels["sb-release"], nil
}

// Deploy creates or updates svc at image, as its own service account.
func (c CloudRun) Deploy(ctx context.Context, svc runtime.Service, image, releaseID string) error {
	if err := digestOnly(image); err != nil {
		return err
	}
	body := map[string]any{
		"labels": map[string]string{"sb-release": releaseID},
		"template": map[string]any{
			"containers":     []any{map[string]any{"image": image}},
			"serviceAccount": ServiceAccount(c.Project.ID, svc.Name),
		},
	}
	cur, _, err := c.Running(ctx, svc)
	if err != nil {
		return err
	}
	if cur == "" {
		body["name"] = svc.Name
		return c.Project.do(ctx, http.MethodPost, c.path(svc)+"?"+map[bool]string{true: "jobId", false: "serviceId"}[svc.Job]+"="+svc.Name, body, nil)
	}
	return c.Project.do(ctx, http.MethodPatch, c.path(svc)+"/"+svc.Name, body, nil)
}
