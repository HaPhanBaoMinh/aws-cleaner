package main

import (
	"context"
	"os"
	"strconv"

	"go.uber.org/zap"

	"aws-cleaner/logger"
	"aws-cleaner/services"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/joho/godotenv"
)

func init() {
	// Set the log level to DEBUG (or any other level you prefer)
	os.Setenv("LOG_LEVEL", "DEBUG")
	logger.InitLogger()

	if err := godotenv.Load(".env"); err != nil {
		logger.Warnf("No .env file found, fallback to system env")
	}
}

func main() {
	// Load env
	resourceType := os.Getenv("RESOURCE_TYPE")
	tagKey := os.Getenv("TAG_KEY")
	tagValue := os.Getenv("TAG_VALUE")
	deleteCountStr := os.Getenv("DELETE_COUNT")
	keepCountStr := os.Getenv("KEEP_COUNT")
	awsRegion := os.Getenv("AWS_REGION")
	sortBy := os.Getenv("SORT_BY") // time|id

	logger.Info("Loaded config",
		zap.String("RESOURCE_TYPE", resourceType),
		zap.String("TAG_KEY", tagKey),
		zap.String("TAG_VALUE", tagValue),
		zap.String("DELETE_COUNT", deleteCountStr),
		zap.String("KEEP_COUNT", keepCountStr),
		zap.String("AWS_REGION", awsRegion),
		zap.String("SORT_BY", sortBy),
	)

	// Validate variables
	invalidVar := false
	if awsRegion == "" {
		logger.Errorf("AWS_REGION is empty!")
		invalidVar = true
	}

	if tagKey == "" {
		logger.Errorf("TAG_KEY is empty!")
		invalidVar = true
	}

	if tagValue == "" {
		logger.Errorf("TAG_VALUE is empty!")
		invalidVar = true
	}

	if resourceType == "" {
		logger.Errorf("RESOURCE_TYPE is empty!")
		invalidVar = true
	}

	var deleteCount *int
	if deleteCountStr != "" {
		val, err := strconv.Atoi(deleteCountStr)

		if err != nil {
			logger.Errorf("Invalid DELETE_COUNT!")
			invalidVar = true
		} else {
			deleteCount = &val
		}
	} else {
		deleteCount = nil
	}

	var keepCount *int
	if keepCountStr != "" {
		val, err := strconv.Atoi(keepCountStr)

		if err != nil {
			logger.Errorf("Invalid KEEP_COUNT")
			invalidVar = true
		} else {
			keepCount = &val
		}
	} else {
		keepCount = nil
	}

	logger.Debug(deleteCountStr)

	if sortBy == "" {
		sortBy = "created_time"
	}

	if invalidVar {
		return
	}

	// Init AWS SDK
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(awsRegion))
	if err != nil {
		logger.Errorf("Unable to load SDK config: %v", err)
		return
	}
	ec2Client := ec2.NewFromConfig(cfg)

	switch resourceType {
	case "ebs-snapshot":
		logger.Debug("CleanupSnapshots")
		services.CleanupSnapshots(ec2Client, tagKey, tagValue, deleteCount, keepCount, sortBy)
	default:
		logger.Errorf("Unsupported resource type: %s", resourceType)
		return
	}
}
