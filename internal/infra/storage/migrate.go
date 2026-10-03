package storage

import (
	"fmt"
	"log"
)

func RunMigration(storage Provider) error {
	log.Printf(`storage: running migration`)

	buckets := []Bucket{
		{
			Name:   "public-assets",
			Public: true,
		},
		{
			Name:   "private-assets",
			Public: false,
		},
	}

	for _, b := range buckets {
		exist, err := storage.EnsureBucket(b.Name)
		if err != nil {
			return fmt.Errorf("storage: %s failed: %w", b.Name, err)
		}
		if exist {
			log.Printf("storage: %s exists", b.Name)
			continue
		}

		if err := storage.CreateBucket(b.Name, b.Public); err != nil {
			return fmt.Errorf("storage: %s failed: %w", b.Name, err)
		}
		log.Printf("storage: %s created", b.Name)
	}

	return nil
}
