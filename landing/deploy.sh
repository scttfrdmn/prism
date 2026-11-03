#!/bin/bash
# Deploy the Prism landing page to prismcloud.host (S3 + CloudFront).
#
# The bucket is private: CloudFront reads it through Origin Access Control, so
# the bucket has no public policy or static-website hosting. The bucket, ACM
# certificate, CloudFront distribution and Route 53 records were created once
# (#686); this script only uploads the page and invalidates the cache.

set -e

BUCKET_NAME="${PRISM_WEBSITE_BUCKET:-prismcloud-host-website}"
REGION="us-east-1"
DOMAIN="prismcloud.host"
AWS_PROFILE="${AWS_PROFILE:-spore-host-infra}"      # holds the prismcloud.host zone
DISTRIBUTION_ID="${PRISM_CLOUDFRONT_ID:-E5FM6YPIFHGR9}"

GREEN='\033[0;32m'; BLUE='\033[0;34m'; RED='\033[0;31m'; NC='\033[0m'

echo -e "${BLUE}Deploying Prism landing page → ${DOMAIN}${NC}"

command -v aws >/dev/null 2>&1 || { echo -e "${RED}AWS CLI not installed${NC}"; exit 1; }

echo -e "${BLUE}→${NC} Checking AWS credentials (profile: ${AWS_PROFILE})..."
aws sts get-caller-identity --profile "$AWS_PROFILE" >/dev/null 2>&1 || {
  echo -e "${RED}AWS credentials not configured for profile $AWS_PROFILE${NC}"; exit 1; }
echo -e "${GREEN}✓${NC} credentials OK"

aws s3api head-bucket --bucket "$BUCKET_NAME" --profile "$AWS_PROFILE" >/dev/null 2>&1 || {
  echo -e "${RED}Bucket $BUCKET_NAME not found (see landing/README.md for the one-time setup)${NC}"; exit 1; }

# Upload the landing page (this directory), excluding tooling
echo -e "${BLUE}→${NC} uploading landing files..."
aws s3 sync "$(dirname "$0")" "s3://$BUCKET_NAME/" \
  --region "$REGION" \
  --delete \
  --exclude ".DS_Store" --exclude "*.sh" --exclude "README.md" \
  --cache-control "max-age=3600" \
  --profile "$AWS_PROFILE"
echo -e "${GREEN}✓${NC} uploaded"

echo -e "${BLUE}→${NC} invalidating CloudFront ($DISTRIBUTION_ID)..."
aws cloudfront create-invalidation --distribution-id "$DISTRIBUTION_ID" \
  --paths "/*" --profile "$AWS_PROFILE" --query 'Invalidation.Id' --output text

echo -e "${GREEN}✓ done${NC}  (https://$DOMAIN)"
