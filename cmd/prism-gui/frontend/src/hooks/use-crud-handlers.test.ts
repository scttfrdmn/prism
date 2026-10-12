import { describe, it, expect, vi, beforeEach } from 'vitest';
import { toast } from 'sonner';
import { useCrudHandlers } from './use-crud-handlers';

vi.mock('sonner', () => ({
  toast: {
    loading: vi.fn(() => 'toast-1'),
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  },
}));

// handleLaunchInstance toasts (#704): a dry run validates the configuration and launches nothing,
// so none of its toasts may claim a launch.
function setup(launchInstance: ReturnType<typeof vi.fn>) {
  const loadApplicationData = vi.fn(() => Promise.resolve());
  const handlers = useCrudHandlers({
    api: { launchInstance } as any,
    setState: vi.fn(),
    selectedTemplate: { Name: 'Python ML', Slug: 'python-ml' } as any,
    handleModalDismiss: vi.fn(),
    loadApplicationData,
    usersVersionRef: { current: 0 },
    setProjectModalVisible: vi.fn(),
    setUserModalVisible: vi.fn(),
    setSendInvitationModalVisible: vi.fn(),
    setRedeemTokenModalVisible: vi.fn(),
    selectedProjectForInvitation: '',
  });
  return { handlers, loadApplicationData };
}

const config = (dryRun: boolean) => ({ name: 'my-project', size: 'M', dryRun }) as any;

describe('handleLaunchInstance', () => {
  beforeEach(() => vi.clearAllMocks());

  it('dry run: reports validation with the daemon message, not a launch', async () => {
    const launch = vi.fn().mockResolvedValue({ message: 'Dry run: configuration for my-project is valid; nothing was launched' });
    const { handlers, loadApplicationData } = setup(launch);

    await handlers.handleLaunchInstance(config(true));

    expect(launch).toHaveBeenCalledWith('python-ml', 'my-project', 'M', true, expect.any(Object));
    expect(vi.mocked(toast.loading).mock.calls[0][0]).toContain('dry run');
    expect(toast.success).toHaveBeenCalledWith('Dry Run Passed', {
      id: 'toast-1',
      description: 'Dry run: configuration for my-project is valid; nothing was launched',
    });
    const allText = JSON.stringify([...vi.mocked(toast.loading).mock.calls, ...vi.mocked(toast.success).mock.calls]);
    expect(allText).not.toMatch(/launched successfully|Workspace Launched|Successfully launched/);
    expect(loadApplicationData).not.toHaveBeenCalled();
  });

  it('dry run without a daemon message falls back to a truthful default', async () => {
    const { handlers } = setup(vi.fn().mockResolvedValue({}));
    await handlers.handleLaunchInstance(config(true));
    expect(vi.mocked(toast.success).mock.calls[0][1]).toMatchObject({ description: expect.stringContaining('Nothing was launched') });
  });

  it('dry run failure says the configuration did not validate', async () => {
    const { handlers } = setup(vi.fn().mockRejectedValue(new Error('unknown template')));
    await handlers.handleLaunchInstance(config(true));
    expect(toast.error).toHaveBeenCalledWith('Dry Run Failed', {
      id: 'toast-1',
      description: 'Configuration for my-project did not validate: unknown template',
    });
  });

  it('real launch is unchanged', async () => {
    const { handlers } = setup(vi.fn().mockResolvedValue({ message: 'Instance my-project launched successfully' }));
    await handlers.handleLaunchInstance(config(false));
    expect(vi.mocked(toast.loading).mock.calls[0][0]).toBe('Launching my-project... This may take a few minutes.');
    expect(toast.success).toHaveBeenCalledWith('Workspace Launched', expect.objectContaining({ id: 'toast-1' }));
  });

  it('real launch failure is unchanged', async () => {
    const { handlers } = setup(vi.fn().mockRejectedValue(new Error('quota exceeded')));
    await handlers.handleLaunchInstance(config(false));
    expect(toast.error).toHaveBeenCalledWith('Launch Failed', {
      id: 'toast-1',
      description: 'Failed to launch my-project: quota exceeded',
    });
  });
});
