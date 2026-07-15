import { describe, it, expect, beforeEach, vi } from 'vitest';
import { SafePrismAPI } from './api';

// Verifies the GUI launch path threads the optional launch capabilities onto the
// wire body with the snake_case keys the daemon (pkg/types.LaunchRequest) expects.
describe('SafePrismAPI.launchInstance wire body', () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      statusText: 'OK',
      headers: { get: () => null },
      json: async () => ({ id: 'i-123', name: 'ws' }),
    });
    vi.stubGlobal('fetch', fetchMock);
  });

  function lastBody() {
    const call = fetchMock.mock.calls.find(([url]) => String(url).includes('/api/v1/instances'));
    expect(call, 'expected a POST to /api/v1/instances').toBeTruthy();
    return JSON.parse((call![1] as RequestInit).body as string);
  }

  it('sends spot, spot_max_price, efa, and placement_group when set', async () => {
    const api = new SafePrismAPI();
    await api.launchInstance('python-ml', 'ws', 'M', false, {
      spot: true,
      spotMaxPrice: '0.50',
      efa: true,
      placementGroup: 'cluster-1',
    });

    const body = lastBody();
    expect(body.spot).toBe(true);
    expect(body.spot_max_price).toBe('0.50');
    expect(body.efa).toBe(true);
    expect(body.placement_group).toBe('cluster-1');
  });

  it('sends on_complete, completion_file, and completion_delay when set', async () => {
    const api = new SafePrismAPI();
    await api.launchInstance('python-ml', 'ws', 'M', false, {
      onComplete: 'terminate',
      completionFile: '/tmp/done',
      completionDelay: '1m',
    });

    const body = lastBody();
    expect(body.on_complete).toBe('terminate');
    expect(body.completion_file).toBe('/tmp/done');
    expect(body.completion_delay).toBe('1m');
  });

  it('omits the optional keys when unset (opt-in, matches Go omitempty)', async () => {
    const api = new SafePrismAPI();
    await api.launchInstance('python-ml', 'ws', 'M');

    const body = lastBody();
    expect(body.spot).toBeUndefined();
    expect(body.spot_max_price).toBeUndefined();
    expect(body.efa).toBeUndefined();
    expect(body.placement_group).toBeUndefined();
    expect(body.on_complete).toBeUndefined();
    expect(body.completion_file).toBeUndefined();
    expect(body.completion_delay).toBeUndefined();
    // core fields still present
    expect(body.template).toBe('python-ml');
    expect(body.name).toBe('ws');
  });
});
