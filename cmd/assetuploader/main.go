package main

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws/credentials"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/labstack/echo"
	"github.com/tgracchus/assetuploader/pkg/assets"
	"github.com/tgracchus/assetuploader/pkg/endpoints"
)

func main() {
	pflag.String("region", "us-west-2", "aws region")
	pflag.String("bucket", "dmc-asset-uploader-test", "aws bucket")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.BindPFlags(pflag.CommandLine)
	pflag.Parse()
	region := viper.GetString("region")
	bucket := viper.GetString("bucket")
	// Setted as env variables only, so we dont see credentials in cmd history
	awsKey := viper.GetString("AWS_ACCESS_KEY_ID")
	if awsKey == "" {
		panic("AWS_ACCESS_KEY_ID should be present in env vars")
	}
	awsSecret := viper.GetString("AWS_SECRET_ACCESS_KEY")
	if awsSecret == "" {
		panic("AWS_SECRET_ACCESS_KEY should be present in env vars")
	}

	e := echo.New()
	e.HTTPErrorHandler = endpoints.AssetUploaderHTTPErrorHandler
	credentials := credentials.NewStaticCredentials(awsKey, awsSecret, "")
	session, err := assets.NewAwsSession(credentials, region)
	if err != nil {
		panic(err)
	}
	svc := assets.NewS3Client(session, region)
	manager := assets.NewDefaultFileManager(svc)
	endpoints.RegisterAssetsEndpoints(e, manager, bucket)
	endpoints.RegisterHealthCheck(e, svc, bucket)
	e.Logger.Fatal(e.Start(":8080"))
}
