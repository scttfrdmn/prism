// Package aws provides AWS cloud operations for Prism.
//
// capacity_blocks.go implements EC2 Capacity Reservations / Capacity Blocks for
// ML GPU workloads.  Capacity Blocks allow users to reserve GPU instances for a
// fixed time window so batch ML jobs are guaranteed to start on schedule.
//
// Issue #63 / sub-issue 63a
package aws

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// CapacityBlock describes an EC2 Capacity Reservation used as a capacity block.
type CapacityBlock struct {
	ID               string    `json:"id"`
	InstanceType     string    `json:"instance_type"`
	InstanceCount    int       `json:"instance_count"`
	AvailabilityZone string    `json:"availability_zone"`
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
	DurationHours    int       `json:"duration_hours"`
	State            string    `json:"state"` // payment-pending|active|expired|cancelled
	TotalCost        float64   `json:"total_cost"`
}

// CapacityBlockRequest describes the parameters for reserving a capacity block.
type CapacityBlockRequest struct {
	InstanceType     string    `json:"instance_type"`
	InstanceCount    int       `json:"instance_count"`
	AvailabilityZone string    `json:"availability_zone,omitempty"`
	StartTime        time.Time `json:"start_time"`
	DurationHours    int       `json:"duration_hours"` // 1, 2, 4, 8, 12, 24
}

// ReserveCapacityBlock creates an EC2 Capacity Reservation with targeted match criteria.
func (m *Manager) ReserveCapacityBlock(ctx context.Context, req CapacityBlockRequest) (*CapacityBlock, error) {
	if req.InstanceCount <= 0 {
		req.InstanceCount = 1
	}
	if req.InstanceCount > math.MaxInt32 {
		return nil, fmt.Errorf("instance_count %d is too large", req.InstanceCount)
	}
	if req.DurationHours <= 0 {
		return nil, fmt.Errorf("duration_hours must be positive")
	}

	endTime := req.StartTime.Add(time.Duration(req.DurationHours) * time.Hour)

	input := &ec2.CreateCapacityReservationInput{
		InstanceType:          aws.String(req.InstanceType),
		InstanceCount:         aws.Int32(int32(req.InstanceCount)), //nolint:gosec // G115: bounded to (0, MaxInt32] above
		InstancePlatform:      ec2Types.CapacityReservationInstancePlatformLinuxUnix,
		InstanceMatchCriteria: ec2Types.InstanceMatchCriteriaTargeted,
		StartDate:             aws.Time(req.StartTime),
		EndDate:               aws.Time(endTime),
		EndDateType:           ec2Types.EndDateTypeLimited,
	}
	if req.AvailabilityZone != "" {
		input.AvailabilityZone = aws.String(req.AvailabilityZone)
	}

	out, err := m.ec2.CreateCapacityReservation(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create capacity reservation: %w", err)
	}
	if out.CapacityReservation == nil {
		return nil, fmt.Errorf("empty response from CreateCapacityReservation")
	}

	return capacityBlockFromReservation(out.CapacityReservation), nil
}

// ListCapacityBlocks returns all non-cancelled Capacity Reservations created by Prism.
func (m *Manager) ListCapacityBlocks(ctx context.Context) ([]CapacityBlock, error) {
	out, err := m.ec2.DescribeCapacityReservations(ctx, &ec2.DescribeCapacityReservationsInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("state"),
				Values: []string{"payment-pending", "payment-failed", "active", "expired"},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe capacity reservations: %w", err)
	}

	blocks := make([]CapacityBlock, 0, len(out.CapacityReservations))
	for i := range out.CapacityReservations {
		blocks = append(blocks, *capacityBlockFromReservation(&out.CapacityReservations[i]))
	}
	return blocks, nil
}

// DescribeCapacityBlock returns a single Capacity Reservation by ID.
func (m *Manager) DescribeCapacityBlock(ctx context.Context, id string) (*CapacityBlock, error) {
	out, err := m.ec2.DescribeCapacityReservations(ctx, &ec2.DescribeCapacityReservationsInput{
		CapacityReservationIds: []string{id},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe capacity reservation %s: %w", id, err)
	}
	if len(out.CapacityReservations) == 0 {
		return nil, fmt.Errorf("capacity reservation %s not found", id)
	}
	return capacityBlockFromReservation(&out.CapacityReservations[0]), nil
}

// CancelCapacityBlock cancels an active Capacity Reservation.
func (m *Manager) CancelCapacityBlock(ctx context.Context, id string) error {
	_, err := m.ec2.CancelCapacityReservation(ctx, &ec2.CancelCapacityReservationInput{
		CapacityReservationId: aws.String(id),
	})
	if err != nil {
		return fmt.Errorf("failed to cancel capacity reservation %s: %w", id, err)
	}
	return nil
}

// capacityBlockFromReservation maps an EC2 CapacityReservation to a CapacityBlock.
func capacityBlockFromReservation(r *ec2Types.CapacityReservation) *CapacityBlock {
	cb := &CapacityBlock{
		ID:            aws.ToString(r.CapacityReservationId),
		InstanceType:  aws.ToString(r.InstanceType),
		InstanceCount: int(aws.ToInt32(r.TotalInstanceCount)),
		State:         string(r.State),
	}
	if r.AvailabilityZone != nil {
		cb.AvailabilityZone = *r.AvailabilityZone
	}
	if r.StartDate != nil {
		cb.StartTime = *r.StartDate
	}
	if r.EndDate != nil {
		cb.EndTime = *r.EndDate
		if !cb.StartTime.IsZero() {
			cb.DurationHours = int(cb.EndTime.Sub(cb.StartTime).Hours())
		}
	}
	return cb
}
