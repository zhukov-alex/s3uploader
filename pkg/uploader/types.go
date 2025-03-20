package uploader

type S3Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Url 	  string
}

type UploadRequest struct {
	Bucket   string
	FilePath string
	Key      string
}
