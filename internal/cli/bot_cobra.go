// Package cli provides Slack/Teams bot integration commands for Prism.
//
// These commands allow workspace owners to register their Prism instances
// with the shared spore-bot Slack service and manage per-user access:
//
//	prism bot workspace-add   -- register Slack app credentials
//	prism bot register        -- link a Slack/Teams user to an instance
//	prism bot enable          -- activate bot access (off by default)
//	prism bot disable         -- suspend access without removing registration
//	prism bot revoke          -- remove registration
//	prism bot status          -- show registered users and their access state
package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/spf13/cobra"
)

const (
	// botAdminURL is the API Gateway admin endpoint (SigV4-signed via assumed role).
	// Distinct from the Lambda Function URL used for Slack/Teams webhooks.
	botAdminURL   = "https://g8iytgzrn8.execute-api.us-east-1.amazonaws.com"
	botAWSRegion  = "us-east-1"
	botAWSService = "execute-api"

	// botAdminRoleARN is a public assume-role in spore-host-infra that grants
	// execute-api:Invoke on the admin API. Cross-account HTTP API v2 with AWS_IAM
	// auth cannot use resource policies, so callers assume this role instead.
	botAdminRoleARN    = "arn:aws:iam::966362334030:role/SpawnBotAdminCaller"
	botAdminExternalID = "spore-bot"

	// botCrossAccountRoleName is the IAM role created in the professor's AWS account
	// that allows the spore-bot Lambda to call EC2 start/stop/describe.
	botCrossAccountRoleName   = "SpawnBotCrossAccount"
	botLambdaExecutionRoleARN = "arn:aws:iam::966362334030:role/prism-bot-PrismBotFunctionRole-U2vZFZXgWBeM"
)

// BotCobraCommands provides the `prism bot` command group.
type BotCobraCommands struct {
	app *App
}

// NewBotCobraCommands creates new bot CLI commands.
func NewBotCobraCommands(app *App) *BotCobraCommands {
	return &BotCobraCommands{app: app}
}

// CreateBotCommand creates the `prism bot` parent command.
func (b *BotCobraCommands) CreateBotCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bot",
		Short: "Manage Slack/Teams bot integration",
		Long: `Register your Prism workspaces with the Slack/Teams bot so collaborators
can start, stop, and check status using slash commands — no CLI required.

Quick setup:
  1. Create a Slack app at https://api.slack.com/apps (slash command /prism)
  2. prism bot workspace-add  -- register your Slack credentials
  3. aws cloudformation deploy ...  -- cross-account IAM role (see docs)
  4. prism bot register  -- link collaborators to instances
  5. prism bot enable    -- activate access (off by default)

Slash commands your team can then use:
  /prism status [name]      -- instance state and URL
  /prism start [name]       -- start a stopped instance
  /prism stop [name]        -- stop a running instance
  /prism hibernate [name]   -- hibernate (saves RAM, zero compute cost)
  /prism url [name]         -- get the RStudio/Jupyter URL
  /prism notify <name>      -- subscribe to lifecycle notifications
  /prism connect [duration] -- generate a one-time registration code`,
	}
	cmd.AddCommand(b.createWorkspaceAddCommand())
	cmd.AddCommand(b.createWorkspaceListCommand())
	cmd.AddCommand(b.createRegisterCommand())
	cmd.AddCommand(b.createEnableCommand())
	cmd.AddCommand(b.createDisableCommand())
	cmd.AddCommand(b.createRevokeCommand())
	cmd.AddCommand(b.createStatusCommand())
	return cmd
}

func (b *BotCobraCommands) createWorkspaceAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace-add",
		Short: "Register Slack workspace credentials with the bot service",
		Long: `Store your Slack app's bot token and signing secret in the shared
spore-bot service so the Lambda can verify incoming slash commands and
post responses back to your workspace.

Run this once after creating your Slack app. Credentials are encrypted
at rest in DynamoDB and isolated by workspace key.`,
		Example: `  prism bot workspace-add \
    --workspace-id T0AU2S6FU86 \
    --workspace-name "Prism Research" \
    --bot-token xoxb-... \
    --signing-secret abc123...`,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID, _ := cmd.Flags().GetString("workspace-id")
			workspaceName, _ := cmd.Flags().GetString("workspace-name")
			botToken, _ := cmd.Flags().GetString("bot-token")
			signingSecret, _ := cmd.Flags().GetString("signing-secret")
			platform, _ := cmd.Flags().GetString("platform")
			allowedChannels, _ := cmd.Flags().GetString("allowed-channels")

			payload := map[string]interface{}{
				"platform":       platform,
				"workspace_id":   workspaceID,
				"workspace_name": workspaceName,
				"bot_token":      botToken,
				"signing_secret": signingSecret,
			}
			if allowedChannels != "" {
				payload["allowed_channels"] = allowedChannels
			}

			fmt.Printf("Registering workspace %s...\n", workspaceID)
			resp, err := b.callAdmin(cmd.Context(), "POST", "/admin/workspace-add", payload)
			if err != nil {
				return fmt.Errorf("workspace registration failed: %w", err)
			}

			fmt.Printf("✅ Workspace registered: %s (%s)\n", workspaceName, workspaceID)
			if v, ok := resp["message"].(string); ok && v != "" {
				fmt.Printf("   %s\n", v)
			}
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Println("  1. Deploy cross-account IAM role in your AWS account:")
			fmt.Println("       aws cloudformation deploy \\")
			fmt.Println("         --stack-name prism-bot-cross-account \\")
			fmt.Println("         --template-file spawn/deployment/cloudformation/bot-cross-account-role.yaml \\")
			fmt.Println("         --capabilities CAPABILITY_NAMED_IAM \\")
			fmt.Println("         --parameter-overrides TagPrefix=prism")
			fmt.Println("  2. Register users:  prism bot register --user email@example.com ...")
			fmt.Println("  3. Enable access:   prism bot enable ...")
			return nil
		},
	}
	cmd.Flags().String("workspace-id", "", "Slack workspace ID (e.g., T0AU2S6FU86)")
	cmd.Flags().String("workspace-name", "", "Human-readable workspace name")
	cmd.Flags().String("bot-token", "", "Slack bot token (xoxb-...)")
	cmd.Flags().String("signing-secret", "", "Slack app signing secret")
	cmd.Flags().String("platform", "slack", "Platform: slack or teams")
	cmd.Flags().String("allowed-channels", "", "Comma-separated channel IDs to restrict commands (empty = all)")
	_ = cmd.MarkFlagRequired("workspace-id")
	_ = cmd.MarkFlagRequired("workspace-name")
	_ = cmd.MarkFlagRequired("bot-token")
	_ = cmd.MarkFlagRequired("signing-secret")
	return cmd
}

func (b *BotCobraCommands) createWorkspaceListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "workspace-list",
		Short: "List registered Slack/Teams workspaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := b.callAdmin(cmd.Context(), "GET", "/admin/workspace-list", nil)
			if err != nil {
				return fmt.Errorf("failed to list workspaces: %w", err)
			}
			workspaces, _ := resp["workspaces"].([]interface{})
			if len(workspaces) == 0 {
				fmt.Println("No workspaces registered. Run: prism bot workspace-add")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "WORKSPACE ID\tNAME\tPLATFORM")
			for _, ws := range workspaces {
				if m, ok := ws.(map[string]interface{}); ok {
					fmt.Fprintf(w, "%s\t%s\t%s\n", m["workspace_id"], m["workspace_name"], m["platform"])
				}
			}
			return w.Flush()
		},
	}
}

func (b *BotCobraCommands) createRegisterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Link a Slack/Teams user to a Prism instance",
		Long: `Create a bot registration mapping a chat user to a specific instance.
Access is disabled by default — run 'prism bot enable' to activate it.

Users can self-register by typing '/prism connect' in Slack to get a
one-time code, then pass it via --connect-code here.`,
		Example: `  # By email (auto-resolves to Slack user ID)
  prism bot register \
    --user professor@illinois.edu \
    --workspace-id T0AU2S6FU86 \
    --instance my-rstudio \
    --nickname rstudio \
    --role-arn arn:aws:iam::123456789012:role/prism-bot-SpawnBotCrossAccount

  # Via connect code (/prism connect in Slack)
  prism bot register \
    --connect-code SPORE-3F9A2C \
    --instance my-rstudio \
    --nickname rstudio \
    --role-arn arn:aws:iam::123456789012:role/prism-bot-SpawnBotCrossAccount`,
		RunE: func(cmd *cobra.Command, args []string) error {
			user, _ := cmd.Flags().GetString("user")
			userID, _ := cmd.Flags().GetString("user-id")
			workspaceID, _ := cmd.Flags().GetString("workspace-id")
			connectCode, _ := cmd.Flags().GetString("connect-code")
			instance, _ := cmd.Flags().GetString("instance")
			nickname, _ := cmd.Flags().GetString("nickname")
			roleARN, _ := cmd.Flags().GetString("role-arn")
			allow, _ := cmd.Flags().GetString("allow")
			platform, _ := cmd.Flags().GetString("platform")

			if user == "" && userID == "" && connectCode == "" {
				return fmt.Errorf("one of --user, --user-id, or --connect-code is required")
			}

			// Auto-create the cross-account IAM role if not explicitly provided.
			if roleARN == "" {
				fmt.Println("Ensuring SpawnBotCrossAccount role exists in your account...")
				baseCfg, err := config.LoadDefaultConfig(cmd.Context())
				if err != nil {
					return fmt.Errorf("load AWS config: %w", err)
				}
				roleARN, err = ensureCrossAccountRole(cmd.Context(), baseCfg)
				if err != nil {
					return fmt.Errorf("create cross-account role: %w (use --role-arn to provide an existing role)", err)
				}
				fmt.Printf("  ✓ %s\n", roleARN)
			}

			payload := map[string]interface{}{
				"platform":   platform,
				"instance":   instance,
				"nickname":   nickname,
				"role_arn":   roleARN,
				"tag_prefix": "prism",
				"allow":      allow,
			}
			if user != "" {
				payload["user"] = user
			}
			if userID != "" {
				payload["user_id"] = userID
			}
			if workspaceID != "" {
				payload["workspace_id"] = workspaceID
			}
			if connectCode != "" {
				payload["connect_code"] = connectCode
			}

			resp, err := b.callAdmin(cmd.Context(), "POST", "/admin/register", payload)
			if err != nil {
				return fmt.Errorf("registration failed: %w", err)
			}

			displayUser := user
			if displayUser == "" {
				displayUser = userID
			}
			if displayUser == "" {
				displayUser = "(via connect code)"
			}
			fmt.Printf("✅ Registered: %s → %s (nickname: %s)\n", displayUser, instance, nickname)
			if v, ok := resp["message"].(string); ok && v != "" {
				fmt.Printf("   %s\n", v)
			}
			fmt.Printf("\nAccess is disabled by default. To activate:\n")
			fmt.Printf("  prism bot enable --user %s --workspace-id %s --nickname %s\n",
				displayUser, workspaceID, nickname)
			return nil
		},
	}
	cmd.Flags().String("user", "", "User email (resolved to Slack ID via workspace bot token)")
	cmd.Flags().String("user-id", "", "Slack/Teams user ID (e.g., U04KZABCD)")
	cmd.Flags().String("workspace-id", "", "Slack workspace ID")
	cmd.Flags().String("connect-code", "", "One-time code from /prism connect")
	cmd.Flags().String("instance", "", "Prism instance name or ID")
	cmd.Flags().String("nickname", "", "Short name for slash commands (e.g., rstudio)")
	cmd.Flags().String("role-arn", "", "Cross-account IAM role ARN (auto-created if omitted; requires iam:CreateRole)")
	cmd.Flags().String("allow", "start,stop,status,hibernate,url", "Comma-separated allowed actions")
	cmd.Flags().String("platform", "slack", "Platform: slack or teams")
	_ = cmd.MarkFlagRequired("instance")
	_ = cmd.MarkFlagRequired("nickname")
	return cmd
}

func (b *BotCobraCommands) createEnableCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable bot access for a registered user (off by default)",
		Long: `Registrations are disabled by default. The instance owner must
explicitly enable each user's access. This is the security gate that
prevents unauthorized control even if a registration exists.`,
		RunE: func(cmd *cobra.Command, args []string) error { return b.setEnabled(cmd, true) },
	}
	addBotUserFlags(cmd)
	return cmd
}

func (b *BotCobraCommands) createDisableCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Suspend bot access without removing the registration",
		Long: `Temporarily suspend a user's bot access. The registration remains so
it can be re-enabled quickly. Use during sensitive computations or
when you need a temporary pause. To permanently remove, use 'revoke'.`,
		RunE: func(cmd *cobra.Command, args []string) error { return b.setEnabled(cmd, false) },
	}
	addBotUserFlags(cmd)
	return cmd
}

func (b *BotCobraCommands) createRevokeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke",
		Short: "Permanently remove a bot registration",
		RunE: func(cmd *cobra.Command, args []string) error {
			user, _ := cmd.Flags().GetString("user")
			userID, _ := cmd.Flags().GetString("user-id")
			workspaceID, _ := cmd.Flags().GetString("workspace-id")
			nickname, _ := cmd.Flags().GetString("nickname")
			platform, _ := cmd.Flags().GetString("platform")

			payload := map[string]interface{}{
				"platform":     platform,
				"workspace_id": workspaceID,
				"nickname":     nickname,
			}
			if user != "" {
				payload["user"] = user
			}
			if userID != "" {
				payload["user_id"] = userID
			}

			_, err := b.callAdmin(cmd.Context(), "POST", "/admin/deregister", payload)
			if err != nil {
				return fmt.Errorf("revoke failed: %w", err)
			}
			displayUser := user
			if displayUser == "" {
				displayUser = userID
			}
			fmt.Printf("✅ Revoked: %s (%s)\n", displayUser, nickname)
			return nil
		},
	}
	addBotUserFlags(cmd)
	_ = cmd.MarkFlagRequired("nickname")
	return cmd
}

func (b *BotCobraCommands) createStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show bot registrations and access state for your workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID, _ := cmd.Flags().GetString("workspace-id")
			platform, _ := cmd.Flags().GetString("platform")

			resp, err := b.callAdmin(cmd.Context(), "GET", "/admin/list", map[string]interface{}{
				"platform":     platform,
				"workspace_id": workspaceID,
			})
			if err != nil {
				return fmt.Errorf("failed to get status: %w", err)
			}

			registrations, _ := resp["registrations"].([]interface{})
			if len(registrations) == 0 {
				fmt.Println("No bot registrations found.")
				fmt.Println("Run: prism bot register --user email@example.com ...")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "USER\tNICKNAME\tINSTANCE\tSTATUS\tLAST USED")
			for _, reg := range registrations {
				if m, ok := reg.(map[string]interface{}); ok {
					status := "⏸  disabled"
					if e, ok := m["enabled"].(bool); ok && e {
						status = "✅ enabled"
					}
					lastUsed := "never"
					if lu, ok := m["last_used"].(string); ok && lu != "" {
						lastUsed = lu
					}
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
						m["user_email"], m["nickname"], m["instance_id"], status, lastUsed)
				}
			}
			return w.Flush()
		},
	}
	cmd.Flags().String("workspace-id", "", "Filter by Slack workspace ID")
	cmd.Flags().String("platform", "slack", "Platform: slack or teams")
	return cmd
}

// --- helpers ---

func addBotUserFlags(cmd *cobra.Command) {
	cmd.Flags().String("user", "", "User email")
	cmd.Flags().String("user-id", "", "Slack/Teams user ID")
	cmd.Flags().String("workspace-id", "", "Slack workspace ID")
	cmd.Flags().String("nickname", "", "Instance nickname")
	cmd.Flags().String("platform", "slack", "Platform: slack or teams")
}

func (b *BotCobraCommands) setEnabled(cmd *cobra.Command, enabled bool) error {
	user, _ := cmd.Flags().GetString("user")
	userID, _ := cmd.Flags().GetString("user-id")
	workspaceID, _ := cmd.Flags().GetString("workspace-id")
	nickname, _ := cmd.Flags().GetString("nickname")
	platform, _ := cmd.Flags().GetString("platform")

	payload := map[string]interface{}{
		"platform":     platform,
		"workspace_id": workspaceID,
		"nickname":     nickname,
		"enabled":      enabled,
	}
	if user != "" {
		payload["user"] = user
	}
	if userID != "" {
		payload["user_id"] = userID
	}

	_, err := b.callAdmin(cmd.Context(), "POST", "/admin/set-enabled", payload)
	if err != nil {
		return fmt.Errorf("failed to update access: %w", err)
	}

	displayUser := user
	if displayUser == "" {
		displayUser = userID
	}
	if enabled {
		fmt.Printf("✅ Bot access enabled: %s (%s)\n", displayUser, nickname)
		fmt.Println("   Users can now use /prism commands in Slack.")
	} else {
		fmt.Printf("⏸️  Bot access suspended: %s (%s)\n", displayUser, nickname)
		fmt.Printf("   Re-enable: prism bot enable --user %s --nickname %s\n", displayUser, nickname)
	}
	return nil
}

// ensureCrossAccountRole creates (or reuses) the SpawnBotCrossAccount IAM role in the
// caller's AWS account. This role allows the spore-bot Lambda to call EC2 start/stop
// on the professor's instances. Called automatically by 'prism bot register' when
// --role-arn is not provided. Requires iam:GetRole, iam:CreateRole, iam:PutRolePolicy.
func ensureCrossAccountRole(ctx context.Context, cfg aws.Config) (string, error) {
	client := iam.NewFromConfig(cfg)

	existing, err := client.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(botCrossAccountRoleName),
	})
	if err == nil {
		return *existing.Role.Arn, nil
	}

	// Check it's a not-found error, not something else
	var notFound *iamtypes.NoSuchEntityException
	if !errors.As(err, &notFound) {
		return "", fmt.Errorf("check role: %w", err)
	}

	trustPolicy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [{
			"Effect": "Allow",
			"Principal": {"AWS": %q},
			"Action": "sts:AssumeRole",
			"Condition": {"StringEquals": {"sts:ExternalId": "spawn-bot"}}
		}]
	}`, botLambdaExecutionRoleARN)

	created, err := client.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String(botCrossAccountRoleName),
		AssumeRolePolicyDocument: aws.String(trustPolicy),
		Description:              aws.String("Allows spore-bot Lambda to control EC2 instances via Slack/Teams"),
	})
	if err != nil {
		return "", fmt.Errorf("create role: %w", err)
	}

	_, err = client.PutRolePolicy(ctx, &iam.PutRolePolicyInput{
		RoleName:   aws.String(botCrossAccountRoleName),
		PolicyName: aws.String("SpawnBotEC2Control"),
		PolicyDocument: aws.String(`{
			"Version": "2012-10-17",
			"Statement": [{
				"Effect": "Allow",
				"Action": [
					"ec2:DescribeInstances",
					"ec2:DescribeTags",
					"ec2:StartInstances",
					"ec2:StopInstances"
				],
				"Resource": "*"
			}]
		}`),
	})
	if err != nil {
		return "", fmt.Errorf("attach policy: %w", err)
	}
	return *created.Role.Arn, nil
}

// callAdmin makes an AWS SigV4-signed HTTPS request to the spore-bot admin API.
// Assumes the public SpawnBotAdminCaller role in spore-host-infra so the caller
// is same-account to the HTTP API (cross-account HTTP API v2 IAM auth is unsupported).
// The Lambda enforces per-account isolation via verifyWorkspaceOwner().
func (b *BotCobraCommands) callAdmin(ctx context.Context, method, path string, payload interface{}) (map[string]interface{}, error) {
	var bodyBytes []byte
	var err error
	if payload != nil {
		bodyBytes, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
	}

	url := botAdminURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Prism-Tag-Prefix", "prism")

	// Load base config from current AWS profile, then assume the admin role.
	// stscreds.NewAssumeRoleProvider handles caching and automatic refresh.
	baseCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	stsClient := sts.NewFromConfig(baseCfg)
	roleProvider := stscreds.NewAssumeRoleProvider(stsClient, botAdminRoleARN,
		func(o *stscreds.AssumeRoleOptions) {
			o.RoleSessionName = "prism-bot-admin"
			o.ExternalID = aws.String(botAdminExternalID)
		},
	)
	creds, err := roleProvider.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("assume admin role: %w", err)
	}

	h := sha256.Sum256(bodyBytes)
	bodyHash := hex.EncodeToString(h[:])

	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, creds, req, bodyHash, botAWSService, botAWSRegion, time.Now()); err != nil {
		return nil, fmt.Errorf("sign request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		if json.Unmarshal(body, &errResp) == nil {
			if msg, ok := errResp["error"].(string); ok {
				return nil, fmt.Errorf("%s", msg)
			}
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if len(body) > 0 {
		_ = json.Unmarshal(body, &result)
	}
	if result == nil {
		result = map[string]interface{}{}
	}
	return result, nil
}
