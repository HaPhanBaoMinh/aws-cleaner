package services

import (
	"context"
	"os"
	"sort"

	"aws-cleaner/logger"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func printSnapShotsList(snapshotList []types.Snapshot, message string) {
	for _, snap := range snapshotList {
		logger.Infof("%s SnapshotID=%s, StartTime=%s, VolumeId=%s", message,
			aws.ToString(snap.SnapshotId),
			snap.StartTime.Format("2006-01-02 15:04:05"),
			aws.ToString(snap.VolumeId),
		)
	}
}

func sortByCreatedTime(snapshotList []types.Snapshot, order string) []types.Snapshot {
	sort.Slice(snapshotList, func(i, j int) bool {
		if order == "desc" {
			return snapshotList[i].StartTime.After(*snapshotList[j].StartTime)
		}
		// default asc
		return snapshotList[i].StartTime.Before(*snapshotList[j].StartTime)
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

	// If keepCount is set -> keep the first N items, delete the rest
	if keepCount != nil {
		if *keepCount >= n {
			return []types.Snapshot{} // keep all, nothing to delete
		}
		return snapshotList[*keepCount:] // delete everything after the first N
	}

	// If deleteCount is set -> delete the first N items
	if deleteCount != nil {
		if *deleteCount == 0 {
			return []types.Snapshot{} // nothing to delete
		}

		if *deleteCount >= n || *deleteCount < 0 {
			return snapshotList // delete all
		}
		return snapshotList[:*deleteCount] // delete the first N
	}

	// If neither keepCount nor deleteCount is set -> nothing to delete
	return []types.Snapshot{}
}

func deleteSnapShot(snap types.Snapshot, client *ec2.Client) error {
	// _, err := client.DeleteSnapshot(context.TODO(), &ec2.DeleteSnapshotInput{
	// 	SnapshotId: snap.SnapshotId,
	// })

	// if err != nil {
	// 	logger.Errorf("Failed to delete snapshot %v: %v", snap.SnapshotId, err)
	// } else {
	// 	logger.Infof("Deleted snapshot %s (%s)", *snap.SnapshotId, snap.StartTime.Format(time.RFC3339))
	// }
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
		os.Exit(1)
	}

	// Delete count
	logger.Debug("getToDeleteSnapshots")
	removeSnapShot = getToDeleteSnapshots(removeSnapShot, deleteCount, keepCount)
	printSnapShotsList(removeSnapShot, "Delete Snapshot: ")

	for _, snap := range removeSnapShot {
		deleteSnapShot(snap, client)
	}
}
