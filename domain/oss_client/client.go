package oss_client

import (
	"bytes"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type Oss struct {
	Bucket *oss.Bucket
}

func NewOssWithConfig(conf OssConfig) (*Oss, error) {

	client, err := oss.New(conf.EndPoint, conf.AccessKeyId, conf.AccessKeySecret)
	if err != nil {
		return nil, err
	}
	bucket, err := client.Bucket(conf.Bucket)
	if err != nil {
		return nil, err
	}

	newOss := new(Oss)

	newOss.Bucket = bucket

	return newOss, nil
}

type OssConfig struct {
	EndPoint        string `json:"endpoint"`
	AccessKeyId     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	Bucket          string `json:"bucket"`
}

func (o *Oss) PutObject(objectKey string, buf []byte) error {
	return o.Bucket.PutObject(objectKey, bytes.NewBuffer(buf))
}
