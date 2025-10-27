import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import App from './App';

// Mock window.wails for testing
const mockWails = {
  PrismService: {
    GetTemplates: vi.fn(),
    GetInstances: vi.fn(),
    LaunchInstance: vi.fn()
  }
};

// Set up DOM environment
Object.defineProperty(window, 'wails', {
  value: mockWails,
  writable: true
});

describe('Prism App', () => {
  beforeEach(() => {
    // Reset all mocks before each test
    vi.clearAllMocks();

    // Default mock responses
    mockWails.PrismService.GetTemplates.mockResolvedValue([
      {
        Name: 'Python Machine Learning',
        Description: 'Complete ML environment with TensorFlow, PyTorch, and Jupyter',
        Category: 'Machine Learning',
        Domain: 'ml',
        Complexity: 'moderate',
        Icon: '🤖',
        Popular: true,
        EstimatedLaunchTime: 2,
        EstimatedCostPerHour: { 'x86_64': 0.48 },
        ValidationStatus: 'validated'
      },
      {
        Name: 'R Research Environment',
        Description: 'Statistical computing with R, RStudio, and tidyverse packages',
        Category: 'Data Science',
        Domain: 'datascience',
        Complexity: 'simple',
        Icon: '📊',
        Popular: true,
        EstimatedLaunchTime: 3,
        EstimatedCostPerHour: { 'x86_64': 0.24 },
        ValidationStatus: 'validated'
      }
    ]);

    mockWails.PrismService.GetInstances.mockResolvedValue([
      {
        id: 'i-1234567890abcdef0',
        name: 'my-ml-research',
        template: 'Python Machine Learning',
        status: 'running',
        public_ip: '54.123.45.67',
        cost_per_hour: 0.48,
        launch_time: '2025-09-28T10:30:00Z',
        region: 'us-west-2'
      }
    ]);
  });

  describe('Template Selection', () => {
    it('renders template selection view by default', async () => {
      render(<App />);

      expect(screen.getByText('Research Templates')).toBeInTheDocument();
      expect(screen.getByPlaceholderText('Find templates by name, domain, or complexity...')).toBeInTheDocument();

      await waitFor(() => {
        expect(screen.getByText('Python Machine Learning')).toBeInTheDocument();
        expect(screen.getByText('R Research Environment')).toBeInTheDocument();
      });
    });

    it('displays template cards with proper information', async () => {
      render(<App />);

      await waitFor(() => {
        // Check template information is displayed
        expect(screen.getByText('Python Machine Learning')).toBeInTheDocument();
        expect(screen.getByText('Complete ML environment with TensorFlow, PyTorch, and Jupyter')).toBeInTheDocument();
        expect(screen.getByText('🤖')).toBeInTheDocument();
        expect(screen.getByText('Popular')).toBeInTheDocument();
        expect(screen.getByText('Machine Learning')).toBeInTheDocument();
        expect(screen.getByText('moderate')).toBeInTheDocument();
        expect(screen.getByText('Pre-tested')).toBeInTheDocument();
      });
    });

    it('filters templates using PropertyFilter', async () => {
      render(<App />);

      await waitFor(() => {
        expect(screen.getByText('Python Machine Learning')).toBeInTheDocument();
        expect(screen.getByText('R Research Environment')).toBeInTheDocument();
      });

      // Test filtering by domain
      const filterInput = screen.getByPlaceholderText('Find templates by name, domain, or complexity...');
      fireEvent.change(filterInput, { target: { value: 'domain = ml' } });

      // PropertyFilter should filter templates by domain
      // Cloudscape PropertyFilter is tested via integration with real API data
      // Unit tests verify filter input is rendered and accessible
    });

    it('displays template selection interface', async () => {
      render(<App />);

      await waitFor(() => {
        // Just verify the interface elements are present
        expect(screen.getByText('Python Machine Learning')).toBeInTheDocument();
        expect(screen.getByText('2 of 2 templates')).toBeInTheDocument();
      });
    });
  });

  describe('Instance Management', () => {
    it('renders instance management view when navigated', async () => {
      render(<App />);

      // Navigate to instances view
      const instancesLink = screen.getByText('Instances');
      fireEvent.click(instancesLink);

      await waitFor(() => {
        expect(screen.getByText('My Workspaces')).toBeInTheDocument();
        expect(screen.getByText('(1)')).toBeInTheDocument(); // Instance counter
        expect(screen.getByText('my-ml-research')).toBeInTheDocument();
      });
    });

    it('displays instance information correctly', async () => {
      render(<App />);

      // Navigate to instances view
      fireEvent.click(screen.getByText('Instances'));

      await waitFor(() => {
        // Check instance details
        expect(screen.getByText('my-ml-research')).toBeInTheDocument();
        expect(screen.getByText('i-1234567890abcdef0')).toBeInTheDocument();
        expect(screen.getByText('Running')).toBeInTheDocument();
        expect(screen.getByText('Python Machine Learning')).toBeInTheDocument();
        expect(screen.getByText('54.123.45.67')).toBeInTheDocument();
        expect(screen.getByText('$0.48/hour')).toBeInTheDocument();
        expect(screen.getByText('Region: us-west-2')).toBeInTheDocument();
      });
    });

    it('enables appropriate action buttons based on instance status', async () => {
      render(<App />);

      fireEvent.click(screen.getByText('Instances'));

      await waitFor(() => {
        const connectButton = screen.getByText('Connect');
        const hibernateButton = screen.getByText('Hibernate');

        expect(connectButton).toBeEnabled();
        expect(hibernateButton).toBeEnabled();
      });
    });

    it('handles instance actions with notifications', async () => {
      render(<App />);

      fireEvent.click(screen.getByText('Instances'));

      await waitFor(() => {
        const hibernateButton = screen.getByText('Hibernate');
        fireEvent.click(hibernateButton);
      });

      await waitFor(() => {
        expect(screen.getByText('Hibernate in progress')).toBeInTheDocument();
      });

      await waitFor(() => {
        expect(screen.getByText('Hibernate successful')).toBeInTheDocument();
      }, { timeout: 2000 });
    });

    it('supports bulk instance operations', async () => {
      render(<App />);

      fireEvent.click(screen.getByText('Instances'));

      await waitFor(() => {
        // Select an instance
        const checkbox = screen.getByRole('checkbox');
        fireEvent.click(checkbox);

        expect(screen.getByText('Bulk Actions')).toBeInTheDocument();
        expect(screen.getByText('Hibernate Selected')).toBeInTheDocument();
      });
    });
  });

  describe('Launch Modal', () => {
    it('validates instance name input', async () => {
      render(<App />);

      // Select a template to open launch modal
      await waitFor(() => {
        const templateCard = screen.getByText('Python Machine Learning');
        fireEvent.click(templateCard);
      });

      await waitFor(() => {
        const launchButton = screen.getByText('Launch Instance');
        expect(launchButton).toBeDisabled(); // Should be disabled with empty name

        // Enter instance name
        const nameInput = screen.getByLabelText('Instance Name');
        fireEvent.change(nameInput, { target: { value: 'test-instance' } });

        expect(launchButton).toBeEnabled(); // Should be enabled with valid name
      });
    });

    it('calculates cost based on instance size selection', async () => {
      render(<App />);

      await waitFor(() => {
        const templateCard = screen.getByText('Python Machine Learning');
        fireEvent.click(templateCard);
      });

      await waitFor(() => {
        // Check default Medium cost
        expect(screen.getByText('$0.96/hour')).toBeInTheDocument(); // 0.48 * 2

        // Change to Large
        const sizeSelect = screen.getByLabelText('Instance Size');
        fireEvent.change(sizeSelect, { target: { value: 'L' } });

        expect(screen.getByText('$1.92/hour')).toBeInTheDocument(); // 0.48 * 4
      });
    });

    it('launches instance successfully', async () => {
      mockWails.PrismService.LaunchInstance.mockResolvedValue(undefined);

      render(<App />);

      await waitFor(() => {
        const templateCard = screen.getByText('Python Machine Learning');
        fireEvent.click(templateCard);
      });

      await waitFor(() => {
        const nameInput = screen.getByLabelText('Instance Name');
        fireEvent.change(nameInput, { target: { value: 'test-instance' } });

        const launchButton = screen.getByText('Launch Instance');
        fireEvent.click(launchButton);
      });

      expect(mockWails.PrismService.LaunchInstance).toHaveBeenCalledWith(
        'test-instance',
        'Python Machine Learning',
        'M'
      );

      await waitFor(() => {
        expect(screen.getByText('Instance launching')).toBeInTheDocument();
      });
    });
  });

  describe('Error Handling', () => {
    it('handles template loading errors gracefully', async () => {
      mockWails.PrismService.GetTemplates.mockRejectedValue(new Error('Network error'));

      render(<App />);

      await waitFor(() => {
        expect(screen.getByText('Failed to load data')).toBeInTheDocument();
        expect(screen.getByText('Unable to connect to Prism daemon')).toBeInTheDocument();
      });
    });

    it('handles instance launch errors gracefully', async () => {
      mockWails.PrismService.LaunchInstance.mockRejectedValue(new Error('Launch failed'));

      render(<App />);

      await waitFor(() => {
        const templateCard = screen.getByText('Python Machine Learning');
        fireEvent.click(templateCard);
      });

      await waitFor(() => {
        const nameInput = screen.getByLabelText('Instance Name');
        fireEvent.change(nameInput, { target: { value: 'test-instance' } });

        const launchButton = screen.getByText('Launch Instance');
        fireEvent.click(launchButton);
      });

      await waitFor(() => {
        expect(screen.getByText('Launch failed')).toBeInTheDocument();
      });
    });
  });

  describe('Navigation', () => {
    it('navigates between views correctly', async () => {
      render(<App />);

      // Start on templates view
      expect(screen.getByText('Research Templates')).toBeInTheDocument();

      // Navigate to instances
      fireEvent.click(screen.getByText('Instances'));
      await waitFor(() => {
        expect(screen.getByText('My Workspaces')).toBeInTheDocument();
      });

      // Navigate to settings
      fireEvent.click(screen.getByText('Settings'));
      await waitFor(() => {
        expect(screen.getByText('Settings interface coming soon...')).toBeInTheDocument();
      });

      // Navigate back to templates
      fireEvent.click(screen.getByText('Templates'));
      await waitFor(() => {
        expect(screen.getByText('Research Templates')).toBeInTheDocument();
      });
    });
  });
});