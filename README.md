# AWS Resource Cleaner

A lightweight Go application to automatically clean up AWS resources based on tags.  
Currently supports **EBS Snapshots**.  
Designed to run as a **Kubernetes CronJob** or standalone CLI tool.

---

## Features
- Clean up EBS snapshots filtered by **Tag Key/Value**
- Sort snapshots by **creation time** (asc/desc)
- Delete snapshots based on a configured `DELETE_COUNT` or `KEEP_COUNT`
- Structured logging with rotation

---

## Environment Variables

| Variable       | Description                                    | Default        | Required |
|----------------|------------------------------------------------|----------------|----------|
| `RESOURCE_TYPE`| Resource type to clean (currently: `ebs-snapshot`) | -              | Yes       |
| `TAG_KEY`      | Tag key filter                                 | -              | Yes       |
| `TAG_VALUE`    | Tag value filter                               | -              | Yes       |
| `DELETE_COUNT` | Number of snapshots to delete (top -> bottom)           | -| No      |
| `KEEP_COUNT`   | Number of snapshots to keep (from the top of the sorted list)   | -               | No |
| `AWS_REGION`   | AWS region                                     | -| Yes      |
| `SORT_BY`      | Sort order (`created_time_asc` / `created_time_desc`) | - | No       |
| `LOG_LEVEL`    | Logging level (`DEBUG`, `INFO`, `WARN`, `ERROR`)| -           | No       |


---

## Deploy to Kubernetes

### 1. Create AWS Secret

Save your AWS credentials (use **IAM user** or **assume role** with minimum required permissions):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aws-credentials
  namespace: default
type: Opaque
stringData:
  AWS_ACCESS_KEY_ID: "<your-access-key-id>"
  AWS_SECRET_ACCESS_KEY: "<your-secret-access-key>"
  AWS_REGION: "ap-southeast-1"
```

### 2. Create CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: aws-cleaner
  namespace: default
spec:
  # Run every 10 minutes
  schedule: "*/10 * * * *"
  successfulJobsHistoryLimit: 1
  failedJobsHistoryLimit: 1
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
            - name: aws-cleaner
              image: baominh/aws-cleaner:latest
              imagePullPolicy: IfNotPresent
              env:
                - name: RESOURCE_TYPE
                  value: "ebs-snapshot"
                - name: TAG_KEY
                  value: "Environment"
                - name: TAG_VALUE
                  value: "dev"
                - name: DELETE_COUNT \ KEEP_COUNT
                  value: "2"
                - name: SORT_BY
                  value: "created_time_desc"
                - name: LOG_LEVEL
                  value: "INFO"
              envFrom:
                - secretRef:
                    name: aws-credentials
```
