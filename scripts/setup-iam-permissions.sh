#!/bin/bash
# CloudWorkstation IAM Permissions Setup Script
# This script helps users set up the required IAM permissions for CloudWorkstation

set -e

echo "🔐 CloudWorkstation IAM Permissions Setup"
echo "==========================================="
echo ""

# Check if AWS CLI is installed
if ! command -v aws &> /dev/null; then
    echo "❌ Error: AWS CLI is not installed"
    echo "   Please install it from: https://aws.amazon.com/cli/"
    exit 1
fi

echo "✅ AWS CLI found"

# Check if AWS credentials are configured
if ! aws sts get-caller-identity &> /dev/null; then
    echo "❌ Error: AWS credentials not configured"
    echo "   Please run: aws configure"
    exit 1
fi

echo "✅ AWS credentials configured"

# Get current AWS identity
IDENTITY=$(aws sts get-caller-identity)
ACCOUNT_ID=$(echo "$IDENTITY" | jq -r '.Account')
USER_ARN=$(echo "$IDENTITY" | jq -r '.Arn')

echo ""
echo "Current AWS Identity:"
echo "  Account ID: $ACCOUNT_ID"
echo "  User/Role:  $USER_ARN"
echo ""

# Determine if this is an IAM user or role
if [[ "$USER_ARN" == *":user/"* ]]; then
    ENTITY_TYPE="user"
    ENTITY_NAME=$(echo "$USER_ARN" | cut -d'/' -f2)
elif [[ "$USER_ARN" == *":role/"* ]]; then
    ENTITY_TYPE="role"
    ENTITY_NAME=$(echo "$USER_ARN" | cut -d'/' -f2)
else
    echo "❌ Error: Unable to determine IAM entity type"
    exit 1
fi

echo "IAM Entity: $ENTITY_TYPE ($ENTITY_NAME)"
echo ""

# Prompt for action
echo "What would you like to do?"
echo "  1) Create new IAM policy and attach to current $ENTITY_TYPE"
echo "  2) Attach existing CloudWorkstation policy to current $ENTITY_TYPE"
echo "  3) Create new IAM user for CloudWorkstation"
echo "  4) Show policy JSON only (no changes)"
echo "  5) Exit"
echo ""
read -p "Enter choice [1-5]: " CHOICE

case $CHOICE in
    1)
        echo ""
        echo "📝 Creating CloudWorkstation IAM policy..."

        # Check if policy already exists
        POLICY_ARN="arn:aws:iam::$ACCOUNT_ID:policy/CloudWorkstationAccess"
        if aws iam get-policy --policy-arn "$POLICY_ARN" &> /dev/null; then
            echo "⚠️  Policy already exists: $POLICY_ARN"
            read -p "Would you like to create a new version? [y/N]: " CREATE_VERSION
            if [[ "$CREATE_VERSION" =~ ^[Yy]$ ]]; then
                # Create new policy version
                POLICY_DOCUMENT=$(cat "$(dirname "$0")/../docs/prism-iam-policy.json")
                aws iam create-policy-version \
                    --policy-arn "$POLICY_ARN" \
                    --policy-document "$POLICY_DOCUMENT" \
                    --set-as-default
                echo "✅ Created new policy version"
            fi
        else
            # Create new policy
            aws iam create-policy \
                --policy-name CloudWorkstationAccess \
                --policy-document file://$(dirname "$0")/../docs/prism-iam-policy.json \
                --description "Full access permissions for CloudWorkstation research platform"
            echo "✅ Created policy: $POLICY_ARN"
        fi

        # Attach policy to current entity
        echo "📎 Attaching policy to $ENTITY_TYPE: $ENTITY_NAME"
        if [[ "$ENTITY_TYPE" == "user" ]]; then
            aws iam attach-user-policy \
                --user-name "$ENTITY_NAME" \
                --policy-arn "$POLICY_ARN"
        else
            aws iam attach-role-policy \
                --role-name "$ENTITY_NAME" \
                --policy-arn "$POLICY_ARN"
        fi

        echo "✅ Policy attached successfully"
        echo ""
        echo "🎉 Setup complete! You can now use CloudWorkstation."
        ;;

    2)
        echo ""
        POLICY_ARN="arn:aws:iam::$ACCOUNT_ID:policy/CloudWorkstationAccess"
        echo "📎 Attaching existing policy: $POLICY_ARN"

        # Check if policy exists
        if ! aws iam get-policy --policy-arn "$POLICY_ARN" &> /dev/null; then
            echo "❌ Error: Policy does not exist: $POLICY_ARN"
            echo "   Please choose option 1 to create it first"
            exit 1
        fi

        # Attach policy
        if [[ "$ENTITY_TYPE" == "user" ]]; then
            aws iam attach-user-policy \
                --user-name "$ENTITY_NAME" \
                --policy-arn "$POLICY_ARN"
        else
            aws iam attach-role-policy \
                --role-name "$ENTITY_NAME" \
                --policy-arn "$POLICY_ARN"
        fi

        echo "✅ Policy attached successfully"
        ;;

    3)
        echo ""
        read -p "Enter new IAM user name: " NEW_USER_NAME

        echo "👤 Creating IAM user: $NEW_USER_NAME"
        aws iam create-user --user-name "$NEW_USER_NAME"

        echo "🔑 Creating access key..."
        ACCESS_KEY=$(aws iam create-access-key --user-name "$NEW_USER_NAME")
        ACCESS_KEY_ID=$(echo "$ACCESS_KEY" | jq -r '.AccessKey.AccessKeyId')
        SECRET_ACCESS_KEY=$(echo "$ACCESS_KEY" | jq -r '.AccessKey.SecretAccessKey')

        echo "📝 Creating CloudWorkstation policy..."
        POLICY_ARN="arn:aws:iam::$ACCOUNT_ID:policy/CloudWorkstationAccess"
        if ! aws iam get-policy --policy-arn "$POLICY_ARN" &> /dev/null; then
            aws iam create-policy \
                --policy-name CloudWorkstationAccess \
                --policy-document file://$(dirname "$0")/../docs/prism-iam-policy.json \
                --description "Full access permissions for CloudWorkstation research platform"
        fi

        echo "📎 Attaching policy to user..."
        aws iam attach-user-policy \
            --user-name "$NEW_USER_NAME" \
            --policy-arn "$POLICY_ARN"

        echo ""
        echo "✅ User created successfully!"
        echo ""
        echo "⚠️  IMPORTANT: Save these credentials securely (they won't be shown again):"
        echo "  Access Key ID:     $ACCESS_KEY_ID"
        echo "  Secret Access Key: $SECRET_ACCESS_KEY"
        echo ""
        echo "Add to ~/.aws/credentials:"
        echo "  [$NEW_USER_NAME]"
        echo "  aws_access_key_id = $ACCESS_KEY_ID"
        echo "  aws_secret_access_key = $SECRET_ACCESS_KEY"
        echo ""
        echo "Then create CloudWorkstation profile:"
        echo "  cws profiles add myprofile myprofile --aws-profile $NEW_USER_NAME --region us-west-2"
        ;;

    4)
        echo ""
        echo "📄 CloudWorkstation IAM Policy:"
        echo "================================"
        cat "$(dirname "$0")/../docs/prism-iam-policy.json"
        echo ""
        echo "To apply manually:"
        echo "  aws iam create-policy \\"
        echo "    --policy-name CloudWorkstationAccess \\"
        echo "    --policy-document file://docs/prism-iam-policy.json"
        ;;

    5)
        echo "Exiting..."
        exit 0
        ;;

    *)
        echo "❌ Invalid choice"
        exit 1
        ;;
esac

echo ""
echo "📚 For more information, see: docs/AWS_IAM_PERMISSIONS.md"
