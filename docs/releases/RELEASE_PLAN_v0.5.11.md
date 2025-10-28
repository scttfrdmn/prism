# Prism v0.5.11 Release Plan: User Invitation & Role Systems

**Release Date**: Target March 14, 2026
**Focus**: Enable project collaboration through invitation workflows and enhanced role-based permissions

## 🎯 Release Goals

### Primary Objective
Implement a complete user invitation system with role-based permissions, enabling project owners to easily onboard collaborators with appropriate access levels.

### Success Metrics
- Lab onboarding: Invite new member → accepted and active < 5 minutes
- Class setup: Bulk invite 30 students < 10 minutes
- Role clarity: Zero "permission denied" support tickets after first week
- Cross-institutional: External collaborator can join project seamlessly

---

## 📦 Features & Implementation

### 1. Email-Based Invitation System
**Priority**: P0 (Core feature)
**Effort**: Large (4-5 days)
**Impact**: Critical (Enables collaboration)

**Invitation Workflow**:
```
1. Project Owner → Send Invitation (email + role)
2. System → Generate invitation token (expires in 7 days)
3. System → Send invitation email
4. Invitee → Click link in email
5. Invitee → Accept/Decline invitation
6. System → Add user to project with assigned role
7. Invitee → Receives confirmation email
```

**Data Model**:
```go
type Invitation struct {
    ID            string
    ProjectID     string
    Email         string          // Invitee email
    Role          ProjectRole     // Role to be assigned
    Token         string          // Secure random token
    InvitedBy     string          // User ID of inviter
    InvitedAt     time.Time
    ExpiresAt     time.Time       // Default: 7 days
    Status        InvitationStatus // pending, accepted, declined, expired
    AcceptedAt    *time.Time
    DeclinedAt    *time.Time
    DeclineReason string
    ResendCount   int             // Track number of resends
    LastResent    *time.Time
}

type InvitationStatus string
const (
    InvitationPending  InvitationStatus = "pending"
    InvitationAccepted InvitationStatus = "accepted"
    InvitationDeclined InvitationStatus = "declined"
    InvitationExpired  InvitationStatus = "expired"
    InvitationRevoked  InvitationStatus = "revoked"
)
```

**Database Schema**:
```sql
CREATE TABLE invitations (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    token TEXT NOT NULL UNIQUE,
    invited_by TEXT NOT NULL,
    invited_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    accepted_at TIMESTAMP,
    declined_at TIMESTAMP,
    decline_reason TEXT,
    resend_count INTEGER NOT NULL DEFAULT 0,
    last_resent TIMESTAMP
);

CREATE INDEX idx_invitations_project ON invitations(project_id);
CREATE INDEX idx_invitations_email ON invitations(email);
CREATE INDEX idx_invitations_token ON invitations(token);
CREATE INDEX idx_invitations_status ON invitations(status);

-- Ensure one pending invitation per email per project
CREATE UNIQUE INDEX idx_invitations_unique_pending
    ON invitations(project_id, email)
    WHERE status = 'pending';
```

**API Endpoints**:
```
POST   /api/v1/projects/{id}/invitations          # Send invitation
GET    /api/v1/projects/{id}/invitations          # List invitations for project
GET    /api/v1/invitations/my                     # List user's received invitations
GET    /api/v1/invitations/{token}                # Get invitation by token (public)
POST   /api/v1/invitations/{token}/accept         # Accept invitation
POST   /api/v1/invitations/{token}/decline        # Decline invitation
POST   /api/v1/invitations/{id}/resend            # Resend invitation email
DELETE /api/v1/invitations/{id}                   # Revoke invitation
```

**Email Templates**:
```html
<!-- Invitation Email -->
Subject: You've been invited to join "{project.Name}" on Prism

Hi there,

{inviter.Name} ({inviter.Email}) has invited you to collaborate on the
"{project.Name}" project on Prism as a {role}.

Project Description:
{project.Description}

To accept this invitation, click the link below:
{acceptURL}

To decline, click here:
{declineURL}

This invitation will expire in 7 days.

Questions? Reply to this email or contact {inviter.Email}.

Best,
The Prism Team
```

```html
<!-- Acceptance Confirmation -->
Subject: Welcome to "{project.Name}"!

Hi {user.Name},

You've successfully joined the "{project.Name}" project as a {role}.

You can now:
- Launch workspaces under this project
- Access shared EFS storage
- Collaborate with {memberCount} team members
- View project budget and spending

Get started: {projectURL}

Best,
The Prism Team
```

**Implementation Tasks**:
- [ ] Create invitation data model
- [ ] Implement token generation (crypto.rand, 32 bytes)
- [ ] Add invitation CRUD API endpoints
- [ ] Integrate email sending (SMTP or SendGrid)
- [ ] Create email templates
- [ ] Implement accept/decline logic
- [ ] Add invitation expiration checking
- [ ] Handle duplicate invitation prevention
- [ ] Add invitation audit trail

---

### 2. Role System Enhancement
**Priority**: P0 (Security & Permissions)
**Effort**: Medium (3-4 days)
**Impact**: Critical (Access control)

**Current Role System** (from Phase 4):
```go
type ProjectRole string
const (
    RoleOwner  ProjectRole = "owner"   // Full control
    RoleAdmin  ProjectRole = "admin"   // Manage members, budgets
    RoleMember ProjectRole = "member"  // Launch workspaces
    RoleViewer ProjectRole = "viewer"  // Read-only access
)
```

**Enhanced Permission Matrix**:
```go
type Permission string
const (
    // Project Management
    PermProjectEdit          Permission = "project:edit"
    PermProjectDelete        Permission = "project:delete"
    PermProjectViewMembers   Permission = "project:view_members"
    PermProjectManageMembers Permission = "project:manage_members"

    // Budget Management
    PermBudgetView           Permission = "budget:view"
    PermBudgetEdit           Permission = "budget:edit"
    PermBudgetAllocate       Permission = "budget:allocate"

    // Workspace Operations
    PermWorkspaceLaunch      Permission = "workspace:launch"
    PermWorkspaceView        Permission = "workspace:view"
    PermWorkspaceConnect     Permission = "workspace:connect"
    PermWorkspaceControl     Permission = "workspace:control"  // stop, hibernate, terminate
    PermWorkspaceTerminate   Permission = "workspace:terminate"

    // Storage Management
    PermStorageView          Permission = "storage:view"
    PermStorageCreate        Permission = "storage:create"
    PermStorageAttach        Permission = "storage:attach"
    PermStorageDelete        Permission = "storage:delete"

    // Invitations
    PermInvitationSend       Permission = "invitation:send"
    PermInvitationView       Permission = "invitation:view"
    PermInvitationRevoke     Permission = "invitation:revoke"
)

// Role Permission Mapping
var RolePermissions = map[ProjectRole][]Permission{
    RoleOwner: {
        // All permissions
        PermProjectEdit, PermProjectDelete, PermProjectViewMembers, PermProjectManageMembers,
        PermBudgetView, PermBudgetEdit, PermBudgetAllocate,
        PermWorkspaceLaunch, PermWorkspaceView, PermWorkspaceConnect, PermWorkspaceControl, PermWorkspaceTerminate,
        PermStorageView, PermStorageCreate, PermStorageAttach, PermStorageDelete,
        PermInvitationSend, PermInvitationView, PermInvitationRevoke,
    },
    RoleAdmin: {
        // Project & member management, no delete
        PermProjectEdit, PermProjectViewMembers, PermProjectManageMembers,
        PermBudgetView, PermBudgetEdit,
        PermWorkspaceLaunch, PermWorkspaceView, PermWorkspaceConnect, PermWorkspaceControl,
        PermStorageView, PermStorageCreate, PermStorageAttach,
        PermInvitationSend, PermInvitationView, PermInvitationRevoke,
    },
    RoleMember: {
        // Launch workspaces, manage own resources
        PermProjectViewMembers,
        PermBudgetView,
        PermWorkspaceLaunch, PermWorkspaceView, PermWorkspaceConnect, PermWorkspaceControl,
        PermStorageView, PermStorageCreate, PermStorageAttach,
        PermInvitationView,
    },
    RoleViewer: {
        // Read-only access
        PermProjectViewMembers,
        PermBudgetView,
        PermWorkspaceView,
        PermStorageView,
        PermInvitationView,
    },
}

// Permission checking
func (p *Project) HasPermission(userID string, perm Permission) bool {
    member := p.GetMember(userID)
    if member == nil {
        return false
    }

    allowedPerms := RolePermissions[member.Role]
    for _, allowed := range allowedPerms {
        if allowed == perm {
            return true
        }
    }
    return false
}
```

**API Middleware** (Permission Enforcement):
```go
// RequireProjectPermission middleware
func RequireProjectPermission(perm Permission) gin.HandlerFunc {
    return func(c *gin.Context) {
        projectID := c.Param("projectID")
        userID := c.GetString("userID")  // From auth token

        project, err := getProject(projectID)
        if err != nil {
            c.JSON(404, gin.H{"error": "Project not found"})
            c.Abort()
            return
        }

        if !project.HasPermission(userID, perm) {
            c.JSON(403, gin.H{"error": "Permission denied"})
            c.Abort()
            return
        }

        c.Set("project", project)
        c.Next()
    }
}

// Usage:
router.POST("/projects/:projectID/workspaces",
    RequireProjectPermission(PermWorkspaceLaunch),
    LaunchWorkspaceHandler)
```

**Implementation Tasks**:
- [ ] Define comprehensive permission set
- [ ] Create permission checking functions
- [ ] Add API middleware for permission enforcement
- [ ] Update all project API handlers with permission checks
- [ ] Add role-based UI element visibility
- [ ] Document permission matrix
- [ ] Add permission checking tests

---

### 3. Invitation Management GUI
**Priority**: P0 (User Experience)
**Effort**: Medium (3-4 days)
**Impact**: Critical (Usability)

**Components**:

#### Project Members Tab (Enhanced)
```typescript
// cmd/prism-gui/frontend/src/pages/ProjectDetail.tsx (Members Tab)
interface MemberListProps {
  project: Project;
  currentUser: User;
}

Features:
- Member table with columns:
  - Name, Email, Role, Joined Date
  - Actions (Edit Role, Remove) - if has permission
- "Invite Member" button → Opens InvitationDialog
- Pending invitations section:
  - Email, Role, Invited By, Expires In
  - Actions: Resend, Revoke
- Role badges with color coding:
  - Owner: Red badge
  - Admin: Orange badge
  - Member: Blue badge
  - Viewer: Gray badge
```

#### Invitation Dialog
```typescript
// cmd/prism-gui/frontend/src/components/InvitationDialog.tsx
interface InvitationDialogProps {
  project: Project;
  onInvite: (email: string, role: ProjectRole) => Promise<void>;
}

Features:
- Email input with validation
- Role selector with descriptions:
  - Owner: "Full control of project"
  - Admin: "Manage members, budgets, workspaces"
  - Member: "Launch workspaces, manage own resources"
  - Viewer: "Read-only access"
- Optional message to invitee
- Bulk invitation (CSV upload for classes)
- Preview permissions for selected role
```

#### My Invitations Page
```typescript
// cmd/prism-gui/frontend/src/pages/MyInvitations.tsx
Features:
- List of pending invitations
- Project details preview
- Inviter information
- Role description
- Accept/Decline buttons
- Decline reason (optional text field)
- Invitation expiration countdown
```

#### Invitation Email Landing Page
```typescript
// Public page (no auth required)
// /accept-invitation?token={token}

Features:
- Project information display
- Role details and permissions
- Inviter details
- Accept button → Creates account or logs in → Joins project
- Decline button → Optional reason → Marks declined
- Expired invitation message
- Already accepted/declined status
```

**Cloudscape Components**:
- `Table` for member and invitation lists
- `Badge` for roles and statuses
- `Modal` for invitation dialog
- `FormField` for email and role selection
- `Select` for role dropdown
- `Button` for actions
- `Alert` for permission explanations

**Implementation Tasks**:
- [ ] Enhance ProjectDetail Members tab
- [ ] Create InvitationDialog component
- [ ] Create MyInvitations page
- [ ] Create public invitation acceptance page
- [ ] Add bulk invitation CSV upload
- [ ] Implement role permission preview
- [ ] Add invitation status notifications
- [ ] Add expiration countdown timers

---

### 4. Bulk Invitation for Classes
**Priority**: P1 (High-impact use case)
**Effort**: Small (1-2 days)
**Impact**: High (Educational adoption)

**Use Case**:
Professor needs to invite 30-50 students to class project with one action.

**CSV Format**:
```csv
email,role,name
alice@university.edu,member,Alice Smith
bob@university.edu,member,Bob Jones
carol@university.edu,member,Carol Lee
...
```

**Bulk Invitation Workflow**:
```typescript
interface BulkInvitationRequest {
  projectId: string;
  invitations: Array<{
    email: string;
    role: ProjectRole;
    name?: string;  // Optional display name
  }>;
  message?: string;  // Optional custom message
}

// API:
POST /api/v1/projects/{id}/invitations/bulk
{
  "invitations": [
    { "email": "alice@edu.edu", "role": "member" },
    { "email": "bob@edu.edu", "role": "member" },
    ...
  ],
  "message": "Welcome to CS499 Spring 2026!"
}

// Response:
{
  "success": 48,
  "failed": 2,
  "errors": [
    { "email": "invalid@", "error": "Invalid email format" },
    { "email": "existing@edu.edu", "error": "Already a member" }
  ]
}
```

**GUI Features**:
- CSV file upload
- CSV validation preview
- Duplicate detection (already members)
- Email validation
- Role assignment (bulk or per-row)
- Progress bar during sending
- Summary report (success/failed)

**Implementation Tasks**:
- [ ] Add bulk invitation API endpoint
- [ ] Implement CSV parsing and validation
- [ ] Create BulkInvitationDialog component
- [ ] Add progress tracking
- [ ] Generate detailed result report
- [ ] Add CSV template download
- [ ] Handle rate limiting for large batches

---

### 5. Integration with Research User System
**Priority**: P1 (Existing feature integration)
**Effort**: Small (1-2 days)
**Impact**: Medium (Seamless workflow)

**Workflow**:
1. User accepts project invitation
2. System checks if user has research user account
3. If not → Create research user automatically
4. Provision research user on project workspaces
5. User can immediately SSH to workspaces

**Auto-Provisioning**:
```go
// After invitation acceptance
func OnInvitationAccepted(invitation *Invitation) error {
    // 1. Add user to project
    err := project.AddMember(invitation.Email, invitation.Role)
    if err != nil {
        return err
    }

    // 2. Check for research user
    user, err := researchUserManager.GetByEmail(invitation.Email)
    if err == ErrUserNotFound {
        // Create research user automatically
        user, err = researchUserManager.Create(ResearchUserRequest{
            Username: generateUsername(invitation.Email),
            Email:    invitation.Email,
            Profile:  project.Profile,
        })
        if err != nil {
            return err
        }
    }

    // 3. Provision on existing project workspaces
    workspaces := project.GetWorkspaces()
    for _, workspace := range workspaces {
        if workspace.State == "running" {
            err = researchUserProvisioner.Provision(workspace, user)
            if err != nil {
                log.Errorf("Failed to provision user on workspace %s: %v",
                    workspace.ID, err)
                // Non-blocking - user will be provisioned on next launch
            }
        }
    }

    return nil
}
```

**Implementation Tasks**:
- [ ] Add auto-provisioning after invitation acceptance
- [ ] Update workspace launch to provision all project members
- [ ] Add research user status to project member display
- [ ] Handle research user creation errors gracefully
- [ ] Add retroactive provisioning for running workspaces
- [ ] Document research user integration

---

### 6. AWS Quota Validation for Invitations
**Priority**: P0 (Prevents class setup failures)
**Effort**: Medium (2-3 days)
**Impact**: Critical (Classroom use case)

**Problem**:
- Professor invites 40 students to CS499
- Each student will launch python-ml template (t3.xlarge = 4 vCPUs)
- Total needed: 40 × 4 = 160 vCPUs
- AWS account quota: 32 vCPUs (default for new accounts)
- Result: 8 students succeed, 32 fail with quota errors → chaos

**Solution**: Pre-flight quota validation during invitation setup

**Workflow**:
```
1. Professor selects project template (python-ml)
2. Professor enters number of students (40)
3. System calculates: 40 × 4 vCPUs = 160 vCPUs needed
4. System checks AWS quotas:
   - Current quota: 32 vCPUs
   - Current usage: 8 vCPUs (2 existing workspaces)
   - Available: 24 vCPUs
   - Needed: 160 vCPUs
   - Shortfall: 136 vCPUs
5. System shows warning with guidance
6. Professor can request quota increase or adjust plan
```

**Implementation**:

#### Quota Calculator
```go
// pkg/aws/quota.go
type QuotaRequirement struct {
    Template      string
    InstanceType  string
    VCPUsPerInstance int
    Count         int
    TotalVCPUs    int
}

func CalculateQuotaNeeds(template Template, count int) QuotaRequirement {
    instanceType := template.DefaultInstanceType
    vcpus := GetInstanceTypeVCPUs(instanceType)

    return QuotaRequirement{
        Template:         template.Name,
        InstanceType:     instanceType,
        VCPUsPerInstance: vcpus,
        Count:            count,
        TotalVCPUs:       vcpus * count,
    }
}

type QuotaStatus struct {
    QuotaName       string  // "L-1216C47A" (On-Demand Standard vCPUs)
    CurrentQuota    int
    CurrentUsage    int
    Available       int
    Needed          int
    Sufficient      bool
    Shortfall       int
    RequestURL      string  // AWS Service Quotas URL
}

func CheckQuota(region string, requirement QuotaRequirement) (*QuotaStatus, error) {
    // Use AWS Service Quotas API
    svc := servicequotas.New(session.New(), aws.NewConfig().WithRegion(region))

    // Get current quota for on-demand vCPUs
    quotaResp, err := svc.GetServiceQuota(&servicequotas.GetServiceQuotaInput{
        ServiceCode: aws.String("ec2"),
        QuotaCode:   aws.String("L-1216C47A"), // On-Demand Standard vCPUs
    })
    if err != nil {
        return nil, err
    }

    currentQuota := int(*quotaResp.Quota.Value)

    // Get current vCPU usage
    currentUsage, err := getCurrentVCPUUsage(region)
    if err != nil {
        return nil, err
    }

    available := currentQuota - currentUsage
    sufficient := available >= requirement.TotalVCPUs
    shortfall := 0
    if !sufficient {
        shortfall = requirement.TotalVCPUs - available
    }

    return &QuotaStatus{
        QuotaName:    "On-Demand Standard vCPUs",
        CurrentQuota: currentQuota,
        CurrentUsage: currentUsage,
        Available:    available,
        Needed:       requirement.TotalVCPUs,
        Sufficient:   sufficient,
        Shortfall:    shortfall,
        RequestURL:   fmt.Sprintf("https://console.aws.amazon.com/servicequotas/home/services/ec2/quotas/L-1216C47A?region=%s", region),
    }, nil
}
```

#### GUI Quota Warning Dialog
```typescript
// cmd/prism-gui/frontend/src/components/QuotaWarningDialog.tsx
interface QuotaWarningProps {
  requirement: QuotaRequirement;
  status: QuotaStatus;
  onRequestIncrease: () => void;
  onProceedAnyway: () => void;
  onCancel: () => void;
}

// Display:
// ⚠️  Insufficient AWS vCPU Quota
//
// You're inviting 40 students to use the "Python ML" template.
// Each workspace requires 4 vCPUs (t3.xlarge).
//
// Total needed:  160 vCPUs
// Current quota: 32 vCPUs
// Currently used: 8 vCPUs (2 workspaces)
// Available:     24 vCPUs
// Shortfall:     136 vCPUs
//
// ⚠️  Only 6 students will be able to launch workspaces with current quota.
//
// Options:
// 1. Request quota increase from AWS (recommended)
//    - Takes 1-2 business days
//    - Usually approved automatically
//    - [Request Increase] button
//
// 2. Choose smaller instance type
//    - t3.medium: 2 vCPUs → 12 students can launch
//    - t3.small: 1 vCPU → 24 students can launch
//
// 3. Proceed anyway (not recommended)
//    - First 6 students succeed, others fail
//    - Will cause confusion and support issues
```

#### CLI Quota Check
```bash
$ prism project create cs499 --template python-ml --students 40

Checking AWS quotas...

⚠️  Insufficient vCPU Quota

Template: Python ML (t3.xlarge, 4 vCPUs each)
Students: 40
Total needed: 160 vCPUs

Current AWS quota: 32 vCPUs
Currently used: 8 vCPUs (2 workspaces)
Available: 24 vCPUs
Shortfall: 136 vCPUs

Only 6 students will be able to launch workspaces.

Recommendations:
1. Request quota increase (recommended):
   aws service-quotas request-service-quota-increase \
     --service-code ec2 \
     --quota-code L-1216C47A \
     --desired-value 192 \
     --region us-west-2

   Or via web: https://console.aws.amazon.com/servicequotas/...

2. Use smaller instance type:
   --template python-ml --instance-type t3.medium  # 2 vCPUs → 12 students
   --template python-ml --instance-type t3.small   # 1 vCPU → 24 students

3. Proceed anyway (not recommended):
   --skip-quota-check

Abort project creation? [Y/n]:
```

#### API Endpoints
```
POST /api/v1/quotas/check          # Check quota for requirement
POST /api/v1/quotas/request        # Request quota increase
GET  /api/v1/quotas/requests/{id}  # Check quota request status
```

**Integration with Invitation Flow**:

1. **Project Creation with Template**:
   - User specifies expected number of members
   - System calculates quota needs based on template
   - Shows warning if insufficient

2. **Bulk Invitation**:
   - Before sending invitations, check quota
   - Warn if some users won't be able to launch
   - Suggest requesting increase or adjusting plan

3. **Invitation Acceptance**:
   - When user accepts invitation and tries to launch
   - If quota insufficient, show helpful error
   - Guide to contact project admin for quota increase

**Implementation Tasks**:
- [ ] Implement quota calculation logic
- [ ] Add AWS Service Quotas API integration
- [ ] Create quota checking during project setup
- [ ] Add quota warning dialogs (CLI + GUI)
- [ ] Add quota increase request workflow
- [ ] Integrate with invitation flow
- [ ] Add helpful error messages when quota exceeded
- [ ] Test with various template/instance types

---

## 📅 Implementation Schedule

### Week 1 (Mar 1-7): Backend & Email System
**Days 1-2**: Invitation data model and API
- Design invitation schema
- Implement CRUD endpoints
- Add token generation and validation

**Days 3-4**: Email integration
- Set up email sending (SMTP/SendGrid)
- Create email templates
- Test invitation sending and acceptance

**Day 5**: Permission system enhancement
- Define permission matrix
- Implement permission checking
- Add API middleware

### Week 2 (Mar 8-14): Frontend & Integration
**Days 1-2**: Invitation GUI
- Enhance ProjectDetail Members tab
- Create InvitationDialog
- Create MyInvitations page

**Day 3**: Quota validation integration
- Add AWS Service Quotas API integration
- Implement quota checking during project setup
- Add quota warning dialogs
- Test with various scenarios

**Day 4**: Bulk invitation & research users
- Implement CSV upload and parsing
- Add BulkInvitationDialog
- Add research user auto-provisioning
- Test with large class roster

**Day 5**: Testing & Polish
- Persona walkthroughs
- Permission testing
- Quota validation testing
- Email deliverability testing
- Bug fixes
- Documentation

---

## 🧪 Testing Strategy

### Backend Testing
- [ ] Invitation CRUD operations
- [ ] Token generation and validation
- [ ] Expiration handling
- [ ] Duplicate invitation prevention
- [ ] Permission checking accuracy
- [ ] Email sending (mock SMTP)
- [ ] Bulk invitation processing
- [ ] Research user auto-provisioning

### Frontend Testing
- [ ] Invitation dialog workflow
- [ ] Member management interface
- [ ] My Invitations page
- [ ] Acceptance landing page
- [ ] Bulk CSV upload
- [ ] Role permission preview
- [ ] Mobile responsiveness

### Security Testing
- [ ] Token security (entropy, uniqueness)
- [ ] Permission bypass attempts
- [ ] Expired invitation handling
- [ ] Email spoofing prevention
- [ ] CSRF protection on acceptance
- [ ] Rate limiting on invitation sending

### Persona Walkthroughs

#### Lab Collaboration (New Member Onboarding)
**Scenario**: PI adds new postdoc to research group

1. PI opens "ML Research" project
2. Navigate to Members tab
3. Click "Invite Member"
4. Enter: jane.doe@university.edu, Role: Member
5. Send invitation
6. Jane receives email within 1 minute
7. Jane clicks accept link
8. Jane creates Prism account (or logs in)
9. Jane automatically added to project
10. Jane's research user created automatically
11. Jane can immediately launch workspaces
12. Jane can SSH to existing shared workspace

**Time**: < 5 minutes from send to active

#### University Class (Bulk Student Invitation)
**Scenario**: Professor sets up CS499 class with 30 students

1. Professor creates "CS499 Spring 2026" project
2. Navigate to Members tab
3. Click "Bulk Invite"
4. Download CSV template
5. Fill CSV with 30 student emails (all role: member)
6. Upload CSV
7. Review validation preview (all valid)
8. Click "Send Invitations"
9. Progress bar shows 30/30 sent
10. Students receive emails
11. Students accept throughout the week
12. Professor monitors pending/accepted status
13. After all accept, launch template workspaces
14. All students automatically provisioned

**Time**: < 10 minutes to send 30 invitations

#### Cross-Institutional Collaboration
**Scenario**: Multi-university research project

1. Lead researcher creates "Climate Modeling Consortium" project
2. Invite collaborators from 3 universities:
   - alice@stanford.edu (Admin)
   - bob@berkeley.edu (Member)
   - carol@mit.edu (Member)
3. External collaborators receive invitations
4. Each accepts and joins project
5. Research users created with consistent UIDs
6. Shared EFS storage accessible to all
7. Each can launch workspaces under project
8. Budget tracked across all institutions

**Time**: < 5 minutes per collaborator

---

## 📚 Documentation Updates

### New Documentation
- [ ] User invitation guide
- [ ] Role and permissions reference
- [ ] Bulk invitation tutorial (for classes)
- [ ] Cross-institutional collaboration guide

### Updated Documentation
- [ ] Project management section
- [ ] Member management documentation
- [ ] Security and access control
- [ ] API reference (invitation endpoints)

### Release Notes
- [ ] New features (invitations, enhanced roles)
- [ ] Permission changes
- [ ] API additions
- [ ] Integration with research user system

---

## 🚀 Release Criteria

### Must Have (Blocking)
- ✅ Email invitation system working
- ✅ Role permission system enforced
- ✅ Invitation management GUI complete
- ✅ Bulk invitation functional
- ✅ Research user auto-provisioning working
- ✅ All persona tests pass
- ✅ Documentation complete

### Nice to Have (Non-Blocking)
- Invitation templates (save/reuse invitation text)
- Calendar integration (add to calendar reminder)
- Invitation analytics (acceptance rate, time to accept)
- Invitation expiration customization

---

## 📊 Success Metrics (Post-Release)

Track for 2 weeks after release:

1. **Invitation Acceptance Rate**
   - Measure: % of sent invitations accepted
   - Target: >85% acceptance rate

2. **Time to First Collaboration**
   - Measure: Time from send to invitee active on workspace
   - Target: <10 minutes (median)

3. **Bulk Invitation Usage**
   - Measure: % of projects using bulk invite
   - Target: >40% of class/group projects

4. **Permission Clarity**
   - Measure: "Permission denied" support tickets
   - Target: <2% of total support volume

5. **Research User Integration**
   - Measure: % of invitations triggering auto-provisioning
   - Target: >95% success rate

---

## 🔗 Related Documents

- ROADMAP.md - Overall project roadmap
- RELEASE_PLAN_v0.5.10.md - Multi-Project Budgets (prerequisite)
- User Guide: Project Collaboration (to be created)
- Security Guide: Access Control (to be updated)

---

**Last Updated**: October 27, 2025
**Status**: 📋 Planned
**Dependencies**: v0.5.10 (Multi-Project Budgets)
