package uploader

import (
	"context"
	"errors"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"live-webrtc-go/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	client *minio.Client
	cfg    *config.Config
)

func Init(c *config.Config) error {
	cfg = c
	if !c.UploadEnabled {
		return nil
	}
	if c.S3Endpoint == "" || c.S3Bucket == "" || c.S3AccessKey == "" || c.S3SecretKey == "" {
		return errors.New("uploader: missing S3 configuration")
	}
	cl, err := minio.New(c.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(c.S3AccessKey, c.S3SecretKey, ""),
		Secure: c.S3UseSSL,
		Region: c.S3Region,
		BucketLookup: func() minio.BucketLookupType {
			if c.S3PathStyle {
				return minio.BucketLookupPath
			}
			return minio.BucketLookupDNS
		}(),
	})
	if err != nil {
		return err
	}
	client = cl
	return nil
}

func Enabled() bool { return cfg != nil && cfg.UploadEnabled && client != nil }

func Upload(ctx context.Context, localPath string) error {
	if !Enabled() {
		return nil
	}
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	name := filepath.Base(localPath)
	objectName := name
	if p := strings.Trim(cfg.S3Prefix, "/"); p != "" {
		objectName = p + "/" + name
	}
	contentType := mime.TypeByExtension(filepath.Ext(name))
	_, err = client.PutObject(ctx, cfg.S3Bucket, objectName, f, info.Size(), minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return err
	}
	if cfg.DeleteAfterUpload {
		_ = os.Remove(localPath)
	}
	return nil
}
