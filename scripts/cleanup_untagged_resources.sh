#!/bin/bash
# Prism Zombie Resource Cleanup Script
# Identifies and optionally terminates AWS resources NOT tagged with Prism=true

set -e

AWS_PROFILE="${AWS_PROFILE:-aws}"
DRY_RUN="${DRY_RUN:-true}"
FORCE="${FORCE:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "============================================"
echo "Prism Zombie Resource Cleanup"
echo "============================================"
echo "AWS Profile: $AWS_PROFILE"
echo "Dry Run: $DRY_RUN"
echo ""

# Function to check if instance has Prism tag
is_prism_managed() {
    local instance_id=$1
    local tags=$(aws ec2 describe-tags \
        --filters "Name=resource-id,Values=$instance_id" "Name=key,Values=Prism" \
        --query 'Tags[0].Value' \
        --output text \
        --profile "$AWS_PROFILE" 2>/dev/null)

    if [[ "$tags" == "true" ]]; then
        return 0  # Is Prism-managed
    else
        return 1  # Not Prism-managed
    fi
}

# Find all running/stopped instances
echo "🔍 Scanning for EC2 instances..."
instances=$(aws ec2 describe-instances \
    --filters "Name=instance-state-name,Values=running,stopped,pending" \
    --query 'Reservations[*].Instances[*].[InstanceId,State.Name,InstanceType,LaunchTime,Tags[?Key==`Name`].Value|[0]]' \
    --output text \
    --profile "$AWS_PROFILE")

zombie_instances=()
prism_instances=()

while IFS=$'\t' read -r instance_id state instance_type launch_time name; do
    if [[ -z "$instance_id" ]]; then continue; fi

    if is_prism_managed "$instance_id"; then
        prism_instances+=("$instance_id")
        echo -e "${GREEN}✓${NC} $instance_id ($state, $instance_type) - Prism-managed: ${name:-<unnamed>}"
    else
        zombie_instances+=("$instance_id|$state|$instance_type|$launch_time|$name")
        echo -e "${RED}✗${NC} $instance_id ($state, $instance_type) - ZOMBIE: ${name:-<unnamed>} (launched: $launch_time)"
    fi
done <<< "$instances"

echo ""
echo "Summary:"
echo "  Prism-managed instances: ${#prism_instances[@]}"
echo "  Zombie instances: ${#zombie_instances[@]}"
echo ""

# Find unattached EBS volumes
echo "🔍 Scanning for unattached EBS volumes..."
volumes=$(aws ec2 describe-volumes \
    --filters "Name=status,Values=available" \
    --query 'Volumes[*].[VolumeId,Size,CreateTime]' \
    --output text \
    --profile "$AWS_PROFILE")

zombie_volumes=()
while IFS=$'\t' read -r volume_id size create_time; do
    if [[ -z "$volume_id" ]]; then continue; fi
    zombie_volumes+=("$volume_id|$size|$create_time")
    echo -e "${RED}✗${NC} $volume_id (${size}GB) - Created: $create_time"
done <<< "$volumes"

echo "  Unattached volumes: ${#zombie_volumes[@]}"
echo ""

# Calculate potential cost savings
total_cost=0
for zombie in "${zombie_instances[@]}"; do
    IFS='|' read -r id state type launch_time name <<< "$zombie"

    # Rough cost estimation (simplified)
    case $type in
        t3.xlarge) cost=0.166 ;;
        c7g.4xlarge) cost=0.58 ;;
        t3.large) cost=0.083 ;;
        t3.medium) cost=0.042 ;;
        *) cost=0.10 ;;
    esac

    total_cost=$(echo "$total_cost + $cost * 24 * 30" | bc)
done

for zombie in "${zombie_volumes[@]}"; do
    IFS='|' read -r id size create_time <<< "$zombie"
    volume_cost=$(echo "$size * 0.10" | bc)  # $0.10/GB/month
    total_cost=$(echo "$total_cost + $volume_cost" | bc)
done

echo "💰 Estimated monthly cost of zombie resources: \$$(printf "%.2f" $total_cost)"
echo ""

# Cleanup actions
if [[ ${#zombie_instances[@]} -gt 0 ]] || [[ ${#zombie_volumes[@]} -gt 0 ]]; then
    if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "${YELLOW}⚠️  DRY RUN MODE - No resources will be terminated${NC}"
        echo ""
        echo "To actually terminate these resources, run:"
        echo "  DRY_RUN=false $0"
        echo ""
        echo "Or to force cleanup without confirmation:"
        echo "  DRY_RUN=false FORCE=true $0"
    else
        echo -e "${RED}⚠️  TERMINATION MODE ENABLED${NC}"

        if [[ "$FORCE" != "true" ]]; then
            echo ""
            echo "This will terminate:"
            echo "  - ${#zombie_instances[@]} EC2 instances"
            echo "  - ${#zombie_volumes[@]} EBS volumes"
            echo ""
            read -p "Are you sure? Type 'yes' to confirm: " confirmation

            if [[ "$confirmation" != "yes" ]]; then
                echo "Aborted."
                exit 0
            fi
        fi

        # Terminate zombie instances
        for zombie in "${zombie_instances[@]}"; do
            IFS='|' read -r id state type launch_time name <<< "$zombie"
            echo "Terminating $id..."
            aws ec2 terminate-instances --instance-ids "$id" --profile "$AWS_PROFILE"
        done

        # Delete zombie volumes
        for zombie in "${zombie_volumes[@]}"; do
            IFS='|' read -r id size create_time <<< "$zombie"
            echo "Deleting volume $id..."
            aws ec2 delete-volume --volume-id "$id" --profile "$AWS_PROFILE"
        done

        echo -e "${GREEN}✓ Cleanup complete!${NC}"
        echo "Estimated monthly savings: \$$(printf "%.2f" $total_cost)"
    fi
else
    echo -e "${GREEN}✓ No zombie resources found!${NC}"
fi

echo ""
echo "============================================"
