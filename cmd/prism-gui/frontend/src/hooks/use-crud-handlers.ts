import React from 'react';
import { toast } from 'sonner';
import { SafePrismAPI } from '../lib/api';
import { getTemplateName, getTemplateSlug } from '../lib/template-utils';
import type { LaunchConfig } from '../modals/LaunchModal';
import type { Template } from '../lib/types';

interface UseCrudHandlersOptions {
  api: SafePrismAPI;
  setState: React.Dispatch<React.SetStateAction<any>>;
  selectedTemplate: Template | null;
  handleModalDismiss: () => void;
  loadApplicationData: () => Promise<void>;
  usersVersionRef: React.MutableRefObject<number>;
  setProjectModalVisible: React.Dispatch<React.SetStateAction<boolean>>;
  setUserModalVisible: React.Dispatch<React.SetStateAction<boolean>>;
  setSendInvitationModalVisible: React.Dispatch<React.SetStateAction<boolean>>;
  setRedeemTokenModalVisible: React.Dispatch<React.SetStateAction<boolean>>;
  selectedProjectForInvitation: string;
}

export function useCrudHandlers(options: UseCrudHandlersOptions) {
  const {
    api,
    setState,
    selectedTemplate,
    handleModalDismiss,
    loadApplicationData,
    usersVersionRef,
    setProjectModalVisible,
    setUserModalVisible,
    setSendInvitationModalVisible,
    setRedeemTokenModalVisible,
    selectedProjectForInvitation,
  } = options;

  // Safe instance launch
  const handleLaunchInstance = async (config: LaunchConfig) => {
    if (!selectedTemplate || !config.name.trim()) {
      return;
    }

    // Capture inputs before closing modal
    const templateSlug = getTemplateSlug(selectedTemplate);
    const templateName = getTemplateName(selectedTemplate);
    const instanceName = config.name;
    const instanceSize = config.size;
    const isDryRun = config.dryRun || false;

    // Close modal IMMEDIATELY
    handleModalDismiss();

    // Show progress notification via Sonner toast (Flashbar was removed in Phase 2)
    const toastId = toast.loading(`Launching ${instanceName}... This may take a few minutes.`, {
      description: 'Launching Workspace'
    });

    // Fire-and-forget
    try {
      const result = await api.launchInstance(templateSlug, instanceName, instanceSize, isDryRun, {
        dnsName: config.dnsName,
        ttl: config.ttl,
        spot: config.spot,
        spotMaxPrice: config.spotMaxPrice,
        efa: config.efa,
        placementGroup: config.placementGroup,
        onComplete: config.onComplete,
        completionFile: config.completionFile,
        completionDelay: config.completionDelay,
      });
      // HTTP 202 approval pending (#495)
      if (result && (result as any).approval_pending) {
        const approvalId = (result as any).approval_request_id || 'unknown';
        toast.info(`Launch of ${instanceName} requires PI approval. Request created: ${approvalId}. Check the Approvals panel.`, {
          id: toastId
        });
        // Refresh pending approvals count
        api.listAllApprovals('pending').then(approvals =>
          setState((prev: any) => ({ ...prev, pendingApprovalsCount: approvals.length }))
        ).catch(() => {});
        return;
      }
      toast.success(`Workspace Launched`, {
        id: toastId,
        description: `Successfully launched ${instanceName} using ${templateName}`
      });
      // Reload data in background (don't block the success notification)
      setTimeout(loadApplicationData, 1000);
    } catch (error) {
      toast.error(`Launch Failed`, {
        id: toastId,
        description: `Failed to launch ${instanceName}: ${error instanceof Error ? error.message : 'Unknown error'}`
      });
    }
  };

  // Handle Create Project
  const handleCreateProject = async (data: { name: string; description: string; budget: string }) => {
    // Validate
    if (!data.name.trim()) {
      throw new Error('Project name is required');
    }

    // Call API to create project - send budget via budget.total_budget (backend format)
    const budgetPayload = data.budget ? { budget: { total_budget: parseFloat(data.budget) } } : {};
    const createdProject = await api.createProject({
      name: data.name.trim(),
      description: data.description.trim(),
      ...budgetPayload
    });

    // Map backend response (types.Project with budget.total_budget) to frontend
    // ProjectSummary format (with budget_status.total_budget) for the optimistic update
    const rawProject = createdProject as any;
    const projectForState = {
      ...createdProject,
      budget_status: rawProject.budget ? {
        total_budget: rawProject.budget.total_budget,
        spent_amount: rawProject.budget.spent_amount || 0,
        spent_percentage: 0,
        alert_count: 0,
      } : undefined
    };

    // Optimistic UI update: add project directly to state without re-fetching
    // Prepend new project so it appears at top of list (page 1) - fixes Issue #457
    setState((prev: any) => ({
      ...prev,
      projects: [projectForState, ...prev.projects],
    }));

    toast.success('Project Created', { description: `Project "${data.name}" created successfully` });
    setProjectModalVisible(false);
    // Refresh data from backend to get accurate budget status (e.g. test-mode mock spend)
    setTimeout(loadApplicationData, 500);
  };

  // Handle Create User
  const handleCreateUser = async (data: { username: string; email: string; fullName: string }) => {
    // Validate
    if (!data.username.trim()) {
      throw new Error('Username is required');
    }

    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (data.email && !emailRegex.test(data.email)) {
      throw new Error('Please enter a valid email address');
    }

    try {
      // Call API to create user - it returns the created user object
      const newUser = await api.createUser({
        username: data.username.trim(),
        email: data.email.trim(),
        display_name: data.fullName.trim()
      });

      // Increment users version to mark data as fresh
      // This prevents stale API responses from overwriting our optimistic update
      usersVersionRef.current++;

      // Optimistic UI update: Add the new user directly to state
      // This eliminates race conditions from concurrent getUsers() calls
      setState((prev: any) => ({
        ...prev,
        users: [...prev.users, newUser], // Add new user to existing list
      }));

      toast.success('User Created', { description: `User "${data.username}" created successfully` });
      setUserModalVisible(false);
    } catch (error: any) {
      // Check for duplicate error - backend returns HTTP 409
      // Error format: "HTTP 409: Conflict" or error.response.status === 409
      const is409 = error.response?.status === 409 ||
                   (error.message && error.message.includes('HTTP 409'));

      if (is409) {
        throw new Error('A user with this username already exists');
      } else {
        throw error;
      }
    }
  };

  // SSH Key generation handler
  const handleGenerateSSHKey = async (username: string): Promise<any> => {
    try {
      const response = await api.generateSSHKey(username);

      // Refresh users list to update SSH key status
      const users = await api.getUsers();

      // Increment users version to mark this data as fresh
      // This prevents stale data from overwriting our updated list
      usersVersionRef.current++;

      // Update users state
      setState((prev: any) => ({ ...prev, users }));
      toast.success('SSH Key Generated', {
        description: `SSH key pair generated for "${username}". Download the private key before closing.`
      });

      return response;
    } catch (error: any) {
      throw new Error(error.message || 'Failed to generate SSH key');
    }
  };

  // Individual Invitation Handlers
  const handleSendInvitation = async (data: { email: string; role: 'viewer' | 'member' | 'admin'; message: string }) => {
    // Validate
    if (!data.email.trim()) {
      throw new Error('Email address is required');
    }

    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(data.email)) {
      throw new Error('Please enter a valid email address');
    }

    if (!selectedProjectForInvitation) {
      throw new Error('Please select a project');
    }

    try {
      await api.sendInvitation(selectedProjectForInvitation, data.email.trim(), data.role, data.message.trim() || undefined);

      toast.success('Invitation Sent', { description: `Invitation sent to ${data.email}` });
      setSendInvitationModalVisible(false);
    } catch (error: any) {
      throw new Error(`Failed to send invitation: ${error.message || 'Unknown error'}`);
    }
  };

  const handleRedeemToken = async (token: string) => {
    if (!token.trim()) {
      throw new Error('Invitation token is required');
    }

    try {
      const invitationData = await api.getInvitationByToken(token.trim());

      // Show confirmation with invitation details
      const confirmed = confirm(`Accept invitation to project "${invitationData.project_name}" as ${invitationData.role}?`);

      if (confirmed) {
        await api.acceptInvitation(token.trim());

        toast.success('Token Redeemed', { description: `Successfully joined project "${invitationData.project_name}"` });
        setRedeemTokenModalVisible(false);
      }
    } catch (error: any) {
      throw new Error(`Failed to redeem token: ${error.message || 'Invalid or expired token'}`);
    }
  };

  return {
    handleLaunchInstance,
    handleCreateProject,
    handleCreateUser,
    handleGenerateSSHKey,
    handleSendInvitation,
    handleRedeemToken,
  };
}
