package storage

// NoopProvider provides a safe, no-operation fallback implementation of storage.Provider
// when external cloud storage (e.g. Supabase Storage) is not configured.
type NoopProvider struct{}

func NewNoopProvider() *NoopProvider {
	return &NoopProvider{}
}

func (n *NoopProvider) Upload(input UploadInput) (*ObjectResponse, error) {
	return &ObjectResponse{
		Key:         input.Key,
		ContentType: input.ContentType,
	}, nil
}

func (n *NoopProvider) UploadMany(inputs []UploadInput) ([]*ObjectResponse, error) {
	res := make([]*ObjectResponse, len(inputs))
	for i, input := range inputs {
		res[i] = &ObjectResponse{
			Key:         input.Key,
			ContentType: input.ContentType,
		}
	}
	return res, nil
}

func (n *NoopProvider) Delete(key string) error {
	return nil
}

func (n *NoopProvider) Exists(key string) (bool, error) {
	return true, nil
}

func (n *NoopProvider) PublicURL(key string, bucket string) string {
	return "/assets/" + bucket + "/" + key
}

func (n *NoopProvider) SignedURL(key string) (string, error) {
	return "/assets/private/" + key, nil
}

func (n *NoopProvider) EnsureBucket(name string) (bool, error) {
	return true, nil
}

func (n *NoopProvider) CreateBucket(name string, public bool) error {
	return nil
}
