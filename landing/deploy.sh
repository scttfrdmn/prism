#!/bin/bash
# Deploy the Prism landing page to S3 + CloudFront.
#
# Mirrors the spore.host web/deploy.sh pattern (S3 static website + CloudFront +
# Route53 alias) so prismcloud.host follows the same infra shape as spore.host.
#
# One-time setup (see landing/README.md and the tracking issue): create the S3
# bucket, an ACM cert for prismcloud.host in us-east-1, a CloudFront distribution
# with that cert + the apex as an alternate domain name, and a Route53 A/ALIAS
# record for the apex → CloudFront. Then set DISTRIBUTION_ID below (or via env).

set -e

BUCKET_NAME="${PRISM_WEBSITE_BUCKET:-prismcloud-host-website}"
REGION="us-east-1"
DOMAIN="prismcloud.host"
AWS_PROFILE="${AWS_PROFILE:-prism-infra}"        # follow spore-host-infra convention
DISTRIBUTION_ID="${PRISM_CLOUDFRONT_ID:-}"       # set after the distribution exists

GREEN='\033[0;32m'; BLUE='\033[0;34m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'

echo -e "${BLUE}Deploying Prism landing page → ${DOMAIN}${NC}"

command -v aws >/dev/null 2>&1 || { echo -e "${RED}AWS CLI not installed${NC}"; exit 1; }

echo -e "${BLUE}→${NC} Checking AWS credentials (profile: ${AWS_PROFILE})..."
aws sts get-caller-identity --profile "$AWS_PROFILE" >/dev/null 2>&1 || {
  echo -e "${RED}AWS credentials not configured for profile $AWS_PROFILE${NC}"; exit 1; }
echo -e "${GREEN}✓${NC} credentials OK"

# S3 bucket (create if missing)
if ! aws s3 ls "s3://$BUCKET_NAME" --profile "$AWS_PROFILE" >/dev/null 2>&1; then
  echo -e "${YELLOW}!${NC} creating bucket $BUCKET_NAME"
  aws s3 mb "s3://$BUCKET_NAME" --region "$REGION" --profile "$AWS_PROFILE"
fi

aws s3 website "s3://$BUCKET_NAME" \
  --index-document index.html --error-document index.html \
  --profile "$AWS_PROFILE"

cat > /tmp/prism-bucket-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Sid": "PublicReadGetObject",
    "Effect": "Allow",
    "Principal": "*",
    "Action": "s3:GetObject",
    "Resource": "arn:aws:s3:::$BUCKET_NAME/*"
  }]
}
EOF
aws s3api put-bucket-policy --bucket "$BUCKET_NAME" \
  --policy file:///tmp/prism-bucket-policy.json --profile "$AWS_PROFILE"
rm -f /tmp/prism-bucket-policy.json

# Upload the landing page (this directory), excluding tooling
echo -e "${BLUE}→${NC} uploading landing files..."
aws s3 sync "$(dirname "$0")" "s3://$BUCKET_NAME/" \
  --delete \
  --exclude ".DS_Store" --exclude "*.sh" --exclude "README.md" \
  --cache-control "max-age=3600" \
  --profile "$AWS_PROFILE"
echo -e "${GREEN}✓${NC} uploaded"

if [ -n "$DISTRIBUTION_ID" ]; then
  echo -e "${BLUE}→${NC} invalidating CloudFront ($DISTRIBUTION_ID)..."
  aws cloudfront create-invalidation --distribution-id "$DISTRIBUTION_ID" \
    --paths "/*" --profile "$AWS_PROFILE" --query 'Invalidation.Id' --output text
else
  echo -e "${YELLOW}!${NC} PRISM_CLOUDFRONT_ID not set — skipped cache invalidation."
  echo "  Create the CloudFront distribution + Route53 alias once, then set it."
fi

echo -e "${GREEN}✓ done${NC}  (S3 website: http://$BUCKET_NAME.s3-website-$REGION.amazonaws.com)"
