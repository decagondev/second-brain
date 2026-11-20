package gcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"example.com/second-brain/internal/core/backup"
	"example.com/second-brain/internal/core/schedule"
)

// Scheduler creates the cell's Cloud Scheduler jobs from schedule.Jobs, each publishing its
// name to the cell's topic (sb-runtime(7), Schedules, exactly; SB-904).
type Scheduler struct {
	Project  Project
	Location string
	Topic    string // projects/P/topics/TENANT-jobs
	Base     string // Cloud Scheduler API; "" is Google's
}

// SchedulerPermissions are what Scheduler.Ensure uses.
var SchedulerPermissions = []string{"cloudscheduler.jobs.create", "cloudscheduler.jobs.get", "cloudscheduler.jobs.update"}

func (s Scheduler) base() string {
	if s.Base == "" {
		return "https://cloudscheduler.googleapis.com"
	}
	return s.Base
}

// Ensure makes every job exist with its schedule, and corrects one that drifted.
func (s Scheduler) Ensure(ctx context.Context) error {
	parent := "projects/" + s.Project.ID + "/locations/" + s.Location
	for _, j := range schedule.Jobs {
		name := parent + "/jobs/sb-" + j.Name
		body := map[string]any{"name": name, "schedule": j.Cron, "timeZone": "Etc/UTC",
			"pubsubTarget": map[string]any{"topicName": s.Topic, "data": base64.StdEncoding.EncodeToString([]byte(j.Name))}}
		var got struct {
			Schedule string `json:"schedule"`
			TimeZone string `json:"timeZone"`
		}
		err := s.Project.do(ctx, http.MethodGet, s.base()+"/v1/"+name, nil, &got)
		switch {
		case absent(err):
			if err := s.Project.do(ctx, http.MethodPost, s.base()+"/v1/"+parent+"/jobs", body, nil); err != nil {
				return err
			}
		case err != nil:
			return err // unknown is not absent
		case got.Schedule != j.Cron || got.TimeZone != "Etc/UTC":
			if err := s.Project.do(ctx, http.MethodPatch, s.base()+"/v1/"+name, body, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

// Bucket is the cell's WORM bucket: a Cloud Storage bucket with a locked retention policy
// (ADR-0012). It is a backup.Target for the off-site copy and the hourly anchors.
type Bucket struct {
	Project Project
	Bucket  string
	Base    string // Cloud Storage JSON API; "" is Google's
}

var _ backup.Target = Bucket{}

// BucketPermissions are what Bucket uses; the bucket itself is made at bootstrap.
var BucketPermissions = []string{"storage.buckets.get", "storage.objects.create", "storage.objects.get", "storage.objects.list"}

// MinRetention is the shortest locked retention the bucket may have.
const MinRetention = 7 * 365 * 24 * time.Hour

func (b Bucket) base() string {
	if b.Base == "" {
		return "https://storage.googleapis.com"
	}
	return b.Base
}

// Name implements backup.Target.
func (b Bucket) Name() string { return "gcs:" + b.Bucket }

// Medium implements backup.Target.
func (b Bucket) Medium() string { return "object-store" }

// Offsite implements backup.Target: the bucket is outside the cell's database.
func (b Bucket) Offsite() bool { return true }

// Check refuses a bucket whose retention policy isn't locked, or is shorter than 7 years.
func (b Bucket) Check(ctx context.Context) error {
	var got struct {
		RetentionPolicy *struct {
			RetentionPeriod string `json:"retentionPeriod"`
			IsLocked        bool   `json:"isLocked"`
		} `json:"retentionPolicy"`
	}
	if err := b.Project.do(ctx, http.MethodGet, b.base()+"/storage/v1/b/"+url.PathEscape(b.Bucket), nil, &got); err != nil {
		return err
	}
	if got.RetentionPolicy == nil || !got.RetentionPolicy.IsLocked {
		return fmt.Errorf("gcp: bucket %s has no locked retention policy (ADR-0012)", b.Bucket)
	}
	var secs int64
	if _, err := fmt.Sscan(got.RetentionPolicy.RetentionPeriod, &secs); err != nil || time.Duration(secs)*time.Second < MinRetention {
		return fmt.Errorf("gcp: bucket %s keeps objects for less than 7 years", b.Bucket)
	}
	return nil
}

// Put writes an object only if none of that name exists (ifGenerationMatch=0), so nothing is
// ever overwritten; the bucket's locked policy keeps it until its retention ends.
func (b Bucket) Put(ctx context.Context, name string, data []byte, _ time.Time) error {
	if err := b.Check(ctx); err != nil {
		return err
	}
	tok, err := b.Project.Token(ctx)
	if err != nil {
		return err
	}
	u := b.base() + "/upload/storage/v1/b/" + url.PathEscape(b.Bucket) + "/o?uploadType=media&ifGenerationMatch=0&name=" + url.QueryEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := b.Project.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusPreconditionFailed:
		return fmt.Errorf("gcp: %s already exists in %s; WORM objects are never replaced", name, b.Bucket)
	}
	return &apiError{resp.StatusCode, "/upload/" + b.Bucket}
}

// Get implements backup.Target.
func (b Bucket) Get(ctx context.Context, name string) ([]byte, error) {
	tok, err := b.Project.Token(ctx)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.base()+"/storage/v1/b/"+url.PathEscape(b.Bucket)+"/o/"+url.PathEscape(name)+"?alt=media", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := b.Project.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, &apiError{resp.StatusCode, b.Bucket + "/" + name}
	}
	return io.ReadAll(io.LimitReader(resp.Body, 256<<20))
}

// List implements backup.Target.
func (b Bucket) List(ctx context.Context, prefix string) ([]string, error) {
	var out []string
	token := ""
	for {
		u := b.base() + "/storage/v1/b/" + url.PathEscape(b.Bucket) + "/o?prefix=" + url.QueryEscape(prefix)
		if token != "" {
			u += "&pageToken=" + url.QueryEscape(token)
		}
		var page struct {
			Items []struct {
				Name string `json:"name"`
			} `json:"items"`
			NextPageToken string `json:"nextPageToken"`
		}
		if err := b.Project.do(ctx, http.MethodGet, u, nil, &page); err != nil {
			return nil, err
		}
		for _, it := range page.Items {
			out = append(out, it.Name)
		}
		if page.NextPageToken == "" {
			return out, nil
		}
		token = page.NextPageToken
	}
}
