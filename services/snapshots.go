package services

import (
	"context"
	"sort"
	"time"

	"aws-cleaner/logger"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func printSnapShotsList(snapshotList []types.Snapshot, message string) {
	for _, snap := range snapshotList {
		ts := "-"
		if snap.StartTime != nil {
			ts = snap.StartTime.Format("2006-01-02 15:04:05")
		}
		logger.Infof("%s SnapshotID=%s, StartTime=%s, VolumeId=%s",
			message,
			aws.ToString(snap.SnapshotId),
			ts,
			aws.ToString(snap.VolumeId),
		)
	}
}

func sortByCreatedTime(snapshotList []types.Snapshot, order string) []types.Snapshot {
	sort.Slice(snapshotList, func(i, j int) bool {
		ti := snapshotList[i].StartTime
		tj := snapshotList[j].StartTime
		if order == "desc" {
			if ti == nil {
				return false
			}
			if tj == nil {
				return true
			}
			return ti.After(*tj)
		}
		// default asc
		if ti == nil {
			return false
		}
		if tj == nil {
			return true
		}
		return ti.Before(*tj)
	})
	return snapshotList
}

func getToDeleteSnapshots(snapshotList []types.Snapshot, deleteCount *int, keepCount *int) []types.Snapshot {
	n := len(snapshotList)

	if n == 0 {
		return []types.Snapshot{}
	}

	// If both deleteCount and keepCount are set -> log error and return nothing
	if deleteCount != nil && keepCount != nil {
		logger.Errorf("Both deleteCount (%d) and keepCount (%d) are set. Only one parameter should be used.",
			*deleteCount, *keepCount)
		return []types.Snapshot{}
	}

	// Handle keepCount first
	if keepCount != nil {
		if *keepCount < 0 {
			logger.Errorf("Keep count must be greater than 0")
			return []types.Snapshot{}
		}
		if *keepCount == 0 {
			return snapshotList
		}
		if *keepCount >= n {
			return []types.Snapshot{} // keep all, nothing to delete
		}
		return snapshotList[*keepCount:] // delete everything after the first N
	}

	// If deleteCount is set -> delete the first N items
	if deleteCount != nil {
		if *deleteCount <= 0 {
			return []types.Snapshot{} // nothing to delete
		}

		if *deleteCount >= n {
			return snapshotList // delete all
		}
		return snapshotList[:*deleteCount] // delete the first N
	}

	// If neither keepCount nor deleteCount is set -> nothing to delete
	return []types.Snapshot{}
}

func deleteSnapShot(snap types.Snapshot, client *ec2.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := client.DeleteSnapshot(ctx, &ec2.DeleteSnapshotInput{
		SnapshotId: snap.SnapshotId,
	})

	if err != nil {
		logger.Errorf("Failed to delete snapshot %v: %v", snap.SnapshotId, err)
		return err
	}
	ts := "-"
	if snap.StartTime != nil {
		ts = snap.StartTime.Format("2006-01-02 15:04:05")
	}
	logger.Infof("Deleted snapshot %s (%s)", *snap.SnapshotId, ts)
	return nil
}

func CleanupSnapshots(client *ec2.Client, tagKey, tagValue string, deleteCount *int, keepCount *int, sortBy string) {
	input := &ec2.DescribeSnapshotsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:" + tagKey),
				Values: []string{tagValue},
			},
		},
	}

	out, err := client.DescribeSnapshots(context.TODO(), input)

	if err != nil {
		logger.Errorf("Error in cleanupSnapshots: ", err)
	}
	removeSnapShot := out.Snapshots

	if len(out.Snapshots) == 0 {
		logger.Info("No snapshots found for given tag filter")
		return
	}

	// Sort by
	switch sortBy {
	case "created_time_asc":
		removeSnapShot = sortByCreatedTime(removeSnapShot, "asc")
	case "created_time_desc":
		removeSnapShot = sortByCreatedTime(removeSnapShot, "desc")
		printSnapShotsList(removeSnapShot, "Sort by created time desc: ")
	default:
		logger.Error("Not support that sortBy!")
		return
	}

	// Delete count
	removeSnapShot = getToDeleteSnapshots(removeSnapShot, deleteCount, keepCount)
	printSnapShotsList(removeSnapShot, "Delete Snapshot: ")

	for _, snap := range removeSnapShot {
		deleteSnapShot(snap, client)
	}
}
