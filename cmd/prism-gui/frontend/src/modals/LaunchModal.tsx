import { useState, useEffect } from 'react'
import {
  Modal,
  Box,
  SpaceBetween,
  Button,
  Form,
  FormField,
  Input,
  Select,
  Alert,
  Checkbox,
} from '../lib/cloudscape-shim'
import { Template } from '../lib/types'
import { getTemplateName, getTemplateDescription } from '../lib/template-utils'

// onCompleteLabel maps an on-complete action value to its dropdown label.
function onCompleteLabel(v?: string): string {
  switch (v) {
    case 'terminate':
      return 'Terminate'
    case 'stop':
      return 'Stop'
    case 'hibernate':
      return 'Hibernate'
    default:
      return 'None (stay running)'
  }
}

// CompletionOptions renders the completion-signaling controls (Phase 3b): spored
// auto terminate/stop/hibernate the workspace when the completion file appears.
// Extracted from LaunchModal to keep that component's complexity in check.
function CompletionOptions({ config, setConfig }: {
  config: LaunchConfig
  setConfig: React.Dispatch<React.SetStateAction<LaunchConfig>>
}) {
  return (
    <>
      <FormField
        label="On completion"
        description="Auto-terminate/stop/hibernate the workspace when the job signals completion (writes the completion file)."
      >
        <Select
          selectedOption={{ value: config.onComplete ?? '', label: onCompleteLabel(config.onComplete) }}
          onChange={({ detail }) => setConfig(prev => ({ ...prev, onComplete: detail.selectedOption.value || undefined }))}
          options={[
            { value: '', label: 'None (stay running)' },
            { value: 'terminate', label: 'Terminate' },
            { value: 'stop', label: 'Stop' },
            { value: 'hibernate', label: 'Hibernate' },
          ]}
          data-testid="launch-on-complete"
        />
      </FormField>

      <FormField
        label="Completion file"
        description="File spored watches to trigger the on-completion action (default /tmp/SPAWN_COMPLETE). Requires an on-completion action."
      >
        <Input
          type="text"
          value={config.completionFile ?? ''}
          onChange={({ detail }) => setConfig(prev => ({ ...prev, completionFile: detail.value }))}
          placeholder="/tmp/SPAWN_COMPLETE"
          disabled={!config.onComplete}
          data-testid="launch-completion-file"
        />
      </FormField>

      <FormField
        label="Completion delay"
        description="Grace period before the on-completion action (e.g. 30s). Requires an on-completion action."
      >
        <Input
          type="text"
          value={config.completionDelay ?? ''}
          onChange={({ detail }) => setConfig(prev => ({ ...prev, completionDelay: detail.value }))}
          placeholder="30s"
          disabled={!config.onComplete}
          data-testid="launch-completion-delay"
        />
      </FormField>
    </>
  )
}

export interface LaunchConfig {
  name: string
  size: string
  spot: boolean
  spotMaxPrice?: string
  efa?: boolean
  placementGroup?: string
  hibernation: boolean
  dryRun: boolean
  dnsName?: string
  ttl?: string
  onComplete?: string
  completionFile?: string
  completionDelay?: string
}

export interface LaunchModalProps {
  visible: boolean
  selectedTemplate: Template | null
  onDismiss: () => void
  onLaunch: (config: LaunchConfig) => void
}

export function LaunchModal({ visible, selectedTemplate, onDismiss, onLaunch }: LaunchModalProps) {
  const [launchConfig, setLaunchConfig] = useState<LaunchConfig>({
    name: '',
    size: 'M',
    spot: false,
    hibernation: false,
    dryRun: false,
  })

  // Reset config when modal opens
  useEffect(() => {
    if (visible) {
      setLaunchConfig({
        name: '',
        size: 'M',
        spot: false,
        hibernation: false,
        dryRun: false,
      })
    }
  }, [visible])

  return (
    <Modal
      onDismiss={onDismiss}
      visible={visible}
      header={`Launch ${selectedTemplate ? getTemplateName(selectedTemplate) : 'Research Environment'}`}
      size="medium"
      footer={
        <Box float="right">
          <SpaceBetween direction="horizontal" size="xs">
            <Button variant="link" onClick={onDismiss}>
              Cancel
            </Button>
            <Button
              variant="primary"
              disabled={!launchConfig.name.trim()}
              onClick={() => onLaunch(launchConfig)}
            >
              Launch Workspace
            </Button>
          </SpaceBetween>
        </Box>
      }
    >
      <Form>
        <SpaceBetween size="m">
          <FormField
            label="Workspace name"
            description="Choose a descriptive name for your research project"
            errorText={!launchConfig.name.trim() ? "Workspace name is required" : ""}
          >
            <Input
              value={launchConfig.name}
              onChange={({ detail }) => setLaunchConfig(prev => ({ ...prev, name: detail.value }))}
              placeholder="my-research-project"
            />
          </FormField>

          <FormField label="Workspace size" description="Choose the right size for your workload">
            <Select
              selectedOption={{ label: "Medium (M) - Recommended", value: "M" }}
              onChange={({ detail }) => setLaunchConfig(prev => ({ ...prev, size: detail.selectedOption.value || 'M' }))}
              options={[
                { label: "Small (S) - Light workloads", value: "S" },
                { label: "Medium (M) - Recommended", value: "M" },
                { label: "Large (L) - Heavy compute", value: "L" },
                { label: "Extra Large (XL) - Maximum performance", value: "XL" }
              ]}
              data-testid="instance-size-select"
            />
          </FormField>

          <FormField label="DNS Name" description="Hostname for this workspace (auto-generated from name if empty)">
            <Input
              value={launchConfig.dnsName ?? ''}
              onChange={({ detail }) => setLaunchConfig(prev => ({ ...prev, dnsName: detail.value }))}
              placeholder={launchConfig.name ? launchConfig.name.toLowerCase().replace(/[^a-z0-9-]/g, '-').replace(/^-+|-+$/g, '') : 'my-workspace'}
            />
          </FormField>

          <FormField label="Time Limit" description="Auto-stop after this duration (e.g., 8h, 24h). Leave empty for no limit.">
            <Input
              value={launchConfig.ttl ?? ''}
              onChange={({ detail }) => setLaunchConfig(prev => ({ ...prev, ttl: detail.value }))}
              placeholder="No limit"
            />
          </FormField>

          {selectedTemplate && (
            <Alert type="info">
              <Box>
                <Box variant="strong">Template: {getTemplateName(selectedTemplate)}</Box>
                <Box>Description: {getTemplateDescription(selectedTemplate)}</Box>
                {selectedTemplate.package_manager && (
                  <Box>Package Manager: {selectedTemplate.package_manager}</Box>
                )}
                {selectedTemplate.complexity && (
                  <Box>Complexity: {selectedTemplate.complexity}</Box>
                )}
              </Box>
            </Alert>
          )}

          <FormField
            label="Instance Options"
            description="Configure advanced instance settings"
          >
            <SpaceBetween size="s">
              <Checkbox
                checked={launchConfig.spot || false}
                onChange={({ detail }) => setLaunchConfig(prev => ({ ...prev, spot: detail.checked }))}
              >
                Spot instance - use lower-cost spot pricing
              </Checkbox>
              <Checkbox
                checked={launchConfig.efa || false}
                onChange={({ detail }) => setLaunchConfig(prev => ({ ...prev, efa: detail.checked }))}
              >
                EFA - attach an Elastic Fabric Adapter (MPI / tightly-coupled HPC; requires an EFA-capable instance type)
              </Checkbox>
              <Checkbox
                checked={launchConfig.hibernation || false}
                onChange={({ detail }) => setLaunchConfig(prev => ({ ...prev, hibernation: detail.checked }))}
              >
                Hibernation - enable instance hibernation support
              </Checkbox>
            </SpaceBetween>
          </FormField>

          <FormField
            label="Spot max price"
            description="Optional maximum spot price in $/hr (e.g. 0.50). Leave empty for the on-demand cap. Requires the Spot instance option."
          >
            <Input
              type="text"
              value={launchConfig.spotMaxPrice ?? ''}
              onChange={({ detail }) => setLaunchConfig(prev => ({ ...prev, spotMaxPrice: detail.value }))}
              placeholder="on-demand cap"
              disabled={!launchConfig.spot}
              data-testid="launch-spot-max-price"
            />
          </FormField>

          <FormField
            label="Placement group"
            description="Optional cluster placement group name for tightly-coupled / MPI workloads."
          >
            <Input
              type="text"
              value={launchConfig.placementGroup ?? ''}
              onChange={({ detail }) => setLaunchConfig(prev => ({ ...prev, placementGroup: detail.value }))}
              placeholder="none"
              data-testid="launch-placement-group"
            />
          </FormField>

          <CompletionOptions config={launchConfig} setConfig={setLaunchConfig} />

          <FormField
            label="Validation"
            description="Test your configuration without actually launching resources"
          >
            <Checkbox
              checked={launchConfig.dryRun || false}
              onChange={({ detail }) => setLaunchConfig(prev => ({ ...prev, dryRun: detail.checked }))}
            >
              Dry run mode - validate without creating resources
            </Checkbox>
          </FormField>
        </SpaceBetween>
      </Form>
    </Modal>
  )
}
