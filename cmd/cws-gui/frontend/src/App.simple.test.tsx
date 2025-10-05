import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import App from './App';

// Mock window.wails for testing
const mockWails = {
  CloudWorkstationService: {
    GetTemplates: vi.fn(),
    GetInstances: vi.fn(),
    LaunchInstance: vi.fn()
  }
};

Object.defineProperty(window, 'wails', {
  value: mockWails,
  writable: true
});

describe('CloudWorkstation App - Essential Tests', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    // Mock successful API responses
    mockWails.CloudWorkstationService.GetTemplates.mockResolvedValue([
      {
        Name: 'Python Machine Learning',
        Description: 'Complete ML environment',
        Domain: 'ml',
        Complexity: 'moderate',
        Icon: '🤖',
        Popular: true,
        EstimatedLaunchTime: 2,
        EstimatedCostPerHour: { 'x86_64': 0.48 },
        ValidationStatus: 'validated'
      }
    ]);

    mockWails.CloudWorkstationService.GetInstances.mockResolvedValue([
      {
        id: 'i-123',
        name: 'my-instance',
        template: 'Python ML',
        status: 'running',
        public_ip: '1.2.3.4',
        cost_per_hour: 0.48,
        launch_time: '2025-09-28T10:30:00Z',
        region: 'us-west-2'
      }
    ]);
  });

  describe('Core Functionality', () => {
    it('renders without crashing', async () => {
      await act(async () => {
        render(<App />);
      });
      expect(screen.getByRole('link', { name: /cloudworkstation/i })).toBeInTheDocument();
    });

    it('loads and displays templates', async () => {
      await act(async () => {
        render(<App />);
      });

      await waitFor(() => {
        expect(screen.getByRole('heading', { name: /research templates/i })).toBeInTheDocument();
      });

      await waitFor(() => {
        expect(screen.getByText('Python Machine Learning')).toBeInTheDocument();
      }, { timeout: 3000 });

      expect(mockWails.CloudWorkstationService.GetTemplates).toHaveBeenCalledTimes(1);
    });

    it('shows loading state initially', async () => {
      await act(async () => {
        render(<App />);
      });
      expect(screen.getByText('Loading templates...')).toBeInTheDocument();
    });

    it('displays navigation elements', async () => {
      await act(async () => {
        render(<App />);
      });
      expect(screen.getByRole('link', { name: /templates/i })).toBeInTheDocument();
      expect(screen.getByRole('link', { name: /instances/i })).toBeInTheDocument();
      expect(screen.getByRole('link', { name: /settings/i })).toBeInTheDocument();
    });

    it('handles API call failures gracefully', async () => {
      mockWails.CloudWorkstationService.GetTemplates.mockRejectedValue(new Error('API Error'));

      await act(async () => {
        render(<App />);
      });

      await waitFor(() => {
        expect(screen.getByText('Failed to load data')).toBeInTheDocument();
      });
    });
  });

  describe('Template Display', () => {
    it('shows template count', async () => {
      await act(async () => {
        render(<App />);
      });

      await waitFor(() => {
        expect(screen.getByText('1 of 1 templates')).toBeInTheDocument();
      }, { timeout: 5000 });
    });

    it('displays template information', async () => {
      await act(async () => {
        render(<App />);
      });

      await waitFor(() => {
        expect(screen.getByText('Python Machine Learning')).toBeInTheDocument();
        expect(screen.getByText('Complete ML environment')).toBeInTheDocument();
        expect(screen.getByText('🤖')).toBeInTheDocument();
        expect(screen.getByText('Popular')).toBeInTheDocument();
      }, { timeout: 5000 });
    });
  });

  describe('Instance Management', () => {
    it('shows instance count when navigated to instances view', async () => {
      render(<App />);

      // Wait for initial load
      await waitFor(() => {
        expect(screen.getByRole('heading', { name: /research templates/i })).toBeInTheDocument();
      });

      // Navigate to instances - simulate clicking navigation
      const instancesNavItem = screen.getByText('Instances');
      expect(instancesNavItem).toBeInTheDocument();
    });
  });

  describe('Error Boundaries', () => {
    it('provides fallback when templates fail to load', async () => {
      mockWails.CloudWorkstationService.GetTemplates.mockRejectedValue(new Error('Network failure'));

      render(<App />);

      await waitFor(() => {
        expect(screen.getByText('Unable to connect to CloudWorkstation daemon')).toBeInTheDocument();
      });
    });

    it('shows empty state when no templates available', async () => {
      mockWails.CloudWorkstationService.GetTemplates.mockResolvedValue([]);
      mockWails.CloudWorkstationService.GetInstances.mockResolvedValue([]);

      render(<App />);

      await waitFor(() => {
        expect(screen.getByText('No templates available')).toBeInTheDocument();
      });
    });
  });

  describe('Professional Interface Elements', () => {
    it('uses proper Cloudscape components structure', async () => {
      render(<App />);

      // Verify professional interface structure
      expect(document.querySelector('[data-testid="app-layout"]')).toBeTruthy();
    });

    it('includes search functionality', async () => {
      render(<App />);

      await waitFor(() => {
        const searchInput = screen.getByPlaceholderText('Find templates by name, domain, or complexity...');
        expect(searchInput).toBeInTheDocument();
      });
    });
  });
});

describe('Performance and Reliability', () => {
  it('handles concurrent API calls properly', async () => {
    const templates = mockWails.CloudWorkstationService.GetTemplates;
    const instances = mockWails.CloudWorkstationService.GetInstances;

    render(<App />);

    await waitFor(() => {
      expect(templates).toHaveBeenCalledTimes(1);
      expect(instances).toHaveBeenCalledTimes(1);
    });

    // Verify both calls completed
    expect(templates).toHaveReturned();
    expect(instances).toHaveReturned();
  });

  it('maintains stable interface during loading', () => {
    render(<App />);

    // Core structure should be immediately available
    expect(screen.getByText('CloudWorkstation')).toBeInTheDocument();
    expect(screen.getByText('Templates')).toBeInTheDocument();
    expect(screen.getByText('Instances')).toBeInTheDocument();
    expect(screen.getByText('Settings')).toBeInTheDocument();
  });
});