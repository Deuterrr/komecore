package gcs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/iam"
	gcsSDK "cloud.google.com/go/storage"
)

func (p *GCSProvider) EnsureBucket(name string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := p.Client.Bucket(name).Attrs(ctx)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gcsSDK.ErrBucketNotExist) {
		return false, nil
	}

	return false, fmt.Errorf("check gcs bucket %s failed: %w", name, err)
}

func (p *GCSProvider) CreateBucket(name string, public bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	attrs := &gcsSDK.BucketAttrs{
		UniformBucketLevelAccess: gcsSDK.UniformBucketLevelAccess{
			Enabled: true,
		},
	}

	projectID := p.GCSConfig.ProjectID
	if err := p.Client.Bucket(name).Create(ctx, projectID, attrs); err != nil {
		return fmt.Errorf("create gcs bucket %s failed: %w", name, err)
	}

	if public {
		bkt := p.Client.Bucket(name)
		policy, err := bkt.IAM().Policy(ctx)
		if err == nil {
			policy.Add(iam.AllUsers, iam.RoleName("roles/storage.objectViewer"))
			_ = bkt.IAM().SetPolicy(ctx, policy)
		}
	}

	return nil
}
